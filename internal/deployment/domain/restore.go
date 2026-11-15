package domain

import (
	"context"
	"slices"
	"time"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	shared "github.com/checkship-deploy/checkship/pkg/domain"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

const (
	RestoreStatusPending   RestoreStatus = "pending"
	RestoreStatusRunning   RestoreStatus = "running"
	RestoreStatusSucceeded RestoreStatus = "succeeded"
	RestoreStatusFailed    RestoreStatus = "failed"
)

type (
	RestoreID     string
	RestoreStatus string

	// Puts the volumes of a backup back into an application environment. Kept so
	// the outcome stays visible once its job is gone: a restore is never retried
	// on its own, since each attempt stops the application.
	Restore struct {
		event.Emitter

		id            RestoreID
		backup        BackupID
		app           AppID
		environment   Environment
		target        TargetID
		volumes       VolumeKeys // Asked for, every volume the destination declares if empty
		snapshotFirst bool       // Back up the destination before replacing its volumes
		status        RestoreStatus
		restored      VolumeKeys
		snapshot      monad.Maybe[BackupID]
		errorCode     monad.Maybe[string]
		requested     shared.Action[auth.UserID]
		finishedAt    monad.Maybe[time.Time]
	}

	RestoresReader interface {
		GetByID(context.Context, RestoreID) (Restore, error)
		GetByBackup(context.Context, BackupID) ([]Restore, error)
	}

	RestoresWriter interface {
		Write(context.Context, ...*Restore) error
	}

	RestoresStore interface {
		RestoresReader
		RestoresWriter
	}

	RestoreRequested struct {
		bus.Notification

		ID            RestoreID
		Backup        BackupID
		App           AppID
		AppName       AppName
		Environment   Environment
		Target        TargetID
		Volumes       VolumeKeys
		SnapshotFirst bool
		Requested     shared.Action[auth.UserID]
	}

	RestoreStarted struct {
		bus.Notification

		ID RestoreID
	}

	RestoreSucceeded struct {
		bus.Notification

		ID         RestoreID
		App        AppID
		Restored   VolumeKeys
		Snapshot   monad.Maybe[BackupID]
		FinishedAt time.Time
	}

	RestoreFailed struct {
		bus.Notification

		ID         RestoreID
		App        AppID
		ErrorCode  string
		Snapshot   monad.Maybe[BackupID]
		FinishedAt time.Time
	}
)

func (RestoreRequested) Name_() string { return "deployment.event.restore_requested" }
func (RestoreStarted) Name_() string   { return "deployment.event.restore_started" }
func (RestoreSucceeded) Name_() string { return "deployment.event.restore_succeeded" }
func (RestoreFailed) Name_() string    { return "deployment.event.restore_failed" }

// Requests the restore of a backup into an application environment, which may
// be another environment or another application than the one it came from.
func NewRestore(
	backup Backup,
	app App,
	env Environment,
	volumes []string,
	snapshotFirst bool,
	uid auth.UserID,
) (r Restore, err error) {
	result, err := backup.Restorable()

	if err != nil {
		return r, err
	}

	if app.cleanupRequested.HasValue() {
		return r, ErrAppCleanupRequested
	}

	if _, err = EnvironmentFrom(string(env)); err != nil {
		return r, err
	}

	for _, key := range volumes {
		if !slices.ContainsFunc(result.Volumes, func(v volumearchive.VolumeStats) bool { return v.Key == key }) {
			return r, ErrVolumeNotFound
		}
	}

	r.apply(RestoreRequested{
		ID:            id.New[RestoreID](),
		Backup:        backup.id,
		App:           app.id,
		AppName:       app.name,
		Environment:   env,
		Target:        app.Config(env).Target(),
		Volumes:       volumes,
		SnapshotFirst: snapshotFirst,
		Requested:     shared.NewAction(uid),
	})

	return r, nil
}

// Recreates a restore from the persistent storage.
func RestoreFrom(scanner storage.Scanner) (r Restore, err error) {
	var (
		snapshot    monad.Maybe[string] // Maybe cannot scan into a named string type
		requestedAt time.Time
		requestedBy auth.UserID
	)

	err = scanner.Scan(
		&r.id,
		&r.backup,
		&r.app,
		&r.environment,
		&r.target,
		&r.volumes,
		&r.snapshotFirst,
		&r.status,
		&r.restored,
		&snapshot,
		&r.errorCode,
		&requestedAt,
		&requestedBy,
		&r.finishedAt,
	)

	r.requested = shared.ActionFrom(requestedBy, requestedAt)

	if id, isSet := snapshot.TryGet(); isSet {
		r.snapshot.Set(BackupID(id))
	}

	return r, err
}

func (r *Restore) Started() {
	if r.status != RestoreStatusPending {
		return
	}

	r.apply(RestoreStarted{ID: r.id})
}

func (r *Restore) Succeeded(restored []string, snapshot monad.Maybe[BackupID]) {
	r.apply(RestoreSucceeded{
		ID:         r.id,
		App:        r.app,
		Restored:   restored,
		Snapshot:   snapshot,
		FinishedAt: time.Now().UTC(),
	})
}

func (r *Restore) Failed(err error, snapshot monad.Maybe[BackupID]) {
	r.apply(RestoreFailed{
		ID:         r.id,
		App:        r.app,
		ErrorCode:  errorCode(err),
		Snapshot:   snapshot,
		FinishedAt: time.Now().UTC(),
	})
}

// Finished restores are not run again by a job that comes back after a restart.
func (r *Restore) IsFinished() bool {
	return r.status == RestoreStatusSucceeded || r.status == RestoreStatusFailed
}

func (r *Restore) ID() RestoreID                         { return r.id }
func (r *Restore) Backup() BackupID                      { return r.backup }
func (r *Restore) App() AppID                            { return r.app }
func (r *Restore) Environment() Environment              { return r.environment }
func (r *Restore) Target() TargetID                      { return r.target }
func (r *Restore) Volumes() []string                     { return r.volumes }
func (r *Restore) SnapshotFirst() bool                   { return r.snapshotFirst }
func (r *Restore) Status() RestoreStatus                 { return r.status }
func (r *Restore) Restored() []string                    { return r.restored }
func (r *Restore) Snapshot() monad.Maybe[BackupID]       { return r.snapshot }
func (r *Restore) ErrorCode() monad.Maybe[string]        { return r.errorCode }
func (r *Restore) Requested() shared.Action[auth.UserID] { return r.requested }

func (r *Restore) apply(e event.Event) {
	switch v := e.(type) {
	case RestoreRequested:
		r.id = v.ID
		r.backup = v.Backup
		r.app = v.App
		r.environment = v.Environment
		r.target = v.Target
		r.volumes = v.Volumes
		r.snapshotFirst = v.SnapshotFirst
		r.requested = v.Requested
		r.status = RestoreStatusPending
	case RestoreStarted:
		r.status = RestoreStatusRunning
	case RestoreSucceeded:
		r.status = RestoreStatusSucceeded
		r.restored = v.Restored
		r.snapshot = v.Snapshot
		r.finishedAt.Set(v.FinishedAt)
	case RestoreFailed:
		r.status = RestoreStatusFailed
		r.errorCode.Set(v.ErrorCode)
		r.snapshot = v.Snapshot
		r.finishedAt.Set(v.FinishedAt)
	}

	event.Store(r, e)
}
