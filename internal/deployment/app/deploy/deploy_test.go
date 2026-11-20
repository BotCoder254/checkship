package deploy_test

import (
	"context"
	"errors"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/deploy"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/artifact"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/running"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source/raw"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/spy"
	"github.com/checkship-deploy/checkship/pkg/log"
)

func arrangeWithRunning(
	tb testing.TB,
	source domain.Source,
	provider domain.Provider,
	seed ...fixture.SeedBuilder,
) (
	bus.RequestHandler[bus.UnitType, deploy.Command],
	context.Context,
	spy.Dispatcher,
	domain.RunningDeployments,
) {
	context := fixture.PrepareDatabase(tb, seed...)
	logger, _ := log.NewLogger()
	artifactManager := artifact.NewLocal(context.Config, logger)
	registry := running.New()
	return deploy.Handler(context.DeploymentsStore, context.DeploymentsStore, artifactManager, source, provider, context.TargetsStore, context.RegistriesStore, registry), context.Context, context.Dispatcher, registry
}

func Test_Deploy(t *testing.T) {

	arrange := func(
		tb testing.TB,
		source domain.Source,
		provider domain.Provider,
		seed ...fixture.SeedBuilder,
	) (
		bus.RequestHandler[bus.UnitType, deploy.Command],
		context.Context,
		spy.Dispatcher,
	) {
		handler, ctx, dispatcher, _ := arrangeWithRunning(tb, source, provider, seed...)
		return handler, ctx, dispatcher
	}

	t.Run("should fail silently if the deployment does not exists", func(t *testing.T) {
		handler, ctx, _ := arrange(t, source(nil), provider(nil))

		r, err := handler(ctx, deploy.Command{})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)
	})

	t.Run("should mark the deployment has failed if the target is configuring", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		handler, ctx, dispatcher := arrange(t, source(nil), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		_, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		assert.HasLength(t, 0, dispatcher.Signals())
	})

	t.Run("should mark the deployment has failed if source does not succeed", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		sourceErr := errors.New("source_failed")
		handler, ctx, dispatcher := arrange(t, source(sourceErr), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, sourceErr.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should mark the deployment has failed in the target is not correctly configured", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, errors.New("target_failed"))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		handler, ctx, dispatcher := arrange(t, source(nil), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, domain.ErrTargetConfigurationFailed.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should mark the deployment has failed if provider does not run the deployment successfully", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		providerErr := errors.New("provider_failed")
		handler, ctx, dispatcher := arrange(t, source(nil), provider(providerErr),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, providerErr.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should mark the deployment has succeeded if all is good", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		handler, ctx, dispatcher := arrange(t, source(nil), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusSucceeded, changed.State.Status())
	})

	t.Run("should end a deployment cancelled while it runs as cancelled, without retrying the job", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		started := make(chan struct{})
		handler, ctx, dispatcher, registry := arrangeWithRunning(t, source(nil), &blockingProvider{started: started},
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		go func() {
			<-started
			registry.Cancel(deployment.ID())
		}()

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)
		assert.False(t, registry.Cancel(deployment.ID()), "the deployment should not be tracked anymore")

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, domain.ErrDeploymentCancelled.Error(), changed.State.ErrCode().MustGet())
	})
}

// Blocks in Deploy until its context is cancelled, like compose pulling a large image.
type blockingProvider struct {
	dummyProvider
	started chan struct{}
}

func (b *blockingProvider) Deploy(ctx context.Context, _ domain.DeploymentContext, _ domain.Deployment, _ domain.Target, _ []domain.Registry) (domain.Services, error) {
	close(b.started)
	<-ctx.Done()
	return nil, errors.New("compose_failed")
}

type dummySource struct {
	err error
}

func source(failedWithErr error) domain.Source {
	return &dummySource{failedWithErr}
}

func (*dummySource) Prepare(context.Context, domain.App, any) (domain.SourceData, error) {
	return raw.Data(""), nil
}

func (t *dummySource) Fetch(context.Context, domain.DeploymentContext, domain.Deployment) error {
	return t.err
}

type dummyProvider struct {
	domain.Provider
	err error
}

func provider(failedWithErr error) domain.Provider {
	return &dummyProvider{
		err: failedWithErr,
	}
}

func (b *dummyProvider) Prepare(context.Context, any, ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	return nil, nil
}

func (b *dummyProvider) Deploy(context.Context, domain.DeploymentContext, domain.Deployment, domain.Target, []domain.Registry) (domain.Services, error) {
	return domain.Services{}, b.err
}
