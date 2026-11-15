package request_restore

import (
	"context"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/restore_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

// Put the volumes of a backup back, into the environment it came from or into
// another environment or application.
type Command struct {
	bus.Command[string]

	BackupID      string              `json:"-"`
	AppID         monad.Maybe[string] `json:"app_id"`         // The backup's application if not set
	Environment   monad.Maybe[string] `json:"environment"`    // The backup's environment if not set
	Volumes       []string            `json:"volumes"`        // Every volume the destination declares if empty
	SnapshotFirst monad.Maybe[bool]   `json:"snapshot_first"` // Back up the current data first, true if not set
}

func (Command) Name_() string { return "deployment.command.request_restore" }

func Handler(
	backups domain.BackupsReader,
	apps domain.AppsReader,
	writer domain.RestoresWriter,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		backup, err := backups.GetByID(ctx, domain.BackupID(cmd.BackupID))

		if err != nil {
			return "", err
		}

		destination, err := apps.GetByID(ctx, domain.AppID(cmd.AppID.Get(string(backup.App()))))

		if err != nil {
			return "", err
		}

		env := domain.Environment(cmd.Environment.Get(string(backup.Environment())))

		restore, err := domain.NewRestore(backup, destination, env, cmd.Volumes, cmd.SnapshotFirst.Get(true), auth.CurrentUser(ctx).MustGet())

		if err != nil {
			return "", err
		}

		if err = writer.Write(ctx, &restore); err != nil {
			return "", err
		}

		return string(restore.ID()), nil
	}
}

// Queue the restore in the group of the destination's deployments, so it waits
// for a deployment in progress (the redeploy of a target change for example).
func OnRestoreRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.RestoreRequested] {
	return func(ctx context.Context, evt domain.RestoreRequested) error {
		return scheduler.Queue(ctx, restore_volumes.Command{ID: string(evt.ID)},
			bus.WithGroup(app.EnvironmentGroup(domain.ProjectName(evt.AppName, evt.Environment, evt.App))),
			bus.WithPolicy(bus.JobPolicyRetryPreserveOrder|bus.JobPolicyCancellable),
		)
	}
}
