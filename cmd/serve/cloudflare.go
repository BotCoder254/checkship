package serve

import (
	"github.com/checkship-deploy/checkship/internal/deployment/app/cloudflare_setup"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/http"
	"github.com/gin-gonic/gin"
)

// Connect Cloudflare, per target. Session only: the token gives access to the person's
// Cloudflare account, so the API key can't set it.

func (s *server) cloudflareStatusHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		status, err := bus.Send(s.bus, ctx.Request.Context(), cloudflare_setup.StatusQuery{TargetID: ctx.Param("id")})

		if err != nil {
			return err
		}

		return http.Ok(ctx, status)
	})
}

func (s *server) connectCloudflareHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd cloudflare_setup.ConnectCommand) error {
		cmd.TargetID = ctx.Param("id")
		status, err := bus.Send(s.bus, ctx.Request.Context(), cmd)

		if err != nil {
			return err
		}

		return http.Ok(ctx, status)
	})
}

func (s *server) setupCloudflareHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, cmd cloudflare_setup.SyncCommand) error {
		cmd.TargetID = ctx.Param("id")
		result, err := bus.Send(s.bus, ctx.Request.Context(), cmd)

		if err != nil {
			return err
		}

		return http.Ok(ctx, result)
	})
}

func (s *server) disconnectCloudflareHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		if _, err := bus.Send(s.bus, ctx.Request.Context(), cloudflare_setup.DisconnectCommand{TargetID: ctx.Param("id")}); err != nil {
			return err
		}

		return http.NoContent(ctx)
	})
}
