package serve

import (
	nethttp "net/http"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/check_app_domains"
	"github.com/checkship-deploy/checkship/internal/deployment/app/create_app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_detail"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_apps"
	"github.com/checkship-deploy/checkship/internal/deployment/app/request_app_cleanup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/update_app"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/http"
	"github.com/gin-gonic/gin"
)

func (s *server) createAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd create_app.Command) error {
		context := ctx.Request.Context()
		appid, err := bus.Send(s.bus, context, cmd)

		if err != nil {
			return err
		}

		data, err := bus.Send(s.bus, context, get_app_detail.Query{
			ID: appid,
		})

		if err != nil {
			return err
		}

		return http.Created(s, ctx, data, "/api/v1/apps/%s", appid)
	})
}

func (s *server) updateAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd update_app.Command) error {
		cmd.ID = ctx.Param("id")
		context := ctx.Request.Context()

		if _, err := bus.Send(s.bus, context, cmd); err != nil {
			return err
		}

		data, err := bus.Send(s.bus, context, get_app_detail.Query{
			ID: cmd.ID,
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, data)
	})
}

func (s *server) listAppsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		apps, err := bus.Send(s.bus, ctx.Request.Context(), get_apps.Query{})

		if err != nil {
			return err
		}

		return http.Ok(ctx, apps)
	})
}

func (s *server) getAppByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		app, err := bus.Send(s.bus, ctx.Request.Context(), get_app_detail.Query{
			ID: ctx.Param("id"),
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, app)
	})
}

// DNS, CAA and certificate status of an environment's custom domains (seelf#5). Only the
// domains saved on the application are checked, a few checks at a time per user.
func (s *server) checkAppDomainsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		uid, _ := domain.CurrentUser(ctx.Request.Context()).TryGet()
		release, err := s.domainChecksLimiter.Acquire(string(uid))

		if err != nil {
			ctx.AbortWithStatusJSON(nethttp.StatusTooManyRequests, gin.H{"code": err.Error()})
			return nil
		}

		defer release()

		result, err := bus.Send(s.bus, ctx.Request.Context(), check_app_domains.Query{
			AppID:       ctx.Param("id"),
			Environment: environmentOrProduction(ctx.Query("environment")),
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, result)
	})
}

func (s *server) requestAppCleanupHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		cmd := request_app_cleanup.Command{ID: ctx.Param("id")}

		// DELETE /apps/:id?backup_volumes=false removes the volumes without a backup
		if value, isSet := ctx.GetQuery("backup_volumes"); isSet {
			cmd.BackupVolumes.Set(value != "false" && value != "0")
		}

		if _, err := bus.Send(s.bus, ctx.Request.Context(), cmd); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}
