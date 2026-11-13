package cleanup_app

import (
	"context"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

func OnAppEnvChangedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		if !evt.TargetHasChanged() {
			return nil
		}

		return scheduler.Queue(ctx, Command{
			AppID:       string(evt.ID),
			TargetID:    string(evt.OldConfig.Target()),
			Environment: string(evt.Environment),
			From:        evt.OldConfig.Version(),
			To:          time.Now().UTC(),
		}, bus.WithPolicy(bus.JobPolicyCancellable))
	}
}
