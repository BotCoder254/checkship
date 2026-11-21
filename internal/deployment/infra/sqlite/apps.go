package sqlite

import (
	"context"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite/builder"
)

type (
	AppsStore interface {
		domain.AppsReader
		domain.AppsWriter
	}

	appsStore struct {
		db *sqlite.Database
	}
)

func NewAppsStore(db *sqlite.Database) AppsStore {
	return &appsStore{db}
}

func (s *appsStore) CheckAppNamingAvailability(
	ctx context.Context,
	name domain.AppName,
	production domain.EnvironmentConfig,
	staging domain.EnvironmentConfig,
) (domain.EnvironmentConfigRequirement, domain.EnvironmentConfigRequirement, error) {
	r, err := builder.
		Query[appNamingResult](`
		SELECT
			NOT EXISTS(SELECT 1 FROM apps WHERE name = ? AND production_target = ?) AS production_available
			,EXISTS(SELECT 1 FROM targets WHERE id = ? AND cleanup_requested_at IS NULL) AS production_target_exists
			,NOT EXISTS(SELECT 1 FROM apps WHERE name = ? AND staging_target = ?) AS staging_available
			,EXISTS(SELECT 1 FROM targets WHERE id = ? AND cleanup_requested_at IS NULL) AS staging_target_exists
		`, name, production.Target(), production.Target(), name, staging.Target(), staging.Target()).
		One(s.db, ctx, appNameUniquenessResultMapper)

	if err != nil {
		return domain.EnvironmentConfigRequirement{}, domain.EnvironmentConfigRequirement{}, err
	}

	return domain.NewEnvironmentConfigRequirement(production, r.productionTargetFound, r.productionAvailable),
		domain.NewEnvironmentConfigRequirement(staging, r.stagingTargetFound, r.stagingAvailable),
		nil
}

func (s *appsStore) CheckAppNamingAvailabilityByID(
	ctx context.Context,
	id domain.AppID,
	production monad.Maybe[domain.EnvironmentConfig],
	staging monad.Maybe[domain.EnvironmentConfig],
) (
	productionRequirement domain.EnvironmentConfigRequirement,
	stagingRequirement domain.EnvironmentConfigRequirement,
	err error,
) {
	productionValue, hasProductionTarget := production.TryGet()
	stagingValue, hasStagingTarget := staging.TryGet()

	if !hasProductionTarget && !hasStagingTarget {
		return productionRequirement, stagingRequirement, err
	}

	var (
		sql  strings.Builder
		args = make([]any, 0, 5)
	)

	// This one is a bit tricky because the request depends on how many target we should check.
	sql.WriteString("SELECT ")

	if hasProductionTarget {
		sql.WriteString(`
		NOT EXISTS(SELECT 1 FROM apps WHERE apps.id != src.id AND apps.name = src.name AND production_target = ?) AS production_available
		,EXISTS(SELECT 1 FROM targets WHERE id = ? AND cleanup_requested_at IS NULL) AS production_target_exists`)
		args = append(args, productionValue.Target(), productionValue.Target())
	} else {
		sql.WriteString("0 AS production_available, 0 AS production_target_exists")
	}

	if hasStagingTarget {
		sql.WriteString(`
		,NOT EXISTS(SELECT 1 FROM apps WHERE apps.id != src.id AND apps.name = src.name AND staging_target = ?) AS staging_available
		,EXISTS(SELECT 1 FROM targets WHERE id = ? AND cleanup_requested_at IS NULL) AS staging_target_exists`)
		args = append(args, stagingValue.Target(), stagingValue.Target())
	} else {
		sql.WriteString(", 0 AS staging_available, 0 AS staging_target_exists")
	}

	sql.WriteString(" FROM apps src WHERE src.id = ?")
	args = append(args, id)

	r, err := builder.
		Query[appNamingResult](sql.String(), args...).
		One(s.db, ctx, appNameUniquenessResultMapper)

	if err != nil {
		return productionRequirement, stagingRequirement, err
	}

	if hasProductionTarget {
		productionRequirement = domain.NewEnvironmentConfigRequirement(productionValue, r.productionTargetFound, r.productionAvailable)
	}

	if hasStagingTarget {
		stagingRequirement = domain.NewEnvironmentConfigRequirement(stagingValue, r.stagingTargetFound, r.stagingAvailable)
	}

	return productionRequirement, stagingRequirement, err
}

func (s *appsStore) HasAppsOnTarget(ctx context.Context, target domain.TargetID) (domain.HasAppsOnTarget, error) {
	r, err := builder.
		Query[bool](`
		SELECT EXISTS(SELECT 1 FROM apps WHERE production_target = ? OR staging_target = ?)`,
		target, target).
		Extract(s.db, ctx)

	return domain.HasAppsOnTarget(r), err
}

func (s *appsStore) GetByID(ctx context.Context, id domain.AppID) (domain.App, error) {
	return builder.
		Query[domain.App](`
		SELECT
			id
			,name
			,version_control_url
			,version_control_token
			,production_target
			,production_version
			,production_vars
			,production_domains
			,staging_target
			,staging_version
			,staging_vars
			,staging_domains
			,cleanup_requested_at
			,cleanup_requested_by
			,created_at
			,created_by
		FROM apps
		WHERE id = ?`, id).
		One(s.db, ctx, domain.AppFrom)
}

