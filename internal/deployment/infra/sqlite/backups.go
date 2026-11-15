package sqlite

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

type (
	BackupsStore interface {
		domain.BackupsReader
		domain.BackupsWriter
	}

	backupsStore struct {
		db *sqlite.Database
	}
)

func NewBackupsStore(db *sqlite.Database) BackupsStore {
	return &backupsStore{db}
}

const backupColumns = `
	id
	,app_id
	,app_name
	,environment
	,target_id
	,reason
	,status
	,stop_containers
	,volumes
	,result
	,local_copy
	,stored_copy
	,store_error
	,error_code
	,requested_at
	,requested_by
	,finished_at`

func (s *backupsStore) GetByID(ctx context.Context, id domain.BackupID) (domain.Backup, error) {
	return builder.
		Query[domain.Backup]("SELECT"+backupColumns+" FROM backups WHERE id = ?", id).
		One(s.db, ctx, domain.BackupFrom)
}

func (s *backupsStore) GetByApp(ctx context.Context, id domain.AppID) ([]domain.Backup, error) {
	return builder.
		Query[domain.Backup]("SELECT"+backupColumns+" FROM backups WHERE app_id = ? ORDER BY requested_at DESC", id).
		All(s.db, ctx, domain.BackupFrom)
}

func (s *backupsStore) Write(ctx context.Context, backups ...*domain.Backup) error {
	return sqlite.WriteAndDispatch(s.db, ctx, backups, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.BackupRequested:
			return builder.
				Insert("backups", builder.Values{
					"id":              evt.ID,
					"app_id":          evt.App,
					"app_name":        evt.AppName,
					"environment":     evt.Environment,
					"target_id":       evt.Target,
					"reason":          evt.Trigger,
					"status":          domain.BackupStatusPending,
					"stop_containers": evt.StopContainers,
					"volumes":         domain.VolumeKeys(evt.Volumes),
					"requested_at":    evt.Requested.At(),
					"requested_by":    evt.Requested.By(),
				}).
				Exec(s.db, ctx)
		case domain.BackupStarted:
			return builder.
				Update("backups", builder.Values{
					"status": domain.BackupStatusRunning,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.BackupSucceeded:
			return builder.
				Update("backups", builder.Values{
					"status":      domain.BackupStatusSucceeded,
					"result":      evt.Result,
					"local_copy":  evt.Local,
					"stored_copy": evt.Stored,
					"store_error": evt.StoreError,
					"error_code":  nil,
					"finished_at": evt.FinishedAt,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.BackupFailed:
			return builder.
				Update("backups", builder.Values{
					"status":      domain.BackupStatusFailed,
					"error_code":  evt.ErrorCode,
					"finished_at": evt.FinishedAt,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.BackupRetried:
			return builder.
				Update("backups", builder.Values{
					"status":      domain.BackupStatusPending,
					"error_code":  nil,
					"finished_at": nil,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.BackupLocationsChanged:
			return builder.
				Update("backups", builder.Values{
					"local_copy":  evt.Local,
					"stored_copy": evt.Stored,
					"store_error": evt.StoreError,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.BackupDeleted:
			return builder.
				Command("DELETE FROM backups WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		default:
			return nil
		}
	})
}
