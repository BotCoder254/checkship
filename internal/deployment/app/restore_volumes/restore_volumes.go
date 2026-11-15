package restore_volumes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"github.com/checkship-deploy/checkship/internal/deployment/app/backup_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

type (
	// Replaces the volumes of an application environment with a backup. The
	// outcome is recorded on the restore and never retried automatically: each
	// attempt stops the application.
	Command struct {
		bus.Command[bus.UnitType]

		ID string `json:"id"`
	}

	Options interface {
		BackupHelperImage() string
	}
)

func (Command) Name_() string        { return "deployment.command.restore_volumes" }
func (c Command) ResourceID() string { return c.ID }

func Handler(
	restores domain.RestoresReader,
	restoresWriter domain.RestoresWriter,
	backups domain.BackupsReader,
	apps domain.AppsReader,
	targets domain.TargetsReader,
	files domain.BackupFiles,
	provider domain.Provider,
	runner *backup_volumes.Runner,
	options Options,
	logger log.Logger,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		restore, err := restores.GetByID(ctx, domain.RestoreID(cmd.ID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		if restore.IsFinished() {
			return bus.Unit, nil
		}

		restore.Started()

		if err = restoresWriter.Write(ctx, &restore); err != nil {
			return bus.Unit, err
		}

		var snapshot monad.Maybe[domain.BackupID]

		restored, err := func() ([]string, error) {
			backup, err := backups.GetByID(ctx, restore.Backup())

			if err != nil {
				return nil, err
			}

			result, err := backup.Restorable()

			if err != nil {
				return nil, err
			}

			app, err := apps.GetByID(ctx, restore.App())

			if err != nil {
				return nil, err
			}

			env := restore.Environment()
			// The environment may have moved since the request: restore where it runs now.
			target, err := targets.GetByID(ctx, app.Config(env).Target())

			if err != nil {
				return nil, err
			}

			projectName, err := app.ProjectName(env)

			if err != nil {
				return nil, err
			}

			if err = verify(ctx, files, backup, result.SHA256); err != nil {
				return nil, err
			}

			if restore.SnapshotFirst() {
				id, err := takeSnapshot(ctx, provider, runner, target, app, restore)

				if id != "" {
					snapshot.Set(id)
				}

				if err != nil {
					return nil, err // Never replace data that could not be saved first
				}
			}

			archive, _, err := files.Open(ctx, backup)

			if err != nil {
				return nil, err
			}

			defer archive.Close()

			return provider.RestoreVolumes(ctx, target, app.ID(), env, domain.RestoreVolumesOptions{
				ProjectName: projectName,
				Volumes:     restore.Volumes(),
				HelperImage: options.BackupHelperImage(),
			}, archive)
		}()

		if err != nil {
			logger.Errorw("restore failed", "restore", restore.ID(), "backup", restore.Backup(), "app", restore.App(), "environment", restore.Environment(), "error", err)
			restore.Failed(err, snapshot)
		} else {
			restore.Succeeded(restored, snapshot)
		}

		return bus.Unit, restoresWriter.Write(ctx, &restore)
	}
}

// Reads the whole archive and compares its digest with the one recorded when
// the backup was made, so a damaged archive never reaches the volumes.
func verify(ctx context.Context, files domain.BackupFiles, backup domain.Backup, expected string) error {
	rc, _, err := files.Open(ctx, backup)

	if err != nil {
		return err
	}

	defer rc.Close()

	h := sha256.New()

	if _, err = io.Copy(h, rc); err != nil {
		return err
	}

	if hex.EncodeToString(h.Sum(nil)) != expected {
		return domain.ErrBackupChecksum
	}

	return nil
}

// Backs up what the destination has now. Returns no ID when there is nothing
// to save (an environment that was never deployed).
func takeSnapshot(
	ctx context.Context,
	provider domain.Provider,
	runner *backup_volumes.Runner,
	target domain.Target,
	app domain.App,
	restore domain.Restore,
) (domain.BackupID, error) {
	existing, _, err := provider.Volumes(ctx, target, app.ID(), restore.Environment())

	if err != nil {
		return "", err
	}

	if len(existing) == 0 {
		return "", nil
	}

	// Only the volumes this restore replaces, if they exist yet
	var keys []string

	for _, v := range existing {
		if len(restore.Volumes()) == 0 || contains(restore.Volumes(), v.Key) {
			keys = append(keys, v.Key)
		}
	}

	if len(keys) == 0 {
		return "", nil
	}

	snapshot, err := domain.NewBackup(app, restore.Environment(), domain.BackupTriggerBeforeRestore, keys, true, restore.Requested().By())

	if err != nil {
		return "", err
	}

	_, err = runner.Run(ctx, snapshot)

	return snapshot.ID(), err
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}