func (s *appsStore) Write(c context.Context, apps ...*domain.App) error {
	return sqlite.WriteAndDispatch(s.db, c, apps, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.AppCreated:
			if err := builder.
				Insert("apps", builder.Values{
					"id":                 evt.ID,
					"name":               evt.Name,
					"production_target":  evt.Production.Target(),
					"production_version": evt.Production.Version(),
					"production_vars":    evt.Production.Vars(),
					"production_domains": evt.Production.Domains(),
					"staging_target":     evt.Staging.Target(),
					"staging_version":    evt.Staging.Version(),
					"staging_vars":       evt.Staging.Vars(),
					"staging_domains":    evt.Staging.Domains(),
					"created_at":         evt.Created.At(),
					"created_by":         evt.Created.By(),
				}).
				Exec(s.db, ctx); err != nil {
				return err
			}

			if err := s.syncDomains(ctx, evt.ID, domain.Production, evt.Production); err != nil {
				return err
			}

			return s.syncDomains(ctx, evt.ID, domain.Staging, evt.Staging)
		case domain.AppEnvChanged:
			// This is safe to interpolate the column name here since events are raised by our
			// own code.
			if err := builder.
				Update("apps", builder.Values{
					string(evt.Environment) + "_target":  evt.Config.Target(),
					string(evt.Environment) + "_version": evt.Config.Version(),
					string(evt.Environment) + "_vars":    evt.Config.Vars(),
					string(evt.Environment) + "_domains": evt.Config.Domains(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx); err != nil {
				return err
			}

			return s.syncDomains(ctx, evt.ID, evt.Environment, evt.Config)
		case domain.AppVersionControlConfigured:
			return builder.
				Update("apps", builder.Values{
					"version_control_url":   evt.Config.Url(),
					"version_control_token": evt.Config.Token(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppVersionControlRemoved:
			return builder.
				Update("apps", builder.Values{
					"version_control_url":   nil,
					"version_control_token": nil,
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppCleanupRequested:
			return builder.
				Update("apps", builder.Values{
					"cleanup_requested_at": evt.Requested.At(),
					"cleanup_requested_by": evt.Requested.By(),
				}).
				F("WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		case domain.AppDeleted:
			return builder.
				Command("DELETE FROM apps WHERE id = ?", evt.ID).
				Exec(s.db, ctx)
		default:
			return nil
		}
	})
}

// Mirrors an environment's custom domains in app_domains, whose primary key keeps every
// domain on a single application environment.
func (s *appsStore) syncDomains(ctx context.Context, id domain.AppID, env domain.Environment, config domain.EnvironmentConfig) error {
	if err := builder.
		Command("DELETE FROM app_domains WHERE app_id = ? AND environment = ?", id, env).
		Exec(s.db, ctx); err != nil {
		return err
	}

	domains, hasDomains := config.Domains().TryGet()

	if !hasDomains {
		return nil
	}

	for service, hostnames := range domains {
		for _, hostname := range hostnames {
			if err := builder.
				Insert("app_domains", builder.Values{
					"hostname":    hostname,
					"app_id":      id,
					"environment": env,
					"service":     service,
				}).
				Exec(s.db, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// Returns, for each given hostname already used by another application, the name of that
// application. The application being edited (if any) is left out: its own domains can move
// between its environments and services.
func (s *appsStore) GetDomainsOwners(ctx context.Context, except domain.AppID, hostnames []domain.Hostname) (map[domain.Hostname]domain.AppName, error) {
	owners := make(map[domain.Hostname]domain.AppName)

	if len(hostnames) == 0 {
		return owners, nil
	}

	rows, err := builder.
		Query[domainOwner](`
			SELECT app_domains.hostname, apps.name
			FROM app_domains
			INNER JOIN apps ON apps.id = app_domains.app_id
			WHERE app_domains.app_id != ?`, except).
		S(builder.Array("AND app_domains.hostname IN", hostnames)).
		All(s.db, ctx, domainOwnerMapper)

	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		owners[row.hostname] = row.app
	}

	return owners, nil
}

func (s *appsStore) GetDomainsOnTarget(ctx context.Context, target domain.TargetID) ([]domain.TargetDomain, error) {
	return builder.
		Query[domain.TargetDomain](`
			SELECT app_domains.hostname, apps.id, apps.name, app_domains.environment, app_domains.service
			FROM app_domains
			INNER JOIN apps ON apps.id = app_domains.app_id
			WHERE apps.cleanup_requested_at IS NULL
				AND ((app_domains.environment = 'production' AND apps.production_target = ?)
					OR (app_domains.environment = 'staging' AND apps.staging_target = ?))
			ORDER BY app_domains.hostname`, target, target).
		All(s.db, ctx, func(sc storage.Scanner) (d domain.TargetDomain, err error) {
			err = sc.Scan(&d.Hostname, &d.App, &d.AppName, &d.Environment, &d.Service)
			return d, err
		})
}

type domainOwner struct {
	hostname domain.Hostname
	app      domain.AppName
}

func domainOwnerMapper(s storage.Scanner) (o domainOwner, err error) {
	err = s.Scan(&o.hostname, &o.app)
	return o, err
}

type appNamingResult struct {
	productionAvailable   bool
	productionTargetFound bool
	stagingAvailable      bool
	stagingTargetFound    bool
}

func appNameUniquenessResultMapper(s storage.Scanner) (r appNamingResult, err error) {
	err = s.Scan(
		&r.productionAvailable,
		&r.productionTargetFound,
		&r.stagingAvailable,
		&r.stagingTargetFound,
	)

	return r, err
}
