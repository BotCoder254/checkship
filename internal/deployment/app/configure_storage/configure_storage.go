package configure_storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

var (
	ErrStorageUnreachable = apperr.New("storage_unreachable")
	ErrStorageMoveFailed  = apperr.New("storage_move_failed")
)

const checkAttempts = 10 // The gateway needs a moment after its container starts

type (
	// Runs the built-in backup storage on its target, finds how to reach it and,
	// when it moved, copies the archives over from the target it left.
	Command struct {
		bus.Command[bus.UnitType]

		Version        time.Time                               `json:"version"`
		PreviousTarget monad.Maybe[string]                     `json:"previous_target"`
		PreviousAccess monad.Maybe[domain.BackupStorageAccess] `json:"previous_access"`
	}

	Options interface {
		BuiltinStorageImage() string
	}

	// Copies every archive from one storage to another.
	Mover interface {
		Move(ctx context.Context, from, to domain.BackupStorageAccess) (int, error)
	}
)

func (Command) Name_() string        { return "deployment.command.configure_storage" }
func (c Command) ResourceID() string { return "backup_storage" }

func Handler(
	reader domain.BackupStorageReader,
	writer domain.BackupStorageWriter,
	targets domain.TargetsReader,
	provider domain.Provider,
	checker domain.BackupStorageChecker,
	mover Mover,
	options Options,
	logger log.Logger,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		settings, err := reader.Get(ctx)

		if err != nil {
			return bus.Unit, err
		}

		if settings.Mode() != domain.BackupStorageBuiltin || !settings.Version().Equal(cmd.Version) {
			return bus.Unit, nil // Changed since, a newer job takes over
		}

		endpoint, useSSL, setupErr := setup(ctx, settings, targets, provider, checker, options, logger)

		if setupErr == nil {
			setupErr = move(ctx, settings, endpoint, useSSL, cmd, targets, provider, mover, logger)
		}

		if setupErr != nil {
			logger.Errorw("built-in backup storage not ready", "error", setupErr)
		}

		// Read again: the settings may have changed while the storage started
		if settings, err = reader.Get(ctx); err != nil {
			return bus.Unit, err
		}

		settings.BuiltinConfigured(cmd.Version, endpoint, useSSL, setupErr)

		return bus.Unit, writer.Write(ctx, &settings)
	}
}

func setup(
	ctx context.Context,
	settings domain.BackupStorage,
	targets domain.TargetsReader,
	provider domain.Provider,
	checker domain.BackupStorageChecker,
	options Options,
	logger log.Logger,
) (string, bool, error) {
	target, err := targets.GetByID(ctx, settings.Target().MustGet())

	if err != nil {
		return "", false, err
	}

	access := settings.Access()
	endpoints, err := provider.SetupStorage(ctx, target, domain.BuiltinStorageSpec{
		Image:     options.BuiltinStorageImage(),
		AccessKey: access.AccessKey,
		SecretKey: string(access.SecretKey),
	})

	if err != nil {
		return "", false, err
	}

	var errs []error

	for attempt := 0; attempt < checkAttempts; attempt++ {
		for _, e := range endpoints {
			candidate := access
			candidate.Endpoint = e.Address
			candidate.UseSSL = e.UseSSL

			checkErr := checker.Check(ctx, candidate)

			if checkErr == nil {
				logger.Infow("built-in backup storage ready", "endpoint", e.Address, "target", target.ID())
				return e.Address, e.UseSSL, nil
			}

			errs = append(errs, fmt.Errorf("%s: %w", e.Address, checkErr))
		}

		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return "", false, apperr.Wrap(ErrStorageUnreachable, errors.Join(errs[max(0, len(errs)-len(endpoints)):]...))
}

// Copies the archives of the storage left behind, then removes it. Nothing is
// removed unless every archive arrived.
func move(
	ctx context.Context,
	settings domain.BackupStorage,
	endpoint string,
	useSSL bool,
	cmd Command,
	targets domain.TargetsReader,
	provider domain.Provider,
	mover Mover,
	logger log.Logger,
) error {
	from, isMoving := cmd.PreviousAccess.TryGet()

	if !isMoving {
		return nil
	}

	to := settings.Access()
	to.Endpoint = endpoint
	to.UseSSL = useSSL

	count, err := mover.Move(ctx, from, to)

	if err != nil {
		return apperr.Wrap(ErrStorageMoveFailed, err)
	}

	logger.Infow("archives moved to the new built-in storage", "count", count)

	previous, err := targets.GetByID(ctx, domain.TargetID(cmd.PreviousTarget.Get("")))

	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil
		}

		return err
	}

	if err = provider.RemoveStorage(ctx, previous); err != nil {
		logger.Warnw("could not remove the previous built-in storage, its data is still there", "target", previous.ID(), "error", err)
	}

	return nil
}

// Queue the setup of the built-in storage each time its settings change.
func OnBackupStorageChangedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.BackupStorageChanged] {
	return func(ctx context.Context, evt domain.BackupStorageChanged) error {
		if evt.Mode != domain.BackupStorageBuiltin || evt.Status != domain.BackupStorageConfiguring {
			return nil
		}

		var previous monad.Maybe[string]

		if id, isSet := evt.Previous.TryGet(); isSet {
			previous.Set(string(id))
		}

		return scheduler.Queue(ctx, Command{
			Version:        evt.Version,
			PreviousTarget: previous,
			PreviousAccess: evt.PreviousAccess,
		}, bus.WithGroup("deployment.backup_storage"), bus.WithPolicy(bus.JobPolicyMerge))
	}
}
