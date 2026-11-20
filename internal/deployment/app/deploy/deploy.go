package deploy

import (
	"context"
	"errors"
	"strconv"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Process a deployment, this is where the magic happen!
type Command struct {
	bus.Command[bus.UnitType]

	AppID            string `json:"app_id"`
	DeploymentNumber int    `json:"deployment_number"`
}

func (Command) Name_() string        { return "deployment.command.deploy" }
func (c Command) ResourceID() string { return c.AppID + "-" + strconv.Itoa(c.DeploymentNumber) }

// Handle the deployment process.
// If an unexpected error occurs during this process, it uses the bus.PreserveOrder function
// to make sure all deployments are processed linearly.
//
// The deployment is tracked in running for as long as the job runs, so it can be cancelled:
// the context given to the source and the provider is then cancelled and the deployment ends
// with domain.ErrDeploymentCancelled.
func Handler(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	artifactManager domain.ArtifactManager,
	source domain.Source,
	provider domain.Provider,
	targetsReader domain.TargetsReader,
	registriesReader domain.RegistriesReader,
	running domain.RunningDeployments,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(jobCtx context.Context, cmd Command) (result bus.UnitType, finalErr error) {
		result = bus.Unit

		id := domain.DeploymentIDFrom(
			domain.AppID(cmd.AppID),
			domain.DeploymentNumber(cmd.DeploymentNumber),
		)

		// Tracked before anything is read, so a cancel arriving while the job starts always
		// reaches it. ctx is cancelled on cancel; the deployment state itself is always read and
		// written with persistCtx, otherwise the final write would fail and the job be retried.
		ctx, release := running.Track(jobCtx, id)
		defer release()

		persistCtx := context.WithoutCancel(ctx)

		depl, err := reader.GetByID(ctx, id)

		if err != nil {
			// Deployment does not exist anymore, the app should have been deleted, return early
			if errors.Is(err, apperr.ErrNotFound) {
				return result, nil
			}

			return result, err
		}

		// If the target does not exist, fail the deployment
		// If the target is not ready, returns early without starting the deployment
		target, targetErr := targetsReader.GetByID(ctx, depl.Config().Target())

		if targetErr != nil && !errors.Is(targetErr, apperr.ErrNotFound) {
			return result, targetErr
		}

		if err = depl.HasStarted(); err != nil {
			// If the deployment could not be started, it probably means the
			// application has been requested for cleanup and the deployment has been
			// cancelled, so the deploy job will never succeed.
			return result, nil
		}

		var targetAvailabilityErr error

		if targetErr == nil {
			targetAvailabilityErr = target.CheckAvailability()

			// Target configuration is in progress, just retry the job later without writing
			// the deployment, keep it in pending state
			if errors.Is(targetAvailabilityErr, domain.ErrTargetConfigurationInProgress) {
				return result, targetAvailabilityErr
			}
		}

		// Cancelled while the job was starting: the cancel command has already marked the
		// deployment as cancelled, don't start it.
		if ctx.Err() != nil {
			return result, nil
		}

		if err = writer.Write(ctx, &depl); err != nil {
			return result, err
		}

		var (
			deploymentCtx domain.DeploymentContext
			services      domain.Services
			registries    []domain.Registry
		)

		// This one is a special case to avoid to avoid many branches
		// checking for errors when writing the domain.
		// Based on wether or not there was an error, it will update the deployment
		// accordingly.
		defer func() {
			// A cancel stops whatever step was running with an error of its own (context
			// canceled, compose_failed...): record why it really stopped. A deployment which
			// managed to finish anyway keeps its success.
			if finalErr != nil && wasCancelled(ctx) {
				finalErr = domain.ErrDeploymentCancelled
			}

			// Since the deployment process could take some time, retrieve a fresh version of the
			// deployment right now
			if depl, err = reader.GetByID(persistCtx, depl.ID()); err != nil {
				if errors.Is(err, apperr.ErrNotFound) {
					finalErr = nil
				} else {
					finalErr = err
				}
				return
			}

			// An error means it has already been handled
			if err = depl.HasEnded(services, finalErr); err != nil {
				finalErr = nil
				return
			}

			if err = writer.Write(persistCtx, &depl); err != nil {
				finalErr = err
				return
			}

			finalErr = nil
		}()

		// Prepare the build directory
		if deploymentCtx, finalErr = artifactManager.PrepareBuild(ctx, depl); finalErr != nil {
			return
		}

		defer deploymentCtx.Logger().Close()

		// Runs before the logger is closed: the log says why it stopped.
		defer func() {
			if finalErr != nil && wasCancelled(ctx) {
				deploymentCtx.Logger().Warnf("deployment cancelled, stopping")
			}
		}()

		// If the target does not exist, let's fail the deployment correctly
		if targetErr != nil {
			finalErr = targetErr
			return
		}

		// Target not available, fail the deployment
		if targetAvailabilityErr != nil {
			finalErr = targetAvailabilityErr
			return
		}

		// Fetch deployment files
		if finalErr = source.Fetch(ctx, deploymentCtx, depl); finalErr != nil {
			return
		}

		// Fetch custom registries
		if registries, finalErr = registriesReader.GetAll(ctx); finalErr != nil {
			return
		}

		// Ask the provider to actually deploy the app
		if services, finalErr = provider.Deploy(ctx, deploymentCtx, depl, target, registries); finalErr != nil {
			return
		}

		return
	}
}

func wasCancelled(ctx context.Context) bool {
	return errors.Is(context.Cause(ctx), domain.ErrDeploymentCancelled)
}
