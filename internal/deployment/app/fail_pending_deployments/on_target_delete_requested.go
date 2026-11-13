package fail_pending_deployments

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

func OnTargetDeleteRequestedHandler(writer domain.DeploymentsWriter) bus.SignalHandler[domain.TargetCleanupRequested] {
	return func(ctx context.Context, evt domain.TargetCleanupRequested) error {
		return writer.FailDeployments(ctx, domain.ErrTargetCleanupRequested, domain.FailCriteria{
			Status: monad.Value(domain.DeploymentStatusPending),
			Target: monad.Value(evt.ID),
		})
	}
}
