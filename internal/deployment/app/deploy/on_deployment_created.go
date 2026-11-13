package deploy

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Upon receiving a deployment created event, queue a job to deploy the application.
func OnDeploymentCreatedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.DeploymentCreated] {
	return func(ctx context.Context, evt domain.DeploymentCreated) error {
		return scheduler.Queue(ctx, Command{
			AppID:            string(evt.ID.AppID()),
			DeploymentNumber: int(evt.ID.DeploymentNumber()),
		}, bus.WithGroup(app.DeploymentGroup(evt.Config)), bus.WithPolicy(bus.JobPolicyRetryPreserveOrder))
	}
}
