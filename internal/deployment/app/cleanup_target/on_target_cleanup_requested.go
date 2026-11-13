package cleanup_target

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

func OnTargetCleanupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.TargetCleanupRequested] {
	return func(ctx context.Context, evt domain.TargetCleanupRequested) error {
		return scheduler.Queue(ctx, Command{
			ID: string(evt.ID),
		}, bus.WithPolicy(bus.JobPolicyCancellable))
	}
}
