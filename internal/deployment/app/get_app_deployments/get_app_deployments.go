package get_app_deployments

import (
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

type (
	Query struct {
		// Retrieve all deployments for an app.
		bus.Query[storage.Paginated[Deployment]]

		AppID       string              `json:"-"`
		Page        monad.Maybe[int]    `form:"page"`
		Environment monad.Maybe[string] `form:"environment"`
		Status      monad.Maybe[uint8]  `form:"status"`
		// Matches a deployment number (with or without #) or a Git branch or commit.
		Search  monad.Maybe[string] `form:"search"`
		PerPage monad.Maybe[int]    `form:"per_page"`
	}

	Deployment struct {
		AppID            string                       `json:"app_id"`
		DeploymentNumber int                          `json:"deployment_number"`
		Environment      string                       `json:"environment"`
		Target           get_deployment.TargetSummary `json:"target"`
		Source           get_deployment.Source        `json:"source"`
		State            State                        `json:"state"`
		RequestedAt      time.Time                    `json:"requested_at"`
		RequestedBy      app.UserSummary              `json:"requested_by"`
	}

	State struct {
		Status     uint8                  `json:"status"`
		ErrCode    monad.Maybe[string]    `json:"error_code"`
		StartedAt  monad.Maybe[time.Time] `json:"started_at"`
		FinishedAt monad.Maybe[time.Time] `json:"finished_at"`
	}
)

func (Query) Name_() string { return "deployment.query.get_app_deployments" }
