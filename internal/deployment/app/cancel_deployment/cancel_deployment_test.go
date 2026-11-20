package cancel_deployment_test

import (
	"context"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/cancel_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/running"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/spy"
)

func Test_CancelDeployment(t *testing.T) {

	arrange := func(tb testing.TB, registry domain.RunningDeployments, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, cancel_deployment.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return cancel_deployment.Handler(context.DeploymentsStore, context.DeploymentsStore, registry), context.Context, context.Dispatcher
	}

	seed := func(states ...func(*domain.Deployment)) ([]fixture.SeedBuilder, []domain.Deployment) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(fixture.WithDeploymentRequestedBy(user.ID()), fixture.FromApp(app))

		for _, apply := range states {
			apply(&deployment)
		}

		return []fixture.SeedBuilder{
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		}, []domain.Deployment{deployment}
	}

	command := func(d domain.Deployment) cancel_deployment.Command {
		return cancel_deployment.Command{
			AppID:            string(d.ID().AppID()),
			DeploymentNumber: int(d.ID().DeploymentNumber()),
		}
	}

	t.Run("should fail if the deployment does not exist", func(t *testing.T) {
		handler, ctx, _ := arrange(t, running.New())

		_, err := handler(ctx, cancel_deployment.Command{AppID: "nope", DeploymentNumber: 1})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should cancel a pending deployment", func(t *testing.T) {
		options, deployments := seed()
		handler, ctx, dispatcher := arrange(t, running.New(), options...)

		_, err := handler(ctx, command(deployments[0]))

		assert.Nil(t, err)
		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, domain.ErrDeploymentCancelled.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should stop a pending deployment a job has already picked up", func(t *testing.T) {
		options, deployments := seed()
		registry := running.New()
		jobCtx, release := registry.Track(context.Background(), deployments[0].ID())
		defer release()
		handler, ctx, _ := arrange(t, registry, options...)

		_, err := handler(ctx, command(deployments[0]))

		assert.Nil(t, err)
		assert.NotNil(t, jobCtx.Err())
	})

	t.Run("should stop a running deployment through its job", func(t *testing.T) {
		options, deployments := seed(func(d *domain.Deployment) { _ = d.HasStarted() })
		registry := running.New()
		jobCtx, release := registry.Track(context.Background(), deployments[0].ID())
		defer release()
		handler, ctx, dispatcher := arrange(t, registry, options...)

		_, err := handler(ctx, command(deployments[0]))

		assert.Nil(t, err)
		assert.NotNil(t, jobCtx.Err())
		assert.HasLength(t, 0, dispatcher.Signals()) // The job writes the final state
	})

	t.Run("should refuse a running deployment no job is processing", func(t *testing.T) {
		options, deployments := seed(func(d *domain.Deployment) { _ = d.HasStarted() })
		handler, ctx, _ := arrange(t, running.New(), options...)

		_, err := handler(ctx, command(deployments[0]))

		assert.ErrorIs(t, domain.ErrDeploymentNotCancellable, err)
	})

	t.Run("should refuse a finished deployment", func(t *testing.T) {
		options, deployments := seed(func(d *domain.Deployment) {
			_ = d.HasStarted()
			_ = d.HasEnded(domain.Services{}, nil)
		})
		handler, ctx, dispatcher := arrange(t, running.New(), options...)

		_, err := handler(ctx, command(deployments[0]))

		assert.ErrorIs(t, domain.ErrDeploymentNotCancellable, err)
		assert.HasLength(t, 0, dispatcher.Signals())
	})
}
