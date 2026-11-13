package fail_pending_deployments

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

func OnAppEnvChangedHandler(writer domain.DeploymentsWriter) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		if !evt.TargetHasChanged() {
			return nil
		}

		return writer.FailDeployments(ctx, domain.ErrAppTargetChanged, domain.FailCriteria{
			Status:      monad.Value(domain.DeploymentStatusPending),
			App:         monad.Value(evt.ID),
			Environment: monad.Value(evt.Environment),
			Target:      monad.Value(evt.OldConfig.Target()),
		})
	}
}
