package cleanup_app

import (
	"context"
	"errors"
	"slices"
	"time"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/backup_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	shared "github.com/checkship-deploy/checkship/pkg/domain"
)

// Check if the cleanup is needed for a specific app, environment and a specific target.
// It will be skipped if the target is being deleted or if no successful deployment has been made
// in the interval represented by the `From` and `To` parameters.
//
// When Backup is set, the volumes are backed up first and the cleanup only goes on
// once that backup succeeded: a failure makes the job retry, and nothing is removed.
type Command struct {
	bus.Command[bus.UnitType]

	AppID       string    `json:"app_id"`
	TargetID    string    `json:"target_id"`
	Environment string    `json:"environment"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	Backup      bool      `json:"backup,omitempty"`
	Trigger     string    `json:"trigger,omitempty"`      // before_delete or before_target_change
	MoveVolumes bool      `json:"move_volumes,omitempty"` // Restore the backup where the environment runs now
	RequestedBy string    `json:"requested_by,omitempty"` // Author of the backup, the app creator if empty
}

func (Command) Name_() string        { return "deployment.command.cleanup_app" }
func (c Command) ResourceID() string { return c.AppID }

// What a backup before cleanup needs. Nil when backups are not wired (tests of
// the cleanup alone).
type Backups struct {
	Apps     domain.AppsReader
	Reader   domain.BackupsReader
	Restores domain.RestoresStore
	Runner   *backup_volumes.Runner
}

func Handler(
	reader domain.TargetsReader,
	deploymentsReader domain.DeploymentsReader,
	provider domain.Provider,
	backups ...Backups,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.TargetID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		var (
			appid            = domain.AppID(cmd.AppID)
			env              = domain.Environment(cmd.Environment)
			interval         shared.TimeInterval
			runningOrPending domain.HasRunningOrPendingDeploymentsOnAppTargetEnv
			successful       domain.HasSuccessfulDeploymentsOnAppTargetEnv
		)

		if interval, err = shared.NewTimeInterval(cmd.From, cmd.To); err != nil {
			return bus.Unit, err
		}

		if runningOrPending, successful, err = deploymentsReader.HasDeploymentsOnAppTargetEnv(ctx, appid, target.ID(), env, interval); err != nil {
			return bus.Unit, err
		}

		strategy, err := target.AppCleanupStrategy(runningOrPending, successful)

		if err != nil {
			return bus.Unit, err
		}

		if strategy == domain.CleanupStrategyDefault && cmd.Backup && len(backups) > 0 {
			if err = backupFirst(ctx, backups[0], provider, target, cmd); err != nil {
				return bus.Unit, err
			}
		}

		return bus.Unit, provider.Cleanup(ctx, appid, target, env, strategy)
	}
}

func backupFirst(ctx context.Context, deps Backups, provider domain.Provider, target domain.Target, cmd Command) error {
	appid := domain.AppID(cmd.AppID)
	env := domain.Environment(cmd.Environment)

	app, err := deps.Apps.GetByID(ctx, appid)

	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil // Nothing left to name a backup after
		}

		return err
	}

	volumes, _, err := provider.Volumes(ctx, target, appid, env)

	if err != nil {
		return err
	}

	if len(volumes) == 0 {
		return nil
	}

	trigger, err := domain.BackupTriggerFrom(cmd.Trigger)

	if err != nil {
		trigger = domain.BackupTriggerBeforeDelete
	}

	uid := auth.UserID(cmd.RequestedBy)

	if uid == "" {
		uid = app.CreatedBy()
	}

	backup, found, err := previousAttempt(ctx, deps.Reader, cmd, target.ID(), trigger)

	if err != nil {
		return err
	}

	switch {
	case !found:
		backup, err = domain.NewBackupOnTarget(app, env, target.ID(), trigger, nil, true, uid)

		// The app was deleted right after its target changed: it is still a backup before deletion
		if errors.Is(err, domain.ErrAppCleanupRequested) {
			backup, err = domain.NewBackupOnTarget(app, env, target.ID(), domain.BackupTriggerBeforeDelete, nil, true, uid)
		}
	case backup.Status() == domain.BackupStatusFailed:
		err = backup.Retry(true)
	}

	if err != nil {
		return err
	}

	if backup.Status() != domain.BackupStatusSucceeded {
		if backup, err = deps.Runner.Run(ctx, backup); err != nil {
			return err
		}
	}

	if !cmd.MoveVolumes {
		return nil
	}

	restores, err := deps.Restores.GetByBackup(ctx, backup.ID())

	if err != nil || len(restores) > 0 {
		return err // Already queued by a previous attempt
	}

	// The environment now points at its new target: the restore goes there, after the redeploy
	restore, err := domain.NewRestore(backup, app, env, nil, false, uid)

	if errors.Is(err, domain.ErrAppCleanupRequested) {
		return nil // Deleted meanwhile, the backup is kept
	}

	if err != nil {
		return err
	}

	return deps.Restores.Write(ctx, &restore)
}

// The backup a previous run of this job made, so a retry does not pile up new ones.
func previousAttempt(ctx context.Context, reader domain.BackupsReader, cmd Command, target domain.TargetID, trigger domain.BackupTrigger) (domain.Backup, bool, error) {
	all, err := reader.GetByApp(ctx, domain.AppID(cmd.AppID))

	if err != nil {
		return domain.Backup{}, false, err
	}

	idx := slices.IndexFunc(all, func(b domain.Backup) bool {
		return b.Environment() == domain.Environment(cmd.Environment) &&
			b.Target() == target &&
			(b.Trigger() == trigger || b.Trigger() == domain.BackupTriggerBeforeDelete) &&
			!b.Requested().At().Before(cmd.To)
	})

	if idx < 0 {
		return domain.Backup{}, false, nil
	}

	return all[idx], true, nil
}
