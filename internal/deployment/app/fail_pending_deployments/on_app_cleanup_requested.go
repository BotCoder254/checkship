package fail_pending_deployments

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

// When an app is about to be deleted, cancel all pending deployments
func OnAppCleanupRequestedHandler(writer domain.DeploymentsWriter) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		return writer.FailDeployments(ctx, domain.ErrAppCleanupRequested, domain.FailCriteria{
			Status: monad.Value(domain.DeploymentStatusPending),
			App:    monad.Value(evt.ID),
		})
	}
}
