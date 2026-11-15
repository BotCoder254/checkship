package sqlite

import (
	"context"
	"errors"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

type (
	BackupStorageStore interface {
		domain.BackupStorageReader
		domain.BackupStorageWriter
	}

	backupStorageStore struct {
		db *sqlite.Database
	}
)

func NewBackupStorageStore(db *sqlite.Database) BackupStorageStore {
	return &backupStorageStore{db}
}

func (s *backupStorageStore) Get(ctx context.Context) (domain.BackupStorage, error) {
	settings, err := builder.
		Query[domain.BackupStorage](`
		SELECT
			mode
			,target_id
			,endpoint
			,use_ssl
			,path_style
			,region
			,bucket
			,prefix
			,access_key
			,secret_key
			,keep_local
			,status
			,error_code
			,version
		FROM backup_storage
		WHERE id = 1`).
		One(s.db, ctx, domain.BackupStorageFrom)

	if errors.Is(err, apperr.ErrNotFound) {
		return domain.DefaultBackupStorage(), nil
	}

	return settings, err
}

func (s *backupStorageStore) Write(ctx context.Context, settings ...*domain.BackupStorage) error {
	return sqlite.WriteAndDispatch(s.db, ctx, settings, func(ctx context.Context, e event.Event) error {
		evt, ok := e.(domain.BackupStorageChanged)

		if !ok {
			return nil
		}

		return builder.
			Command(`INSERT INTO backup_storage (id, mode, target_id, endpoint, use_ssl, path_style, region, bucket, prefix, access_key, secret_key, keep_local, status, error_code, version)
				VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET
					mode = excluded.mode
					,target_id = excluded.target_id
					,endpoint = excluded.endpoint
					,use_ssl = excluded.use_ssl
					,path_style = excluded.path_style
					,region = excluded.region
					,bucket = excluded.bucket
					,prefix = excluded.prefix
					,access_key = excluded.access_key
					,secret_key = excluded.secret_key
					,keep_local = excluded.keep_local
					,status = excluded.status
					,error_code = excluded.error_code
					,version = excluded.version`,
				evt.Mode, evt.Target, evt.Access.Endpoint, evt.Access.UseSSL, evt.Access.PathStyle,
				evt.Access.Region, evt.Access.Bucket, evt.Access.Prefix, evt.Access.AccessKey,
				string(evt.Access.SecretKey), evt.KeepLocal, evt.Status, evt.ErrorCode, evt.Version).
			Exec(s.db, ctx)
	})
}
