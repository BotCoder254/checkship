package infra

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/log"
)

// Queues a configuration of each target whose proxy is outdated. The configure job is the
// usual one (merged with any pending configuration); a target being configured right now picks
// the new configuration up anyway, since the job runs this version of checkship.
func ReconfigureOutdatedProxies(ctx context.Context, targets sqlite.TargetsStore, logger log.Logger) error {
	outdated, err := targets.GetOutdatedProxies(ctx, sqlite.ProxyRevision)

	if err != nil {
		return err
	}

	for _, target := range outdated {
		if err := target.Reconfigure(); err == nil {
			if err = targets.Write(ctx, &target); err != nil {
				return err
			}

			logger.Infow("reconfiguring the proxy of a target to apply a newer configuration",
				"target", target.ID(), "revision", sqlite.ProxyRevision)
		}

		if err := targets.MarkProxyRevision(ctx, target.ID(), sqlite.ProxyRevision); err != nil {
			return err
		}
	}

	return nil
}
