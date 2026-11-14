package sqlite

import (
	"context"
	"embed"
	"time"

	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/flag"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

var (
	//go:embed migrations/*.sql
	migrations embed.FS

	Migrations = sqlite.NewMigrationsModule("scheduler", "migrations", migrations)
)

type (
	job struct {
		id     string
		msg    bus.Request
		policy bus.JobPolicy
	}

	jobQuery struct {
		JobID       string              `json:"id"`
		ResourceID  string              `json:"resource_id"`
		Group       string              `json:"group"`
		MessageName string              `json:"message_name"`
		MessageData string              `json:"message_data"`
		QueuedAt    time.Time           `json:"queued_at"`
		NotBefore   time.Time           `json:"not_before"`
		ErrorCode   monad.Maybe[string] `json:"error_code"`
		JobPolicy   bus.JobPolicy       `json:"policy"`
		Retrieved   bool                `json:"retrieved"`
	}

	store struct {
		db *sqlite.Database
	}
)

func (j *job) ID() string            { return j.id }
func (j *job) Message() bus.Request  { return j.msg }
func (j *job) Policy() bus.JobPolicy { return j.policy }

func (j *jobQuery) ID() string            { return j.JobID }
func (j *jobQuery) Message() bus.Request  { panic("not implemented") } // Should never happen because this is a query only job
func (j *jobQuery) Policy() bus.JobPolicy { return j.JobPolicy }

// Builds a new adapter persisting jobs in the given sqlite database.
// For it to work, commands must be (de)serializable using the bus.Marshallable mapper.
func NewScheduledJobsStore(db *sqlite.Database) bus.ScheduledJobsStore {
	return &store{db}
}

// Setup the scheduler adapter, migrate the database and reset running jobs by marking
// them as not retrieved so they will be picked up next time GetNextPendingJobs is called.
// You MUST call this method at the application startup.
func (s *store) Setup() error {
	if err := s.db.Migrate(Migrations); err != nil {
		return err
	}

	_, err := s.db.ExecContext(context.Background(), `
		UPDATE scheduled_jobs
		SET retrieved = false
		WHERE retrieved = true`)

	return err
}

func (s *store) Create(
	ctx context.Context,
	msg bus.Schedulable,
	options bus.CreateOptions,
) error {
	jobId := id.New[string]()
	now := time.Now().UTC()
	msgValue, err := storage.ValueJSON(msg)

	if err != nil {
		return err
	}

	var (
		msgName    = msg.Name_()
		resourceId = msg.ResourceID()
	)

	// Could not use the ON CONFLICT here :'(
	if flag.IsSet(options.Policy, bus.JobPolicyMerge) {
		result, err := s.db.ExecContext(ctx, `
			UPDATE scheduled_jobs
			SET message_data = ?
			WHERE id = (
				SELECT id
				FROM scheduled_jobs
				WHERE resource_id = ? AND message_name = ? AND retrieved = false
			)`, msgValue, resourceId, msgName)

		if affected, _ := result.RowsAffected(); affected > 0 {
			return err
		}
	}

	return builder.
		Insert("scheduled_jobs", builder.Values{
			"id":           jobId,
			"resource_id":  resourceId,
			"[group]":      options.Group.Get(jobId), // Default to the job id if no group set
			"message_name": msgName,
			"message_data": msgValue,
			"queued_at":    now,
			"not_before":   now,
			"policy":       options.Policy,
			"retrieved":    false,
		}).
		Exec(s.db, ctx)
}

func (s *store) Delete(ctx context.Context, id string) error {
	r, err := s.db.ExecContext(ctx, "DELETE FROM scheduled_jobs WHERE id = ? AND (policy & ?) != 0",
		id, bus.JobPolicyCancellable)

	if err != nil {
		return err
	}

	affected, err := r.RowsAffected()

	if err != nil {
		return err
	}

	if affected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

func (s *store) GetAllJobs(ctx context.Context, filters bus.GetJobsFilters) (storage.Paginated[bus.ScheduledJob], error) {
	return builder.
		Select[bus.ScheduledJob](`
			id
			,resource_id
			,[group]
			,message_name
			,message_data
			,queued_at
			,not_before
			,errcode
			,policy
			,retrieved
		`).
		F("FROM scheduled_jobs WHERE TRUE").
		S(builder.Maybe(filters.Status, func(status bus.JobStatus) (string, []any) {
			switch status {
			case bus.JobStatusRunning:
				return "AND retrieved = true", nil
			case bus.JobStatusFailed:
				return "AND retrieved = false AND errcode IS NOT NULL", nil
			default:
				return "AND retrieved = false AND errcode IS NULL", nil
			}
		})).
		F("ORDER BY queued_at").
		Paginate(s.db, ctx, jobQueryMapper, filters.Page.Get(1), 10)
}

// Claims up to limit jobs, at most one per group, and marks them as retrieved.
// The head of each group is chosen across every message type before filtering on
// messageNames, so a runner never starts a job while an older job of the same group,
// handled by another runner, is still waiting. No messageNames means every type.
//
// The due check uses julianday: the driver stores times as
// "2006-01-02 15:04:05.999999999-07:00", which compares as greater than DATETIME('now')
// for the rest of the second, so a job queued a moment ago would not be found.
// Ordering stays on the raw strings, julianday only keeps milliseconds and two jobs
// queued in the same millisecond would tie.
func (s *store) GetNextPendingJobs(ctx context.Context, limit int, messageNames ...string) ([]bus.ScheduledJob, error) {
	if limit <= 0 {
		return nil, nil
	}

	// A single UPDATE ... RETURNING is atomic, so two runners can't claim the same job.
	return builder.
		Query[bus.ScheduledJob](`
			UPDATE scheduled_jobs
			SET retrieved = true
			WHERE id IN (
				SELECT id FROM (
					SELECT id, message_name, MIN(not_before) AS not_before FROM scheduled_jobs sj
					WHERE
						sj.retrieved = false
						AND julianday(sj.not_before) <= julianday('now')
						AND sj.[group] NOT IN (SELECT DISTINCT [group] FROM scheduled_jobs WHERE retrieved = true)
						AND (sj.policy & ? = 0 OR (SELECT COUNT(resource_id) FROM scheduled_jobs WHERE resource_id = sj.resource_id) <= 1)
					GROUP BY sj.[group]
				) heads
				WHERE TRUE`, bus.JobPolicyWaitForOthersResourceID).
		S(builder.Array("AND heads.message_name IN", messageNames)).
		F(`
				ORDER BY heads.not_before
				LIMIT ?
			)
			RETURNING id, message_name, message_data, policy`, limit).
		All(s.db, ctx, jobMapper)
}

// Gives claimed jobs back to the queue, used when a runner stops before a worker
// picked them up.
func (s *store) Release(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	return builder.
		Command("UPDATE scheduled_jobs SET retrieved = false WHERE TRUE").
		S(builder.Array("AND id IN", ids)).
		Exec(s.db, ctx)
}

func (s *store) AfterCommit(ctx context.Context, fn func()) { sqlite.AfterCommit(ctx, fn) }

func (s *store) Retry(ctx context.Context, j bus.ScheduledJob, jobErr error) error {
	// If we don't need to preserve the order of related tasks, we simply update the job to queue it again
	// in the future.
	if !flag.IsSet(j.Policy(), bus.JobPolicyRetryPreserveOrder) {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE scheduled_jobs
			SET
				errcode = ?
				,not_before = DATETIME('now', '+15 seconds')
				,retrieved = false
			WHERE id = ?`, jobErr.Error(), j.ID(),
		); err != nil {
			return err
		}
	}

	// If instead, we want all jobs sharing the same dedupe_name to be updated all at once,
	// we should make sure to set all of them in the future by a specific amount to preserve
	// the job order.
	_, err := s.db.ExecContext(ctx, `
		UPDATE scheduled_jobs
		SET
			errcode = v.errcode
			,not_before = v.updated_date
			,retrieved = false
		FROM (
			SELECT
				id
				,CASE WHEN id = ? THEN ? ELSE errcode END AS errcode
				,DATETIME('now', '+' || CAST(14 + 1 * ROW_NUMBER() OVER (ORDER BY not_before) AS TEXT) || ' seconds') AS updated_date
			FROM scheduled_jobs
			WHERE [group] = (SELECT [group] FROM scheduled_jobs WHERE id = ?)
		) v
		WHERE scheduled_jobs.id = v.id`, j.ID(), jobErr.Error(), j.ID())

	return err
}

func (s *store) Done(ctx context.Context, j bus.ScheduledJob) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM scheduled_jobs WHERE id = ?", j.ID())
	return err
}

func jobMapper(scanner storage.Scanner) (bus.ScheduledJob, error) {
	var (
		j       job
		msgName string
		msgData string
	)

	err := scanner.Scan(
		&j.id,
		&msgName,
		&msgData,
		&j.policy,
	)

	if err != nil {
		return &j, err
	}

	j.msg, err = bus.Marshallable.From(msgName, msgData)

	return &j, err
}

func jobQueryMapper(scanner storage.Scanner) (bus.ScheduledJob, error) {
	var j jobQuery

	err := scanner.Scan(
		&j.JobID,
		&j.ResourceID,
		&j.Group,
		&j.MessageName,
		&j.MessageData,
		&j.QueuedAt,
		&j.NotBefore,
		&j.ErrorCode,
		&j.JobPolicy,
		&j.Retrieved,
	)

	return &j, err
}
