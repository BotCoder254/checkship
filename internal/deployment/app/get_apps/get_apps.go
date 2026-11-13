package get_apps

import (
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_deployments"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

type (
	// Retrieve all apps.
	Query struct {
		bus.Query[[]App]
	}

	App struct {
		ID                 string                                                `json:"id"`
		Name               string                                                `json:"name"`
		CleanupRequestedAt monad.Maybe[time.Time]                                `json:"cleanup_requested_at"`
		CleanupRequestedBy monad.Maybe[app.UserSummary]                          `json:"cleanup_requested_by"`
		CreatedAt          time.Time                                             `json:"created_at"`
		CreatedBy          app.UserSummary                                       `json:"created_by"`
		LatestDeployments  app.LatestDeployments[get_app_deployments.Deployment] `json:"latest_deployments"`
		ProductionTarget   app.TargetSummary                                     `json:"production_target"`
		StagingTarget      app.TargetSummary                                     `json:"staging_target"`
	}
)

func (Query) Name_() string { return "deployment.query.get_apps" }
