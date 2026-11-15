package get_backup_storage

import (
	"context"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

type (
	// Where backups are copied, with the secret masked.
	Query struct {
		bus.Query[Storage]
	}

	Storage struct {
		Mode      string                 `json:"mode"`
		TargetID  monad.Maybe[string]    `json:"target_id"`
		Endpoint  string                 `json:"endpoint"`
		UseSSL    bool                   `json:"use_ssl"`
		PathStyle bool                   `json:"path_style"`
		Region    string                 `json:"region"`
		Bucket    string                 `json:"bucket"`
		Prefix    string                 `json:"prefix"`
		AccessKey string                 `json:"access_key"`
		SecretKey storage.SecretString   `json:"secret_key"`
		KeepLocal bool                   `json:"keep_local"`
		Status    string                 `json:"status"`
		ErrorCode monad.Maybe[string]    `json:"error_code"`
		UpdatedAt monad.Maybe[time.Time] `json:"updated_at"`
		// Images pulled on targets: the built-in storage, and the helper that mounts volumes
		Image       string `json:"image"`
		HelperImage string `json:"helper_image"`
	}

	Options interface {
		BuiltinStorageImage() string
		BackupHelperImage() string
	}
)

func (Query) Name_() string { return "deployment.query.get_backup_storage" }

func Handler(reader domain.BackupStorageReader, options Options) bus.RequestHandler[Storage, Query] {
	return func(ctx context.Context, _ Query) (Storage, error) {
		s, err := reader.Get(ctx)

		if err != nil {
			return Storage{}, err
		}

		access := s.Access()
		result := Storage{
			Mode:        string(s.Mode()),
			Endpoint:    access.Endpoint,
			UseSSL:      access.UseSSL,
			PathStyle:   access.PathStyle,
			Region:      access.Region,
			Bucket:      access.Bucket,
			Prefix:      access.Prefix,
			AccessKey:   access.AccessKey,
			SecretKey:   access.SecretKey,
			KeepLocal:   s.KeepLocal(),
			Status:      string(s.Status()),
			ErrorCode:   s.ErrorCode(),
			Image:       options.BuiltinStorageImage(),
			HelperImage: options.BackupHelperImage(),
		}

		if id, isSet := s.Target().TryGet(); isSet {
			result.TargetID.Set(string(id))
		}

		if s.Saved() {
			result.UpdatedAt.Set(s.Version())
		}

		return result, nil
	}
}
