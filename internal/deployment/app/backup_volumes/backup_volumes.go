package backup_volumes

import (
	"context"
	"errors"

	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

type (
	// Copies the volumes of a requested backup into an archive. A failure is
	// recorded on the backup and not retried: the person can retry from the
	// dashboard once the cause is fixed.
	Command struct {
		bus.Command[bus.UnitType]

		ID string `json:"id"`
	}

	Options interface {
		BackupHelperImage() string
		CheckshipVersion() string
	}

	// Runs a backup from start to end. Shared by the backup job and the jobs that
	// back up before they delete or replace data.
	Runner struct {
		targets  domain.TargetsReader
		reader   domain.BackupsReader
		writer   domain.BackupsWriter
		provider domain.Provider
		files    domain.BackupFiles
		options  Options
		logger   log.Logger
	}
)

func (Command) Name_() string        { return "deployment.command.backup_volumes" }
func (c Command) ResourceID() string { return c.ID }

func NewRunner(
	targets domain.TargetsReader,
	reader domain.BackupsReader,
	writer domain.BackupsWriter,
	provider domain.Provider,
	files domain.BackupFiles,
	options Options,
	logger log.Logger,
) *Runner {
	return &Runner{targets, reader, writer, provider, files, options, logger}
}

func Handler(reader domain.BackupsReader, runner *Runner) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		backup, err := reader.GetByID(ctx, domain.BackupID(cmd.ID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		if s := backup.Status(); s == domain.BackupStatusSucceeded || s == domain.BackupStatusFailed {
			return bus.Unit, nil // Already done, the job came back after a restart
		}

		// The error is on the backup, only a failure to record it is worth a retry
		_, err = runner.Run(ctx, backup)

		if errors.Is(err, ErrRecordFailed) {
			return bus.Unit, err
		}

		return bus.Unit, nil
	}
}

// Queue the copy of a failed backup asked to run again from the dashboard.
func OnBackupRetriedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.BackupRetried] {
	return func(ctx context.Context, evt domain.BackupRetried) error {
		if evt.Inline {
			return nil
		}

		return scheduler.Queue(ctx, Command{ID: string(evt.ID)},
			bus.WithGroup(app.EnvironmentGroup(domain.ProjectName(evt.AppName, evt.Environment, evt.App))),
			bus.WithPolicy(bus.JobPolicyRetryPreserveOrder|bus.JobPolicyCancellable),
		)
	}
}

// Queue the copy of a backup requested from the dashboard or the API. The other
// triggers run their backup inline, in the job that needs it.
func OnBackupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.BackupRequested] {
	return func(ctx context.Context, evt domain.BackupRequested) error {
		if evt.Trigger != domain.BackupTriggerManual {
			return nil
		}

		return scheduler.Queue(ctx, Command{ID: string(evt.ID)},
			bus.WithGroup(app.EnvironmentGroup(domain.ProjectName(evt.AppName, evt.Environment, evt.App))),
			bus.WithPolicy(bus.JobPolicyRetryPreserveOrder|bus.JobPolicyCancellable),
		)
	}
}

// Returned by Run when the outcome of the backup could not be saved.
var ErrRecordFailed = errors.New("backup_record_failed")

// Copies the volumes, stores the archive and records the outcome on the backup,
// which may be new (it is saved first) or loaded from the store. The returned
// backup is the saved one; the error is the reason it failed, if it did.
func (r *Runner) Run(ctx context.Context, backup domain.Backup) (domain.Backup, error) {
	backup, err := r.save(ctx, backup)

	if err != nil {
		return backup, err
	}

	if err = backup.Started(); err != nil {
		return backup, err
	}

	if backup, err = r.save(ctx, backup); err != nil {
		return backup, err
	}

	result, local, stored, storeErr, err := r.copy(ctx, backup)

	if err != nil {
		r.logger.Errorw("volume backup failed", "backup", backup.ID(), "app", backup.App(), "environment", backup.Environment(), "error", err)
		backup.Failed(err)
	} else {
		if storeErr != nil {
			r.logger.Warnw("backup kept in the data folder only", "backup", backup.ID(), "error", storeErr)
		}

		backup.Succeeded(result, local, stored, storeErr)
	}

	saved, werr := r.save(ctx, backup)

	if werr != nil {
		return backup, errors.Join(werr, err)
	}

	return saved, err
}

// Writes the pending changes and reads the backup back, so the next write only
// carries what happens after this one.
func (r *Runner) save(ctx context.Context, backup domain.Backup) (domain.Backup, error) {
	if err := r.writer.Write(ctx, &backup); err != nil {
		return backup, errors.Join(ErrRecordFailed, err)
	}

	saved, err := r.reader.GetByID(ctx, backup.ID())

	if err != nil {
		return backup, errors.Join(ErrRecordFailed, err)
	}

	return saved, nil
}

func (r *Runner) copy(ctx context.Context, backup domain.Backup) (result domain.BackupResult, local, stored bool, storeErr, err error) {
	target, err := r.targets.GetByID(ctx, backup.Target())

	if err != nil {
		return result, false, false, nil, err
	}

	file, err := r.files.Create(ctx)

	if err != nil {
		return result, false, false, nil, err
	}

	defer file.Abort()

	copied, err := r.provider.BackupVolumes(ctx, target, backup.App(), backup.Environment(), domain.BackupVolumesOptions{
		Manifest: volumearchive.Manifest{
			Checkship:   r.options.CheckshipVersion(),
			CreatedAt:   backup.Requested().At(),
			AppID:       string(backup.App()),
			AppName:     string(backup.AppName()),
			Environment: string(backup.Environment()),
			TargetID:    string(backup.Target()),
		},
		Volumes:        backup.Volumes(),
		StopContainers: backup.StopContainers(),
		HelperImage:    r.options.BackupHelperImage(),
	}, file)

	if err != nil {
		return result, false, false, nil, err
	}

	if err = file.Commit(backup); err != nil {
		return result, false, false, nil, err
	}

	result = domain.BackupResult{
		Size:    copied.Summary.Size,
		SHA256:  copied.Summary.SHA256,
		Volumes: copied.Summary.Volumes,
		Skipped: copied.Manifest.Skipped,
	}

	store, storeErr := r.files.Store(ctx, backup)
	local = true

	if store.Stored && !store.KeepLocal {
		if rmErr := r.files.RemoveLocal(ctx, backup); rmErr == nil {
			local = false
		} else {
			r.logger.Warnw("could not remove the local copy of a stored backup", "backup", backup.ID(), "error", rmErr)
		}
	}

	return result, local, store.Stored, storeErr, nil
}
