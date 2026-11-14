package serve

import (
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_deployments"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment_log"
	"github.com/checkship-deploy/checkship/internal/deployment/app/promote"
	"github.com/checkship-deploy/checkship/internal/deployment/app/queue_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/app/redeploy"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source/git"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/http"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/gin-gonic/gin"
)

// Specific body for the queue deployment endpoint. Will resolve to the appropriate payload
// based on the provided fields.
type queueDeploymentBody struct {
	queue_deployment.Command

	Raw     monad.Maybe[string]   `json:"raw"`
	Archive *multipart.FileHeader `form:"archive"`
	Git     monad.Maybe[git.Body] `json:"git"`
}

func (s *server) queueDeploymentHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, body queueDeploymentBody) error {
		var context = ctx.Request.Context()

		body.AppID = ctx.Param("id")

		// Resolve the payload data.
		if gitBody, isSet := body.Git.TryGet(); isSet {
			body.Source = gitBody
		} else if rawBody, isSet := body.Raw.TryGet(); isSet {
			body.Source = rawBody
		} else if body.Archive != nil {
			body.Source = body.Archive
		}

		number, err := bus.Send(s.bus, context, body.Command)

		if err != nil {
			return err
		}

		return s.sendDeploymentCreatedResponse(ctx, body.AppID, number)
	})
}

func (s *server) redeployHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		var (
			appid     = ctx.Param("id")
			source, _ = strconv.Atoi(ctx.Param("number"))
		)

		number, err := bus.Send(s.bus, ctx.Request.Context(), redeploy.Command{
			AppID:            appid,
			DeploymentNumber: source,
		})

		if err != nil {
			return err
		}

		return s.sendDeploymentCreatedResponse(ctx, appid, number)
	})
}

func (s *server) promoteHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		var (
			appid     = ctx.Param("id")
			source, _ = strconv.Atoi(ctx.Param("number"))
		)

		number, err := bus.Send(s.bus, ctx.Request.Context(), promote.Command{
			AppID:            appid,
			DeploymentNumber: source,
		})

		if err != nil {
			return err
		}

		return s.sendDeploymentCreatedResponse(ctx, appid, number)
	})
}

func (s *server) getDeploymentByIDHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		number, _ := strconv.Atoi(ctx.Param("number"))

		deployment, err := bus.Send(s.bus, ctx.Request.Context(), get_deployment.Query{
			AppID:            ctx.Param("id"),
			DeploymentNumber: number,
		})

		if err != nil {
			return err
		}

		return http.Ok(ctx, deployment)
	})
}

func (s *server) getDeploymentLogsHandler() gin.HandlerFunc {
	return http.Send(s, func(ctx *gin.Context) error {
		appid := ctx.Param("id")
		number, _ := strconv.Atoi(ctx.Param("number"))

		logpath, err := bus.Send(s.bus, ctx.Request.Context(), get_deployment_log.Query{
			AppID:            appid,
			DeploymentNumber: number,
		})

		if err != nil {
			return err
		}

		return http.File(ctx, logpath)
	})
}

// FIXME: till gin support custom types in query binding...
type getDeploymentsFilters struct {
	Page        int    `form:"page"`
	Environment string `form:"environment"`
	Status      string `form:"status"`
	Search      string `form:"search"`
	PerPage     int    `form:"per_page"`
}

const maxDeploymentsPerPage = 50

func (s *server) listDeploymentsByAppHandler() gin.HandlerFunc {
	return http.Bind(s, func(ctx *gin.Context, request getDeploymentsFilters) error {
		query := get_app_deployments.Query{
			AppID: ctx.Param("id"),
		}

		if request.Environment != "" {
			query.Environment.Set(request.Environment)
		}

		if request.Page != 0 {
			query.Page.Set(request.Page)
		}

		if status, err := strconv.ParseUint(request.Status, 10, 8); err == nil {
			query.Status.Set(uint8(status))
		}

		if search := strings.TrimSpace(request.Search); search != "" {
			query.Search.Set(search)
		}

		if request.PerPage > 0 {
			query.PerPage.Set(min(request.PerPage, maxDeploymentsPerPage))
		}

		deployments, err := bus.Send(s.bus, ctx.Request.Context(), query)

		if err != nil {
			return err
		}

		return http.Ok(ctx, deployments)
	})
}

func (s *server) sendDeploymentCreatedResponse(ctx *gin.Context, appid string, number int) error {
	deployment, err := bus.Send(s.bus, ctx.Request.Context(), get_deployment.Query{
		AppID:            appid,
		DeploymentNumber: number,
	})

	if err != nil {
		return err
	}

	return http.Created(s, ctx, deployment, "/api/v1/apps/%s/deployments/%d", appid, number)
}
