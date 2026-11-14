package sqlite_test

import (
	"testing"
	"time"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_dashboard"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	_ "github.com/checkship-deploy/checkship/internal/deployment/infra/source/raw"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

func Test_Dashboard(t *testing.T) {
	now := time.Now().UTC()
	user := authfixture.User()
	target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
	config := domain.NewEnvironmentConfig(target.ID())
	api := fixture.App(fixture.WithAppName("api"), fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(config, config))
	blog := fixture.App(fixture.WithAppName("blog"), fixture.WithAppCreatedBy(user.ID()), fixture.WithEnvironmentConfig(config, config))

	ctx := fixture.PrepareDatabase(t,
		fixture.WithUsers(&user),
		fixture.WithTargets(&target),
		fixture.WithApps(&api, &blog),
	)
	gateway := sqlite.NewGateway(ctx.DB)

	insert := func(app domain.App, number int, env domain.Environment, status domain.DeploymentStatus, at time.Time) {
		t.Helper()
		var errcode monad.Maybe[string]
		if status == domain.DeploymentStatusFailed {
			errcode.Set("compose_up_failed")
		}
		if _, err := ctx.DB.ExecContext(ctx.Context, `
			INSERT INTO deployments (app_id, deployment_number, config_appid, config_appname, config_environment,
				config_target, state_status, state_errcode, state_started_at, state_finished_at,
				source_discriminator, source, requested_at, requested_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'raw', 'services: {}', ?, ?)`,
			app.ID(), number, app.ID(), "app", env, target.ID(), status, errcode,
			at, at.Add(30*time.Second), at, user.ID(),
		); err != nil {
			t.Fatal(err)
		}
	}

	// Ten api deployments an hour apart, the latest failed; two blog deployments; one 40 days ago.
	for i := 1; i <= 10; i++ {
		status := domain.DeploymentStatusSucceeded
		if i == 10 {
			status = domain.DeploymentStatusFailed
		}
		insert(api, i, domain.Production, status, now.Add(-time.Duration(11-i)*time.Hour))
	}
	insert(blog, 1, domain.Staging, domain.DeploymentStatusSucceeded, now.Add(-30*time.Minute))
	insert(blog, 2, domain.Production, domain.DeploymentStatusSucceeded, now.Add(-40*24*time.Hour))

	t.Run("summary reads the range and the previous one", func(t *testing.T) {
		s, err := gateway.GetDashboardSummary(ctx.Context, get_dashboard.SummaryQuery{
			Filters: get_dashboard.Filters{Days: 30, Until: now.Add(time.Second)},
		})

		assert.Nil(t, err)
		assert.Equal(t, 2, s.Apps)
		assert.Equal(t, 11, s.Current.Deployments)
		assert.Equal(t, 1, s.Current.Failed)
		assert.Equal(t, 1, s.Previous.Deployments)
		assert.Equal(t, 30.0, s.Current.MedianDuration.MustGet())
	})

	t.Run("summary filters by environment", func(t *testing.T) {
		s, err := gateway.GetDashboardSummary(ctx.Context, get_dashboard.SummaryQuery{
			Filters: get_dashboard.Filters{Days: 7, Until: now.Add(time.Second), Environment: monad.Value("staging")},
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, s.Current.Deployments)
		assert.Equal(t, "staging", s.Recovery.Environment)
	})

	t.Run("apps are ranked and paginated", func(t *testing.T) {
		page, err := gateway.GetDashboardApps(ctx.Context, get_dashboard.AppsQuery{
			Filters: get_dashboard.Filters{Days: 7, Until: now.Add(time.Second)},
			Sort:    "failures",
			PerPage: 1,
		})

		assert.Nil(t, err)
		assert.Equal(t, 2, page.Total)
		assert.HasLength(t, 1, page.Data)
		assert.Equal(t, "api", page.Data[0].Name)
		assert.Equal(t, 1, page.Data[0].Failed)
		assert.False(t, page.IsLastPage)
	})

	t.Run("activity pages with a cursor, without gaps or repeats while rows arrive", func(t *testing.T) {
		first, err := gateway.GetDashboardActivity(ctx.Context, get_dashboard.ActivityQuery{Limit: 5})

		assert.Nil(t, err)
		assert.HasLength(t, 5, first.Data)
		assert.Equal(t, blog.ID(), domain.AppID(first.Data[0].AppID))
		assert.Equal(t, 10, first.Data[1].DeploymentNumber)

		// A new deployment arrives before the next page is read.
		insert(api, 11, domain.Production, domain.DeploymentStatusPending, now)

		cursor, err := get_dashboard.DecodeCursor(first.NextCursor.MustGet())
		assert.Nil(t, err)

		second, err := gateway.GetDashboardActivity(ctx.Context, get_dashboard.ActivityQuery{Limit: 5, After: monad.Value(cursor)})

		assert.Nil(t, err)
		assert.HasLength(t, 5, second.Data)
		assert.Equal(t, 6, second.Data[0].DeploymentNumber) // right after api #7, the last of page one

		cursor, _ = get_dashboard.DecodeCursor(second.NextCursor.MustGet())
		third, err := gateway.GetDashboardActivity(ctx.Context, get_dashboard.ActivityQuery{Limit: 5, After: monad.Value(cursor)})

		assert.Nil(t, err)
		assert.HasLength(t, 2, third.Data) // api #1 and blog #2
		assert.False(t, third.NextCursor.HasValue())
	})

	t.Run("activity filters by status and application", func(t *testing.T) {
		failed, err := gateway.GetDashboardActivity(ctx.Context, get_dashboard.ActivityQuery{
			Status: monad.Value(uint8(domain.DeploymentStatusFailed)),
			AppID:  monad.Value(string(api.ID())),
		})

		assert.Nil(t, err)
		assert.HasLength(t, 1, failed.Data)
		assert.Equal(t, 10, failed.Data[0].DeploymentNumber)
	})

	t.Run("activity can be limited to a range", func(t *testing.T) {
		recent, err := gateway.GetDashboardActivity(ctx.Context, get_dashboard.ActivityQuery{
			AppID: monad.Value(string(blog.ID())),
			Since: monad.Value(now.Add(-24 * time.Hour)),
		})

		assert.Nil(t, err)
		assert.HasLength(t, 1, recent.Data) // blog #2 is 40 days old
	})
}
