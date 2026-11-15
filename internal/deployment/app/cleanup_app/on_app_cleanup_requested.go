package cleanup_app

import (
	"context"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Queue the provider cleanup stuff for both production and staging for the app being deleted.
func OnAppCleanupRequestedHandler(scheduler bus.Scheduler, options ...Options) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		now := time.Now().UTC()
		backup := evt.BackupVolumes && (len(options) == 0 || options[0].BackupBeforeCleanup())

		for _, env := range []struct {
			name   domain.Environment
			config domain.EnvironmentConfig
		}{
			{domain.Production, evt.ProductionConfig},
			{domain.Staging, evt.StagingConfig},
		} {
			if err := scheduler.Queue(ctx, Command{
				AppID:       string(evt.ID),
				Environment: string(env.name),
				TargetID:    string(env.config.Target()),
				From:        env.config.Version(),
				To:          now,
				Backup:      backup,
				Trigger:     string(domain.BackupTriggerBeforeDelete),
				RequestedBy: string(evt.Requested.By()),
			}, bus.WithPolicy(bus.JobPolicyCancellable)); err != nil {
				return err
			}
		}

		return nil
	}
}
