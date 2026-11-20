package get_dashboard_test

import (
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app/get_dashboard"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

var until = time.Date(2026, 11, 14, 15, 0, 0, 0, time.UTC)

func row(app, env string, status domain.DeploymentStatus, at time.Time, seconds int) get_dashboard.Row {
	r := get_dashboard.Row{AppID: app, Environment: env, Status: uint8(status), RequestedAt: at}
	if seconds > 0 {
		r.StartedAt.Set(at)
		r.FinishedAt.Set(at.Add(time.Duration(seconds) * time.Second))
	}
	if status == domain.DeploymentStatusFailed {
		r.ErrCode.Set("compose_up_failed")
	}
	return r
}

func Test_Filters(t *testing.T) {
	t.Run("should fall back to 30 days and clamp the offset", func(t *testing.T) {
		f := get_dashboard.Filters{Days: 12, Offset: 5000, Until: until, Environment: monad.Value("nope")}.Normalized()

		assert.Equal(t, 30, f.Days)
		assert.Equal(t, get_dashboard.MaxOffset, f.Offset)
		assert.False(t, f.Environment.HasValue())
	})

	t.Run("should cover today and the days before it in the viewer's time zone", func(t *testing.T) {
		// 15:00 UTC is already the 15th in UTC+10.
		previousFrom, from, to := get_dashboard.Filters{Days: 7, Offset: 600, Until: until}.Normalized().Bounds()

		assert.Equal(t, time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC), from)
		assert.Equal(t, time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC), previousFrom)
		assert.Equal(t, until, to)
	})
}

func Test_Summarize(t *testing.T) {
	day := func(d, h int) time.Time { return time.Date(2026, 11, d, h, 0, 0, 0, time.UTC) }
	rows := []get_dashboard.Row{
		// Previous period (1–7 Nov for 7 days ending the 14th)
		row("a", "production", domain.DeploymentStatusSucceeded, day(3, 10), 40),
		row("a", "production", domain.DeploymentStatusFailed, day(5, 10), 20),
		// Current period (8–14 Nov)
		row("a", "production", domain.DeploymentStatusSucceeded, day(8, 10), 60),
		row("a", "production", domain.DeploymentStatusFailed, day(10, 9), 30),
		row("b", "staging", domain.DeploymentStatusSucceeded, day(10, 12), 50),
		row("a", "production", domain.DeploymentStatusSucceeded, day(10, 11), 40),
		row("b", "staging", domain.DeploymentStatusRunning, day(14, 14), 0),
		// Out of range
		row("a", "production", domain.DeploymentStatusSucceeded, time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC), 10),
	}

	s := get_dashboard.Summarize(get_dashboard.Filters{Days: 7, Until: until}, rows, 2)

	t.Run("should count the current and previous periods", func(t *testing.T) {
		assert.Equal(t, 5, s.Current.Deployments)
		assert.Equal(t, 3, s.Current.Succeeded)
		assert.Equal(t, 1, s.Current.Failed)
		assert.Equal(t, 1, s.Current.Other)
		assert.Equal(t, 0.75, s.Current.SuccessRate.MustGet())
		assert.Equal(t, 2, s.Previous.Deployments)
		assert.Equal(t, 0.5, s.Previous.SuccessRate.MustGet())
		assert.Equal(t, 2, s.Apps)
	})

	t.Run("should compute the median and slowest durations", func(t *testing.T) {
		// 60, 30, 50, 40 → median 45
		assert.Equal(t, 45.0, s.Current.MedianDuration.MustGet())
		assert.Equal(t, 60.0, s.Current.MaxDuration.MustGet())
	})

	t.Run("should bucket days and list errors", func(t *testing.T) {
		assert.HasLength(t, 7, s.Series)
		assert.Equal(t, "2026-11-08", s.Series[0].Date)
		assert.Equal(t, 1, s.Series[2].Failed)
		assert.Equal(t, 2, s.Series[2].Succeeded)
		assert.Equal(t, 1, s.Series[6].Other)
		assert.DeepEqual(t, []get_dashboard.ErrorStat{{Code: "compose_up_failed", Count: 1}}, s.Errors)
		assert.DeepEqual(t, []get_dashboard.EnvironmentStat{
			{Environment: "production", Deployments: 3, Succeeded: 2, Failed: 1},
			{Environment: "staging", Deployments: 2, Succeeded: 1},
		}, s.ByEnvironment)
	})

	t.Run("should measure the time to recover in production", func(t *testing.T) {
		// Failed 5 Nov 10:00:20, recovered 8 Nov 10:01:00 (previous failure counted when the success lands)
		// then failed 10 Nov 09:00:30, recovered 10 Nov 11:00:40.
		assert.Equal(t, "production", s.Recovery.Environment)
		assert.Equal(t, 2, s.Current.Recoveries)
		first := day(8, 10).Add(60 * time.Second).Sub(day(5, 10).Add(20 * time.Second)).Seconds()
		second := day(10, 11).Add(40 * time.Second).Sub(day(10, 9).Add(30 * time.Second)).Seconds()
		assert.Equal(t, (first+second)/2, s.Current.MedianRecovery.MustGet())
	})
}

