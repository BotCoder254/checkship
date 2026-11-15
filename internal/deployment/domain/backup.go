package domain

import (
	"context"
	"database/sql/driver"
	"time"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	shared "github.com/checkship-deploy/checkship/pkg/domain"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

var (
	ErrInvalidBackupTrigger = apperr.New("invalid_backup_trigger")
	ErrBackupNotFinished    = apperr.New("backup_not_finished")
	ErrBackupNotRetryable   = apperr.New("backup_not_retryable")
	ErrBackupNotRestorable  = apperr.New("backup_not_restorable")
	ErrBackupChecksum       = apperr.New("checksum_mismatch")
	ErrBackupStoreFailed    = apperr.New("backup_store_failed")
	ErrStorageUnreachable   = apperr.New("storage_unreachable")
)

const (
	BackupTriggerManual             BackupTrigger = "manual"
	BackupTriggerUpload             BackupTrigger = "upload"
	BackupTriggerBeforeDelete       BackupTrigger = "before_delete"
	BackupTriggerBeforeTargetChange BackupTrigger = "before_target_change"
	BackupTriggerBeforeRestore      BackupTrigger = "before_restore"

	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusSucceeded BackupStatus = "succeeded"
	BackupStatusFailed    BackupStatus = "failed"
)

type (
	BackupID      string
	BackupTrigger string
	BackupStatus  string
	VolumeKeys    []string // Compose names of volumes, stored as JSON

	// Copy of an application environment's volumes, kept as one archive in the
	// data folder and, when configured, in an S3-compatible storage.
	Backup struct {
		event.Emitter

		id             BackupID
		app            AppID
		appName        AppName
		environment    Environment
		target         TargetID
		trigger        BackupTrigger
		status         BackupStatus
		stopContainers bool
		volumes        []string // Keys asked for, every volume if empty
		result         monad.Maybe[BackupResult]
		local          bool                // A copy is in the data folder
		stored         bool                // A copy is in the storage
		storeError     monad.Maybe[string] // Why the storage copy is missing
		errorCode      monad.Maybe[string]
		requested      shared.Action[auth.UserID]
		finishedAt     monad.Maybe[time.Time]
	}

	// What a finished backup contains.
	BackupResult struct {
		Size    int64                        `json:"size"`
		SHA256  string                       `json:"sha256"`
		Volumes []volumearchive.VolumeStats  `json:"volumes"`
		Skipped []volumearchive.SkippedMount `json:"skipped,omitempty"`
	}

	BackupsReader interface {
		GetByID(context.Context, BackupID) (Backup, error)
		GetByApp(context.Context, AppID) ([]Backup, error)
	}

	BackupsWriter interface {
		Write(context.Context, ...*Backup) error
	}

	BackupRequested struct {
		bus.Notification

		ID             BackupID
		App            AppID
		AppName        AppName
		Environment    Environment
		Target         TargetID
		Trigger        BackupTrigger
		StopContainers bool
		Volumes        []string
		Requested      shared.Action[auth.UserID]
	}

	BackupStarted struct {
		bus.Notification

		ID BackupID
	}

	BackupSucceeded struct {
		bus.Notification

		ID         BackupID
		App        AppID
		Result     BackupResult
		Local      bool
		Stored     bool
		StoreError monad.Maybe[string]
		FinishedAt time.Time
	}

	BackupFailed struct {
		bus.Notification

		ID         BackupID
		App        AppID
		ErrorCode  string
		FinishedAt time.Time
	}

	// A failed backup goes back to pending. Inline is true when the job that needs
	// it runs it right away, so no job is queued for it.
	BackupRetried struct {
		bus.Notification

		ID          BackupID
		App         AppID
		AppName     AppName
		Environment Environment
		Inline      bool
	}

	// The copies of a finished backup changed: uploaded again, local copy removed.
	BackupLocationsChanged struct {
		bus.Notification

		ID         BackupID
		App        AppID
		Local      bool
		Stored     bool
		StoreError monad.Maybe[string]
	}

	BackupDeleted struct {
		bus.Notification

		ID     BackupID
		App    AppID
		Local  bool
		Stored bool
	}
)

func (BackupRequested) Name_() string        { return "deployment.event.backup_requested" }
func (BackupStarted) Name_() string          { return "deployment.event.backup_started" }
func (BackupSucceeded) Name_() string        { return "deployment.event.backup_succeeded" }
func (BackupFailed) Name_() string           { return "deployment.event.backup_failed" }
func (BackupRetried) Name_() string          { return "deployment.event.backup_retried" }
func (BackupLocationsChanged) Name_() string { return "deployment.event.backup_locations_changed" }
func (BackupDeleted) Name_() string          { return "deployment.event.backup_deleted" }

func (k VolumeKeys) Value() (driver.Value, error) { return storage.ValueJSON(k) }
func (k *VolumeKeys) Scan(value any) error        { return storage.ScanJSON(value, k) }

func (r BackupResult) Value() (driver.Value, error) { return storage.ValueJSON(r) }
func (r *BackupResult) Scan(value any) error        { return storage.ScanJSON(value, r) }

func BackupTriggerFrom(value string) (BackupTrigger, error) {
	switch t := BackupTrigger(value); t {
	case BackupTriggerManual, BackupTriggerUpload, BackupTriggerBeforeDelete, BackupTriggerBeforeTargetChange, BackupTriggerBeforeRestore:
		return t, nil
	default:
		return "", ErrInvalidBackupTrigger
	}
}

// Requests a backup of an application environment's volumes, on the target the
// environment uses.
func NewBackup(
	app App,
	env Environment,
	trigger BackupTrigger,
	volumes []string,
	stopContainers bool,
	uid auth.UserID,
) (Backup, error) {
	return NewBackupOnTarget(app, env, app.Config(env).Target(), trigger, volumes, stopContainers, uid)
}

// Requests a backup on an explicit target: the one an environment is leaving,
// whose volumes are still there while the app already points elsewhere.
func NewBackupOnTarget(
	app App,
	env Environment,
	target TargetID,
	trigger BackupTrigger,
	volumes []string,
	stopContainers bool,
	uid auth.UserID,
) (b Backup, err error) {
	if _, err = BackupTriggerFrom(string(trigger)); err != nil || trigger == BackupTriggerUpload {
		return b, ErrInvalidBackupTrigger
	}

	// Backups taken because the app is being deleted come after the cleanup request.
	if app.cleanupRequested.HasValue() && trigger != BackupTriggerBeforeDelete {
		return b, ErrAppCleanupRequested
	}

	if _, err = EnvironmentFrom(string(env)); err != nil {
		return b, err
	}

	for _, key := range volumes {
		if err = volumearchive.ValidateVolumeKey(key); err != nil {
			return b, err
		}
	}

	return newBackup(app.id, app.name, env, target, trigger, volumes, stopContainers, uid), nil
}

// Records an archive uploaded by a person, already validated and stored locally.
func NewUploadedBackup(
	app App,
	env Environment,
	result BackupResult,
	uid auth.UserID,
) (b Backup, err error) {
	if app.cleanupRequested.HasValue() {
		return b, ErrAppCleanupRequested
	}

	if _, err = EnvironmentFrom(string(env)); err != nil {
		return b, err
	}

	b = newBackup(app.id, app.name, env, app.Config(env).Target(), BackupTriggerUpload, nil, false, uid)
	b.apply(BackupSucceeded{
		ID:         b.id,
		App:        b.app,
		Result:     result,
		Local:      true,
		FinishedAt: time.Now().UTC(),
	})

	return b, nil
}

func newBackup(app AppID, name AppName, env Environment, target TargetID, trigger BackupTrigger, volumes []string, stop bool, uid auth.UserID) (b Backup) {
	b.apply(BackupRequested{
		ID:             id.New[BackupID](),
		App:            app,
		AppName:        name,
		Environment:    env,
		Target:         target,
		Trigger:        trigger,
		StopContainers: stop,
		Volumes:        volumes,
		Requested:      shared.NewAction(uid),
	})

	return b
}

// Recreates a backup from the persistent storage.
func BackupFrom(scanner storage.Scanner) (b Backup, err error) {
	var (
		volumes     VolumeKeys
		result      monad.Maybe[BackupResult]
		requestedAt time.Time
		requestedBy auth.UserID
	)

	err = scanner.Scan(
		&b.id,
		&b.app,
		&b.appName,
		&b.environment,
		&b.target,
		&b.trigger,
		&b.status,
		&b.stopContainers,
		&volumes,
		&result,
		&b.local,
		&b.stored,
		&b.storeError,
		&b.errorCode,
		&requestedAt,
		&requestedBy,
		&b.finishedAt,
	)

	b.volumes = volumes
	b.result = result
	b.requested = shared.ActionFrom(requestedBy, requestedAt)

	return b, err
}

// The provider started copying the volumes.
func (b *Backup) Started() error {
	if b.status != BackupStatusPending {
		return nil // A retried job starts again from running, nothing to record
	}

	b.apply(BackupStarted{ID: b.id})

	return nil
}

// The archive is written. storeErr is set when the copy to the storage failed,
// in which case the local copy is always kept.
func (b *Backup) Succeeded(result BackupResult, local, stored bool, storeErr error) {
	var storeError monad.Maybe[string]

	if storeErr != nil {
		storeError.Set(errorCode(storeErr))
		local = true
	}

	b.apply(BackupSucceeded{
		ID:         b.id,
		App:        b.app,
		Result:     result,
		Local:      local,
		Stored:     stored,
		StoreError: storeError,
		FinishedAt: time.Now().UTC(),
	})
}

func (b *Backup) Failed(err error) {
	b.apply(BackupFailed{
		ID:         b.id,
		App:        b.app,
		ErrorCode:  errorCode(err),
		FinishedAt: time.Now().UTC(),
	})
}

// Runs a failed backup again. inline: the caller runs it itself instead of a job.
func (b *Backup) Retry(inline bool) error {
	if b.status != BackupStatusFailed {
		return ErrBackupNotRetryable
	}

	b.apply(BackupRetried{
		ID:          b.id,
		App:         b.app,
		AppName:     b.appName,
		Environment: b.environment,
		Inline:      inline,
	})

	return nil
}

// Records where the copies of a finished backup are.
func (b *Backup) HasCopies(local, stored bool, storeErr error) error {
	if b.status != BackupStatusSucceeded {
		return ErrBackupNotFinished
	}

	var storeError monad.Maybe[string]

	if storeErr != nil {
		storeError.Set(errorCode(storeErr))
		local = true // Never drop the last copy
	}

	if !local && !stored {
		local = true
	}

	if local == b.local && stored == b.stored && storeError == b.storeError {
		return nil
	}

	b.apply(BackupLocationsChanged{
		ID:         b.id,
		App:        b.app,
		Local:      local,
		Stored:     stored,
		StoreError: storeError,
	})

	return nil
}

// Checks the backup can be restored and returns what is needed to verify it.
func (b *Backup) Restorable() (BackupResult, error) {
	result, done := b.result.TryGet()

	if b.status != BackupStatusSucceeded || !done || (!b.local && !b.stored) {
		return result, ErrBackupNotRestorable
	}

	return result, nil
}

// Deleting a running backup would leave its job writing an archive nobody owns.
func (b *Backup) Delete() error {
	if b.status == BackupStatusPending || b.status == BackupStatusRunning {
		return ErrBackupNotFinished
	}

	b.apply(BackupDeleted{
		ID:     b.id,
		App:    b.app,
		Local:  b.local,
		Stored: b.stored,
	})

	return nil
}

func (b *Backup) ID() BackupID                      { return b.id }
func (b *Backup) App() AppID                        { return b.app }
func (b *Backup) AppName() AppName                  { return b.appName }
func (b *Backup) Environment() Environment          { return b.environment }
func (b *Backup) Target() TargetID                  { return b.target }
func (b *Backup) Trigger() BackupTrigger            { return b.trigger }
func (b *Backup) Status() BackupStatus              { return b.status }
func (b *Backup) StopContainers() bool              { return b.stopContainers }
func (b *Backup) Volumes() []string                 { return b.volumes }
func (b *Backup) Result() monad.Maybe[BackupResult] { return b.result }
func (b *Backup) Local() bool                       { return b.local }
func (b *Backup) Stored() bool                      { return b.stored }
func (b *Backup) StoreError() monad.Maybe[string]   { return b.storeError }
func (b *Backup) ErrorCode() monad.Maybe[string]    { return b.errorCode }
func (b *Backup) Requested() shared.Action[auth.UserID] {
	return b.requested
}

func (b *Backup) apply(e event.Event) {
	switch v := e.(type) {
	case BackupRequested:
		b.id = v.ID
		b.app = v.App
		b.appName = v.AppName
		b.environment = v.Environment
		b.target = v.Target
		b.trigger = v.Trigger
		b.stopContainers = v.StopContainers
		b.volumes = v.Volumes
		b.requested = v.Requested
		b.status = BackupStatusPending
	case BackupStarted:
		b.status = BackupStatusRunning
	case BackupSucceeded:
		b.status = BackupStatusSucceeded
		b.result.Set(v.Result)
		b.local = v.Local
		b.stored = v.Stored
		b.storeError = v.StoreError
		b.errorCode.Unset()
		b.finishedAt.Set(v.FinishedAt)
	case BackupFailed:
		b.status = BackupStatusFailed
		b.errorCode.Set(v.ErrorCode)
		b.finishedAt.Set(v.FinishedAt)
	case BackupRetried:
		b.status = BackupStatusPending
		b.errorCode.Unset()
		b.finishedAt.Unset()
	case BackupLocationsChanged:
		b.local = v.Local
		b.stored = v.Stored
		b.storeError = v.StoreError
	}

	event.Store(b, e)
}

// Code shown to people for an error: the application error code when there is
// one, a generic one otherwise (details stay in the logs).
func errorCode(err error) string {
	if appErr, ok := apperr.As[apperr.Error](err); ok {
		return appErr.Code
	}

	return "unexpected_error"
}
