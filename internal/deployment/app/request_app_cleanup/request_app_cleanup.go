package request_app_cleanup

import (
	"context"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

// Mark the application for deletion.
type Command struct {
	bus.Command[bus.UnitType]

	ID            string            `json:"-"`
	BackupVolumes monad.Maybe[bool] `json:"backup_volumes"` // Back up the volumes before removing them, true if not set
}

func (Command) Name_() string { return "deployment.command.request_app_cleanup" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return bus.Unit, err
		}

		app.RequestCleanupWithBackup(auth.CurrentUser(ctx).MustGet(), cmd.BackupVolumes.Get(true))

		return bus.Unit, writer.Write(ctx, &app)
	}
}
