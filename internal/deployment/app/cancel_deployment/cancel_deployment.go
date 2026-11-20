package cancel_deployment

import (
	"context"
	"errors"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Cancel a deployment which is pending or running (seelf#7).
//
// A pending deployment is marked as cancelled right away: its queued job is removed by the
// on_deployment_failed_cleanup_jobs trigger and the deployments queued after it run as usual.
// A running deployment is stopped by its job, which ends it as cancelled within seconds.
// Anything else returns domain.ErrDeploymentNotCancellable.
type Command struct {
	bus.Command[bus.UnitType]

	AppID            string `json:"-"`
	DeploymentNumber int    `json:"-"`
}

func (Command) Name_() string { return "deployment.command.cancel_deployment" }

func Handler(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	running domain.RunningDeployments,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		depl, err := reader.GetByID(ctx, domain.DeploymentIDFrom(
			domain.AppID(cmd.AppID),
			domain.DeploymentNumber(cmd.DeploymentNumber),
		))

		if err != nil {
			return bus.Unit, err
		}

		err = depl.Cancel()

		// Not pending anymore: only a deployment a job is processing can still be stopped.
		if errors.Is(err, domain.ErrDeploymentNotCancellable) {
			if running.Cancel(depl.ID()) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		if err != nil {
			return bus.Unit, err
		}

		if err = writer.Write(ctx, &depl); err != nil {
			return bus.Unit, err
		}

		// Written first, so a job which already picked the deployment up and sees its context
		// cancelled finds it cancelled. Most of the time no job has it yet.
		running.Cancel(depl.ID())

		return bus.Unit, nil
	}
}
