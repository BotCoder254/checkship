package get_app_logs_test

import (
	"context"
	"testing"
	"time"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_logs"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

func Test_GetAppLogs(t *testing.T) {
	arrange := func(tb testing.TB, provider domain.Provider, seed ...fixture.SeedBuilder) func(context.Context, get_app_logs.Query) (domain.LogsStream, error) {
		ctx := fixture.PrepareDatabase(tb, seed...)
		return get_app_logs.Handler(ctx.AppsStore, ctx.TargetsStore, provider)
	}

	t.Run("should reject an unknown environment", func(t *testing.T) {
		handler := arrange(t, &logsProvider{})

		_, err := handler(context.Background(), get_app_logs.Query{AppID: "app", Environment: "qa"})

		assert.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
	})

	t.Run("should return not found for an unknown application", func(t *testing.T) {
		handler := arrange(t, &logsProvider{})

		_, err := handler(context.Background(), get_app_logs.Query{AppID: "app", Environment: "production"})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should stream the logs of the environment's target", func(t *testing.T) {
		user := authfixture.User()
		production := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		staging := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(
			domain.NewEnvironmentConfig(production.ID()),
			domain.NewEnvironmentConfig(staging.ID()),
		))
		provider := &logsProvider{}
		handler := arrange(t, provider, fixture.WithUsers(&user), fixture.WithTargets(&production, &staging), fixture.WithApps(&app))
		since := time.Date(2026, 11, 14, 23, 41, 2, 0, time.UTC)

		stream, err := handler(context.Background(), get_app_logs.Query{
			AppID:       string(app.ID()),
			Environment: "staging",
			Tail:        200,
			Since:       monad.Value(since),
		})

		assert.Nil(t, err)
		assert.False(t, provider.called, "nothing runs before the stream is started")

		assert.Nil(t, stream(context.Background(), nil))
		assert.True(t, provider.called)
		assert.Equal(t, staging.ID(), provider.target)
		assert.Equal(t, app.ID(), provider.app)
		assert.Equal(t, domain.Staging, provider.env)
		assert.Equal(t, 200, provider.options.Tail)
		assert.Equal(t, since, provider.options.Since.Get(time.Time{}))
	})
}

type logsProvider struct {
	domain.Provider
	called  bool
	target  domain.TargetID
	app     domain.AppID
	env     domain.Environment
	options domain.LogsOptions
}

func (p *logsProvider) Logs(_ context.Context, target domain.Target, app domain.AppID, env domain.Environment, options domain.LogsOptions, _ domain.LogsConsumer) error {
	p.called = true
	p.target = target.ID()
	p.app = app
	p.env = env
	p.options = options
	return nil
}
