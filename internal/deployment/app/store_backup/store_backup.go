package store_backup

import (
	"context"
	"errors"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/log"
)

type (
	// Ask for a backup kept in the data folder only to be copied to the storage again.
	Request struct {
		bus.Command[bus.UnitType]

		ID string `json:"-"`
	}

	// Copies the local archive of a backup to the storage.
	Command struct {
		bus.Command[bus.UnitType]

		ID string `json:"id"`
	}
)

func (Request) Name_() string        { return "deployment.command.request_backup_store" }
func (Command) Name_() string        { return "deployment.command.store_backup" }
func (c Command) ResourceID() string { return c.ID }

func RequestHandler(reader domain.BackupsReader, scheduler bus.Scheduler) bus.RequestHandler[bus.UnitType, Request] {
	return func(ctx context.Context, req Request) (bus.UnitType, error) {
		backup, err := reader.GetByID(ctx, domain.BackupID(req.ID))

		if err != nil {
			return bus.Unit, err
		}

		if !backup.Local() || backup.Status() != domain.BackupStatusSucceeded {
			return bus.Unit, domain.ErrBackupNotFinished
		}

		return bus.Unit, scheduler.Queue(ctx, Command{ID: req.ID}, bus.WithPolicy(bus.JobPolicyMerge|bus.JobPolicyCancellable))
	}
}

func Handler(
	reader domain.BackupsReader,
	writer domain.BackupsWriter,
	files domain.BackupFiles,
	logger log.Logger,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		backup, err := reader.GetByID(ctx, domain.BackupID(cmd.ID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		if !backup.Local() || backup.Status() != domain.BackupStatusSucceeded {
			return bus.Unit, nil
		}

		store, storeErr := files.Store(ctx, backup)
		local := true

		if store.Stored && !store.KeepLocal {
			if err = files.RemoveLocal(ctx, backup); err == nil {
				local = false
			} else {
				logger.Warnw("could not remove the local copy of a stored backup", "backup", backup.ID(), "error", err)
			}
		}

		if storeErr != nil {
			logger.Warnw("backup still kept in the data folder only", "backup", backup.ID(), "error", storeErr)
		}

		if err = backup.HasCopies(local, store.Stored || backup.Stored(), storeErr); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &backup)
	}
}
