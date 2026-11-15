package sqlite

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

type (
	RestoresStore interface {
		domain.RestoresReader
		domain.RestoresWriter
	}

	restoresStore struct {
		db *sqlite.Database
	}
)

func NewRestoresStore(db *sqlite.Database) RestoresStore {
	return &restoresStore{db}
}

const restoreColumns = `
	id
	,backup_id
	,app_id
	,environment
	,target_id
	,volumes
	,snapshot_first
	,status
	,restored
	,snapshot_id
	,error_code
	,requested_at
	,requested_by
	,finished_at`

func (s *restoresStore) GetByBackup(ctx context.Context, id domain.BackupID) ([]domain.Restore, error) {
	return builder.
		Query[domain.Restore]("SELECT"+restoreColumns+" FROM backup_restores WHERE backup_id = ? ORDER BY requested_at DESC", id).
		All(s.db, ctx, domain.RestoreFrom)
}

func (s *restoresStore) GetByID(ctx context.Context, id domain.RestoreID) (domain.Restore, error) {
	return builder.
		Query[domain.Restore](`
		SELECT
			id
			,backup_id
			,app_id
			,environment
			,target_id
			,volumes
			,snapshot_first
			,status
			,restored
			,snapshot_id
			,error_code
			,requested_at
			,requested_by
			,finished_at
		FROM backup_restores
		WHERE id = ?`, id).
		One(s.db, ctx, domain.RestoreFrom)
}

func (s *restoresStore) Write(ctx context.Context, restores ...*domain.Restore) error {
	return sqlite.WriteAndDispatch(s.db, ctx, restores, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.RestoreRequested:
			return builder.
				Insert("backup_restores", builder.Values{
					"id":             evt.ID,
					"backup_id":      evt.Backup,
					"app_id":         evt.App,
					"environment":    evt.Environment,
					"target_id":      evt.Target,
					"volumes":        evt.Volumes,
					"snapshot_first": evt.SnapshotFirst,
					"status":         domain.RestoreStatusPending,
					"requested_at":   evt.Requested.At(),
					"requested_by":   evt.Requested.By(),
				}).
				Exec(s.db, ctx)
		case domain.RestoreStarted:
			return builder.
				Update("backup_restores", builder.Values{"status": domain.RestoreStatusRunning}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.RestoreSucceeded:
			return builder.
				Update("backup_restores", builder.Values{
					"status":      domain.RestoreStatusSucceeded,
					"restored":    evt.Restored,
					"snapshot_id": evt.Snapshot,
					"finished_at": evt.FinishedAt,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.RestoreFailed:
			return builder.
				Update("backup_restores", builder.Values{
					"status":      domain.RestoreStatusFailed,
					"error_code":  evt.ErrorCode,
					"snapshot_id": evt.Snapshot,
					"finished_at": evt.FinishedAt,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		default:
			return nil
		}
	})
}
