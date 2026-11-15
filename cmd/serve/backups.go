package serve

import (
	nethttp "net/http"

	"github.com/checkship-deploy/checkship/internal/deployment/app/delete_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_volumes"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_backup_archive"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_backup_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_backups"
	"github.com/checkship-deploy/checkship/internal/deployment/app/request_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/request_restore"
	"github.com/checkship-deploy/checkship/internal/deployment/app/retry_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/store_backup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_backup_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/app/upload_backup"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/http"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/gin-gonic/gin"
)

// Multipart overhead allowed on top of the archive itself (boundaries, other fields).
const uploadOverhead = 1 << 20

func (s *server) appVolumesHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		data, err := bus.Send(s.bus, c.Request.Context(), get_app_volumes.Query{
			AppID:       c.Param("id"),
			Environment: c.Query("environment"),
		})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) listAppBackupsHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		data, err := bus.Send(s.bus, c.Request.Context(), get_backups.Query{
			AppID: monad.Value(c.Param("id")),
		})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) listBackupsHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		var query get_backups.Query

		if app := c.Query("app_id"); app != "" {
			query.AppID.Set(app)
		}

		data, err := bus.Send(s.bus, c.Request.Context(), query)

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) getBackupHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		data, err := bus.Send(s.bus, c.Request.Context(), get_backups.ByIDQuery{ID: c.Param("id")})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) requestBackupHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, cmd request_backup.Command) error {
		cmd.AppID = c.Param("id")
		return sendBackupCreated(s, c, cmd)
	})
}

func (s *server) uploadBackupHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Refuse oversized bodies before they are written to a temporary file
		c.Request.Body = nethttp.MaxBytesReader(c.Writer, c.Request.Body, s.options.BackupsMaxUploadSize()+uploadOverhead)

		http.Bind(s, func(c *gin.Context, cmd upload_backup.Command) error {
			cmd.AppID = c.Param("id")
			return sendBackupCreated(s, c, cmd)
		})(c)
	}
}

func (s *server) deleteBackupHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		if _, err := bus.Send(s.bus, c.Request.Context(), delete_backup.Command{ID: c.Param("id")}); err != nil {
			return err
		}

		return http.NoContent(c)
	})
}

func (s *server) storeBackupHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		if _, err := bus.Send(s.bus, c.Request.Context(), store_backup.Request{ID: c.Param("id")}); err != nil {
			return err
		}

		return http.NoContent(c)
	})
}

func (s *server) retryBackupHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		if _, err := bus.Send(s.bus, c.Request.Context(), retry_backup.Command{ID: c.Param("id")}); err != nil {
			return err
		}

		data, err := bus.Send(s.bus, c.Request.Context(), get_backups.ByIDQuery{ID: c.Param("id")})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) restoreBackupHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, cmd request_restore.Command) error {
		cmd.BackupID = c.Param("id")
		ctx := c.Request.Context()

		if _, err := bus.Send(s.bus, ctx, cmd); err != nil {
			return err
		}

		data, err := bus.Send(s.bus, ctx, get_backups.ByIDQuery{ID: cmd.BackupID})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) downloadBackupHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		archive, err := bus.Send(s.bus, c.Request.Context(), get_backup_archive.Query{ID: c.Param("id")})

		if err != nil {
			return err
		}

		defer archive.Content.Close()

		if err = http.Attachment(c, archive.Filename, "application/gzip", archive.Size, archive.Content); err != nil {
			// Headers are sent: the client sees a short download, the log says why
			s.logger.Warnw("backup download interrupted", "backup", c.Param("id"), "error", err)
		}

		return nil
	})
}

func sendBackupCreated[T bus.TypedRequest[string]](s *server, c *gin.Context, cmd T) error {
	ctx := c.Request.Context()

	id, err := bus.Send(s.bus, ctx, cmd)

	if err != nil {
		return err
	}

	data, err := bus.Send(s.bus, ctx, get_backups.ByIDQuery{ID: id})

	if err != nil {
		return err
	}

	return http.Created(s, c, data, "/api/v1/backups/%s", id)
}

func (s *server) getBackupStorageHandler() gin.HandlerFunc {
	return http.Send(s, func(c *gin.Context) error {
		data, err := bus.Send(s.bus, c.Request.Context(), get_backup_storage.Query{})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) updateBackupStorageHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, cmd update_backup_storage.Command) error {
		ctx := c.Request.Context()

		if _, err := bus.Send(s.bus, ctx, cmd); err != nil {
			return err
		}

		data, err := bus.Send(s.bus, ctx, get_backup_storage.Query{})

		if err != nil {
			return err
		}

		return http.Ok(c, data)
	})
}

func (s *server) testBackupStorageHandler() gin.HandlerFunc {
	return http.Bind(s, func(c *gin.Context, cmd update_backup_storage.TestCommand) error {
		if _, err := bus.Send(s.bus, c.Request.Context(), cmd); err != nil {
			return err
		}

		return http.NoContent(c)
	})
}
