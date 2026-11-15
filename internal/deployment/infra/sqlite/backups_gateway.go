package sqlite

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/app/get_backups"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

const backupSummarySelect = `
	SELECT
		backups.id
		,backups.app_id
		,COALESCE(apps.name, backups.app_name)
		,apps.id IS NULL
		,backups.environment
		,backups.target_id
		,targets.name
		,backups.reason
		,backups.status
		,backups.stop_containers
		,backups.volumes
		,backups.result
		,backups.local_copy
		,backups.stored_copy
		,backups.store_error
		,backups.error_code
		,backups.requested_at
		,users.id
		,users.email
		,backups.finished_at
	FROM backups
	INNER JOIN users ON users.id = backups.requested_by
	LEFT JOIN apps ON apps.id = backups.app_id
	LEFT JOIN targets ON targets.id = backups.target_id`

func (s *gateway) GetBackups(ctx context.Context, query get_backups.Query) ([]get_backups.Backup, error) {
	q := builder.Query[get_backups.Backup](backupSummarySelect)

	if id, isSet := query.AppID.TryGet(); isSet {
		q = q.F("WHERE backups.app_id = ?", id)
	}

	return q.F("ORDER BY backups.requested_at DESC").All(s.db, ctx, backupMapper)
}

func (s *gateway) GetBackupByID(ctx context.Context, query get_backups.ByIDQuery) (get_backups.BackupDetail, error) {
	backup, err := builder.
		Query[get_backups.Backup](backupSummarySelect+" WHERE backups.id = ?", query.ID).
		One(s.db, ctx, backupMapper)

	if err != nil {
		return get_backups.BackupDetail{}, err
	}

	restores, err := builder.
		Query[get_backups.Restore](`
		SELECT
			backup_restores.id
			,backup_restores.app_id
			,COALESCE(apps.name, '')
			,apps.id IS NULL
			,backup_restores.environment
			,backup_restores.status
			,backup_restores.volumes
			,backup_restores.restored
			,backup_restores.snapshot_id
			,backup_restores.error_code
			,backup_restores.requested_at
			,users.id
			,users.email
			,backup_restores.finished_at
		FROM backup_restores
		INNER JOIN users ON users.id = backup_restores.requested_by
		LEFT JOIN apps ON apps.id = backup_restores.app_id
		WHERE backup_restores.backup_id = ?
		ORDER BY backup_restores.requested_at DESC`, query.ID).
		All(s.db, ctx, restoreMapper)

	if err != nil {
		return get_backups.BackupDetail{}, err
	}

	return get_backups.BackupDetail{Backup: backup, Restores: restores}, nil
}

func backupMapper(scanner storage.Scanner) (b get_backups.Backup, err error) {
	var (
		volumes    domain.VolumeKeys
		targetName monad.Maybe[string]
	)

	err = scanner.Scan(
		&b.ID,
		&b.App.ID,
		&b.App.Name,
		&b.App.Deleted,
		&b.Environment,
		&b.Target.ID,
		&targetName,
		&b.Trigger,
		&b.Status,
		&b.StopContainers,
		&volumes,
		&b.Result,
		&b.Local,
		&b.Stored,
		&b.StoreError,
		&b.ErrorCode,
		&b.RequestedAt,
		&b.RequestedBy.ID,
		&b.RequestedBy.Email,
		&b.FinishedAt,
	)

	b.Target.Name = targetName
	b.Volumes = volumes

	if b.Volumes == nil {
		b.Volumes = []string{}
	}

	return b, err
}

func restoreMapper(scanner storage.Scanner) (r get_backups.Restore, err error) {
	var volumes, restored domain.VolumeKeys

	err = scanner.Scan(
		&r.ID,
		&r.App.ID,
		&r.App.Name,
		&r.App.Deleted,
		&r.Environment,
		&r.Status,
		&volumes,
		&restored,
		&r.SnapshotID,
		&r.ErrorCode,
		&r.RequestedAt,
		&r.RequestedBy.ID,
		&r.RequestedBy.Email,
		&r.FinishedAt,
	)

	r.Volumes = append([]string{}, volumes...)
	r.Restored = append([]string{}, restored...)

	return r, err
}
