package delete_backup

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Delete a backup and every copy of its archive.
type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"-"`
}

func (Command) Name_() string { return "deployment.command.delete_backup" }

func Handler(
	reader domain.BackupsReader,
	writer domain.BackupsWriter,
	files domain.BackupFiles,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		backup, err := reader.GetByID(ctx, domain.BackupID(cmd.ID))

		if err != nil {
			return bus.Unit, err
		}

		if err = backup.Delete(); err != nil {
			return bus.Unit, err
		}

		// Files first: if the storage is down the record stays, so nothing is left
		// in a bucket without a way to find it again.
		if err = files.Delete(ctx, backup); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &backup)
	}
}