func Test_SummarizeCancelled(t *testing.T) {
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	cancelled := row("a", "production", domain.DeploymentStatusFailed, at.Add(time.Hour), 300)
	cancelled.ErrCode.Set(domain.ErrDeploymentCancelled.Error())
	rows := []get_dashboard.Row{
		row("a", "production", domain.DeploymentStatusSucceeded, at, 40),
		cancelled,
		row("a", "production", domain.DeploymentStatusSucceeded, at.Add(2*time.Hour), 60),
	}

	s := get_dashboard.Summarize(get_dashboard.Filters{Days: 7, Until: until}, rows, 1)

	t.Run("should count a cancelled deployment neither as a success nor as a failure", func(t *testing.T) {
		assert.Equal(t, 3, s.Current.Deployments)
		assert.Equal(t, 2, s.Current.Succeeded)
		assert.Equal(t, 0, s.Current.Failed)
		assert.Equal(t, 1, s.Current.Other)
		assert.Equal(t, 1.0, s.Current.SuccessRate.MustGet())
		assert.HasLength(t, 0, s.Errors)
		assert.Equal(t, 0, s.Current.Recoveries)
	})

	t.Run("should leave its run time out of the durations", func(t *testing.T) {
		assert.Equal(t, 60.0, s.Current.MaxDuration.MustGet())
	})
}

func Test_RankApps(t *testing.T) {
	at := until.Add(-time.Hour)
	apps := []get_dashboard.AppRef{{ID: "a", Name: "api"}, {ID: "b", Name: "blog"}, {ID: "c", Name: "docs"}}
	rows := []get_dashboard.Row{
		row("a", "production", domain.DeploymentStatusSucceeded, at, 1),
		row("a", "production", domain.DeploymentStatusSucceeded, at.Add(-time.Hour), 1),
		row("b", "production", domain.DeploymentStatusFailed, at.Add(time.Minute), 1),
	}

	t.Run("should sort by deploys, then name, and page", func(t *testing.T) {
		page := get_dashboard.RankApps(get_dashboard.AppsQuery{Filters: get_dashboard.Filters{Until: until}, PerPage: 2}, apps, rows)

		assert.Equal(t, 3, page.Total)
		assert.HasLength(t, 2, page.Data)
		assert.Equal(t, "api", page.Data[0].Name)
		assert.Equal(t, "blog", page.Data[1].Name)
		assert.False(t, page.IsLastPage)

		next := get_dashboard.RankApps(get_dashboard.AppsQuery{Filters: get_dashboard.Filters{Until: until}, PerPage: 2, Page: 2}, apps, rows)
		assert.HasLength(t, 1, next.Data)
		assert.Equal(t, "docs", next.Data[0].Name)
		assert.False(t, next.Data[0].SuccessRate.HasValue())
		assert.True(t, next.IsLastPage)
	})

	t.Run("should sort by failures and by most recent", func(t *testing.T) {
		failures := get_dashboard.RankApps(get_dashboard.AppsQuery{Filters: get_dashboard.Filters{Until: until}, Sort: "failures"}, apps, rows)
		assert.Equal(t, "blog", failures.Data[0].Name)

		recent := get_dashboard.RankApps(get_dashboard.AppsQuery{Filters: get_dashboard.Filters{Until: until}, Sort: "recent"}, apps, rows)
		assert.Equal(t, "blog", recent.Data[0].Name)
		assert.Equal(t, "docs", recent.Data[2].Name) // never deployed: last
	})

	t.Run("should cap the page size", func(t *testing.T) {
		page := get_dashboard.RankApps(get_dashboard.AppsQuery{Filters: get_dashboard.Filters{Until: until}, PerPage: 1000}, apps, rows)
		assert.Equal(t, get_dashboard.MaxPerPage, page.PerPage)
	})
}

func Test_Cursor(t *testing.T) {
	t.Run("should round-trip", func(t *testing.T) {
		c := get_dashboard.Cursor{RequestedAt: until.Add(123 * time.Millisecond), AppID: "2fA9q", Number: 42}

		decoded, err := get_dashboard.DecodeCursor(c.Encode())

		assert.Nil(t, err)
		assert.Equal(t, c, decoded)
	})

	t.Run("should refuse anything malformed", func(t *testing.T) {
		for _, value := range []string{"", "%%%", "bm9wZQ", "eyJ0IjoieCIsImEiOiJhIiwibiI6MX0"} {
			_, err := get_dashboard.DecodeCursor(value)
			assert.ErrorIs(t, get_dashboard.ErrInvalidCursor, err)
		}
	})
}

func Test_Median(t *testing.T) {
	assert.Equal(t, 2.0, get_dashboard.Median([]float64{3, 1, 2}))
	assert.Equal(t, 2.5, get_dashboard.Median([]float64{4, 1, 3, 2}))
	assert.Equal(t, 0.0, get_dashboard.Median(nil))
}
