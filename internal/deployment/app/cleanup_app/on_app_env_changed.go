package cleanup_app

import (
	"context"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Settings of the backups taken before checkship removes volumes.
type Options interface {
	BackupBeforeCleanup() bool
}

func OnAppEnvChangedHandler(scheduler bus.Scheduler, options ...Options) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		if !evt.TargetHasChanged() {
			return nil
		}

		// Moving the data needs a backup; otherwise the setting decides
		backup := evt.MoveVolumes || (len(options) > 0 && options[0].BackupBeforeCleanup())

		return scheduler.Queue(ctx, Command{
			AppID:       string(evt.ID),
			TargetID:    string(evt.OldConfig.Target()),
			Environment: string(evt.Environment),
			From:        evt.OldConfig.Version(),
			To:          time.Now().UTC(),
			Backup:      backup,
			Trigger:     string(domain.BackupTriggerBeforeTargetChange),
			MoveVolumes: evt.MoveVolumes,
			RequestedBy: string(evt.ChangedBy.Get("")),
		}, bus.WithPolicy(bus.JobPolicyCancellable))
	}
}
