package sqlite

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_deployments"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_dashboard"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

// Hard cap on the rows read for one summary (two periods of at most 90 days). Far above what a
// self-hosted instance deploys, it only bounds the work if something goes wrong.
const maxDashboardRows = 100_000

func (s *gateway) GetDashboardSummary(ctx context.Context, q get_dashboard.SummaryQuery) (get_dashboard.Summary, error) {
	f := q.Filters.Normalized()
	previousFrom, _, until := f.Bounds()

	rows, err := s.dashboardRows(ctx, f, q.App, q.Target, previousFrom, until)
	if err != nil {
		return get_dashboard.Summary{}, err
	}

	apps, err := s.dashboardApps(ctx)
	if err != nil {
		return get_dashboard.Summary{}, err
	}

	return get_dashboard.Summarize(f, rows, len(apps)), nil
}

func (s *gateway) GetDashboardApps(ctx context.Context, q get_dashboard.AppsQuery) (storage.Paginated[get_dashboard.AppStat], error) {
	q.Filters = q.Filters.Normalized()
	_, from, until := q.Filters.Bounds()

	rows, err := s.dashboardRows(ctx, q.Filters, monad.None[string](), monad.None[string](), from, until)
	if err != nil {
		return storage.Paginated[get_dashboard.AppStat]{}, err
	}

	apps, err := s.dashboardApps(ctx)
	if err != nil {
		return storage.Paginated[get_dashboard.AppStat]{}, err
	}

	return get_dashboard.RankApps(q, apps, rows), nil
}

// Deployments across every application, newest first, paginated with a keyset cursor so rows
// queued while someone reads the feed never shift the next page.
func (s *gateway) GetDashboardActivity(ctx context.Context, q get_dashboard.ActivityQuery) (get_dashboard.Activity, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = get_dashboard.DefaultActivity
	}
	limit = min(limit, get_dashboard.MaxActivity)

	data, err := builder.
		Query[get_app_deployments.Deployment](`
			SELECT
				deployments.app_id
				,deployments.deployment_number
				,deployments.config_environment
				,deployments.config_target
				,targets.name
				,targets.url
				,targets.state_status
				,deployments.source_discriminator
				,deployments.source
				,deployments.state_status
				,deployments.state_errcode
				,deployments.state_started_at
				,deployments.state_finished_at
				,deployments.requested_at
				,users.id
				,users.email
				,'' -- only to use the same mapper as the latest deployments
			FROM deployments
			INNER JOIN users ON users.id = deployments.requested_by
			LEFT JOIN targets ON targets.id = deployments.config_target
			WHERE TRUE`).
		S(builder.MaybeValue(q.Environment, "AND deployments.config_environment = ?")).
		S(builder.MaybeValue(q.Status, "AND deployments.state_status = ?")).
		S(builder.MaybeValue(q.AppID, "AND deployments.app_id = ?")).
		S(builder.MaybeValue(q.Since, "AND deployments.requested_at >= ?")).
		S(builder.Maybe(q.After, func(c get_dashboard.Cursor) (string, []any) {
			return `AND (deployments.requested_at, deployments.app_id, deployments.deployment_number) < (?, ?, ?)`,
				[]any{c.RequestedAt, c.AppID, c.Number}
		})).
		F(`ORDER BY deployments.requested_at DESC, deployments.app_id DESC, deployments.deployment_number DESC
			LIMIT ?`, limit+1).
		All(s.db, ctx, deploymentMapper(nil))

	if err != nil {
		return get_dashboard.Activity{}, err
	}

	result := get_dashboard.Activity{Data: data}

	if len(data) > limit {
		result.Data = data[:limit]
		last := result.Data[limit-1]
		result.NextCursor.Set(get_dashboard.Cursor{
			RequestedAt: last.RequestedAt,
			AppID:       last.AppID,
			Number:      last.DeploymentNumber,
		}.Encode())
	}

	if result.Data == nil {
		result.Data = []get_app_deployments.Deployment{}
	}

	return result, nil
}

func (s *gateway) dashboardRows(ctx context.Context, f get_dashboard.Filters, app, target monad.Maybe[string], from, until any) ([]get_dashboard.Row, error) {
	return builder.
		Query[get_dashboard.Row](`
			SELECT
				deployments.app_id
				,deployments.config_environment
				,deployments.state_status
				,deployments.state_errcode
				,deployments.state_started_at
				,deployments.state_finished_at
				,deployments.requested_at
			FROM deployments
			WHERE deployments.requested_at >= ? AND deployments.requested_at <= ?`, from, until).
		S(builder.MaybeValue(f.Environment, "AND deployments.config_environment = ?")).
		S(builder.MaybeValue(app, "AND deployments.app_id = ?")).
		S(builder.MaybeValue(target, "AND deployments.config_target = ?")).
		F("ORDER BY deployments.requested_at LIMIT ?", maxDashboardRows).
		All(s.db, ctx, func(scanner storage.Scanner) (r get_dashboard.Row, err error) {
			err = scanner.Scan(
				&r.AppID,
				&r.Environment,
				&r.Status,
				&r.ErrCode,
				&r.StartedAt,
				&r.FinishedAt,
				&r.RequestedAt,
			)
			return r, err
		})
}

// Applications not being removed.
func (s *gateway) dashboardApps(ctx context.Context) ([]get_dashboard.AppRef, error) {
	return builder.
		Query[get_dashboard.AppRef]("SELECT id, name FROM apps WHERE cleanup_requested_at IS NULL ORDER BY name").
		All(s.db, ctx, func(scanner storage.Scanner) (a get_dashboard.AppRef, err error) {
			err = scanner.Scan(&a.ID, &a.Name)
			return a, err
		})
}
