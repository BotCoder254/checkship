package get_app_services_test

import (
	"context"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_services"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_GetAppServices(t *testing.T) {
	arrange := func(tb testing.TB, provider domain.Provider, seed ...fixture.SeedBuilder) func(context.Context, get_app_services.Query) ([]domain.ServiceState, error) {
		ctx := fixture.PrepareDatabase(tb, seed...)
		return get_app_services.Handler(ctx.AppsStore, ctx.TargetsStore, provider)
	}

	t.Run("should return not found for an unknown application", func(t *testing.T) {
		handler := arrange(t, &servicesProvider{})

		_, err := handler(context.Background(), get_app_services.Query{AppID: "app", Environment: "production"})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should return the states of the environment's containers", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(
			domain.NewEnvironmentConfig(target.ID()),
			domain.NewEnvironmentConfig(target.ID()),
		))
		provider := &servicesProvider{states: []domain.ServiceState{{Service: "web", State: "running"}}}
		handler := arrange(t, provider, fixture.WithUsers(&user), fixture.WithTargets(&target), fixture.WithApps(&app))

		states, err := handler(context.Background(), get_app_services.Query{AppID: string(app.ID()), Environment: "production"})

		assert.Nil(t, err)
		assert.DeepEqual(t, provider.states, states)
		assert.Equal(t, target.ID(), provider.target)
		assert.Equal(t, domain.Production, provider.env)
	})
}

type servicesProvider struct {
	domain.Provider
	states []domain.ServiceState
	target domain.TargetID
	env    domain.Environment
}

func (p *servicesProvider) Services(_ context.Context, target domain.Target, _ domain.AppID, env domain.Environment) ([]domain.ServiceState, error) {
	p.target = target.ID()
	p.env = env
	return p.states, nil
}
