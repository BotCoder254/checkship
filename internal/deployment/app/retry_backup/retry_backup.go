package retry_backup

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Run a failed backup again, once its cause is fixed.
type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"-"`
}

func (Command) Name_() string { return "deployment.command.retry_backup" }

func Handler(reader domain.BackupsReader, writer domain.BackupsWriter) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		backup, err := reader.GetByID(ctx, domain.BackupID(cmd.ID))

		if err != nil {
			return bus.Unit, err
		}

		if err = backup.Retry(false); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &backup)
	}
}
