package infra

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/live"
)

// Publish a live hint for every change the dashboard shows. Signal handlers run inside the write
// transaction, before it commits: publishing never blocks and never fails, so a slow client can't
// hold or roll back a write. A hint from a write that is then rolled back only causes a refetch.
//
// Some changes send no signal (deployments failed in bulk at startup, the cascade when an app is
// deleted), which is why clients also refresh on a timer.
func RegisterLiveHints(b bus.Bus, publish func(live.Hint)) {
	deployments := func(app domain.AppID) error {
		publish(live.Hint{Topic: "deployments", ID: string(app)})
		return nil
	}
	targets := func(id domain.TargetID) error {
		publish(live.Hint{Topic: "targets", ID: string(id)})
		return nil
	}
	apps := func(id domain.AppID) error {
		publish(live.Hint{Topic: "apps", ID: string(id)})
		return nil
	}

	bus.On(b, func(_ context.Context, e domain.DeploymentCreated) error { return deployments(e.ID.AppID()) })
	bus.On(b, func(_ context.Context, e domain.DeploymentStateChanged) error { return deployments(e.ID.AppID()) })
	bus.On(b, func(_ context.Context, e domain.TargetCreated) error { return targets(e.ID) })
	bus.On(b, func(_ context.Context, e domain.TargetStateChanged) error { return targets(e.ID) })
	bus.On(b, func(_ context.Context, e domain.TargetDeleted) error { return targets(e.ID) })
	bus.On(b, func(_ context.Context, e domain.AppCreated) error { return apps(e.ID) })
	bus.On(b, func(_ context.Context, e domain.AppCleanupRequested) error { return apps(e.ID) })
	bus.On(b, func(_ context.Context, e domain.AppDeleted) error { return apps(e.ID) })
}
