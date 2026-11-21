package domain

import (
	"context"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	shared "github.com/checkship-deploy/checkship/pkg/domain"
	"github.com/checkship-deploy/checkship/pkg/event"
	"github.com/checkship-deploy/checkship/pkg/id"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

var (
	ErrAppNameAlreadyTaken         = apperr.New("app_name_already_taken")
	ErrVersionControlNotConfigured = apperr.New("version_control_not_configured")
	ErrAppCleanupNeeded            = apperr.New("app_cleanup_needed")
	ErrAppCleanupRequested         = apperr.New("app_cleanup_requested")
	ErrAppTargetChanged            = apperr.New("app_target_changed")
)

type (
	AppID           string
	HasAppsOnTarget bool

	App struct {
		event.Emitter

		id               AppID
		name             AppName
		versionControl   monad.Maybe[VersionControl]
		production       EnvironmentConfig
		staging          EnvironmentConfig
		cleanupRequested monad.Maybe[shared.Action[domain.UserID]]
		created          shared.Action[domain.UserID]
	}

	AppsReader interface {
		// Check if the naming is available (not use by another application with the same name on the same targets).
		CheckAppNamingAvailability(
			ctx context.Context,
			name AppName,
			production EnvironmentConfig,
			staging EnvironmentConfig,
		) (EnvironmentConfigRequirement, EnvironmentConfigRequirement, error)
		// Same as CheckAppNamingAvailability but used when updating the environment configuration with optional targets.
		CheckAppNamingAvailabilityByID(
			ctx context.Context,
			id AppID,
			production monad.Maybe[EnvironmentConfig],
			staging monad.Maybe[EnvironmentConfig],
		) (EnvironmentConfigRequirement, EnvironmentConfigRequirement, error)
		// Check if a specific target is used by an application.
		HasAppsOnTarget(context.Context, TargetID) (HasAppsOnTarget, error)
		// Which of the given custom domains are already used by another application than
		// the given one, with that application's name.
		GetDomainsOwners(ctx context.Context, except AppID, hostnames []Hostname) (map[Hostname]AppName, error)
		// Custom domains of every application environment running on the given target.
		GetDomainsOnTarget(ctx context.Context, target TargetID) ([]TargetDomain, error)
		GetByID(context.Context, AppID) (App, error)
	}

	AppsWriter interface {
		Write(context.Context, ...*App) error
	}

	AppCreated struct {
		bus.Notification

		ID         AppID
		Name       AppName
		Production EnvironmentConfig
		Staging    EnvironmentConfig
		Created    shared.Action[domain.UserID]
	}

	AppEnvChanged struct {
		bus.Notification

		ID          AppID
		Environment Environment
		Config      EnvironmentConfig
		OldConfig   EnvironmentConfig          // Old configuration, used to ease the cleanup handling
		MoveVolumes bool                       // When the target changed: copy the volumes to the new one
		ChangedBy   monad.Maybe[domain.UserID] // Who changed it, when known
	}

	// How an environment change is carried out.
	EnvironmentChange struct {
		MoveVolumes bool          // Copy the volumes to the new target, if the target changes
		By          domain.UserID // Who asks for the change
	}

	AppVersionControlConfigured struct {
		bus.Notification

		ID     AppID
		Config VersionControl
	}

	AppVersionControlRemoved struct {
		bus.Notification

		ID AppID
	}

	AppCleanupRequested struct {
		bus.Notification

		ID               AppID
		ProductionConfig EnvironmentConfig
		StagingConfig    EnvironmentConfig
		Requested        shared.Action[domain.UserID]
		BackupVolumes    bool // Back up each environment's volumes before they are removed
	}

	AppDeleted struct {
		bus.Notification

		ID AppID
	}
)

func (AppCreated) Name_() string    { return "deployment.event.app_created" }
func (AppEnvChanged) Name_() string { return "deployment.event.app_env_changed" }
func (AppVersionControlConfigured) Name_() string {
	return "deployment.event.app_version_control_configured"
}
func (AppVersionControlRemoved) Name_() string { return "deployment.event.app_version_control_removed" }
func (AppCleanupRequested) Name_() string      { return "deployment.event.app_cleanup_requested" }
func (AppDeleted) Name_() string               { return "deployment.event.app_deleted" }

func (e AppEnvChanged) TargetHasChanged() bool { return e.Config.target != e.OldConfig.target }

// Instantiates a new App.
func NewApp(
	name AppName,
	productionRequirement EnvironmentConfigRequirement,
	stagingRequirement EnvironmentConfigRequirement,
	createdBy domain.UserID,
) (app App, err error) {
	production, err := productionRequirement.Met()

	if err != nil {
		return app, err
	}

	staging, err := stagingRequirement.Met()

	if err != nil {
		return app, err
	}

	app.apply(AppCreated{
		ID:         id.New[AppID](),
		Name:       name,
		Production: production,
		Staging:    staging,
		Created:    shared.NewAction(createdBy),
	})

	return app, nil
}

// Recreates an app from the persistent storage.
func AppFrom(scanner storage.Scanner) (a App, err error) {
	var (
		url                monad.Maybe[Url]
		token              monad.Maybe[string]
		createdAt          time.Time
		createdBy          domain.UserID
		cleanupRequestedAt monad.Maybe[time.Time]
		cleanupRequestedBy monad.Maybe[string]
	)

	err = scanner.Scan(
		&a.id,
		&a.name,
		&url,
		&token,
		&a.production.target,
		&a.production.version,
		&a.production.vars,
		&a.production.domains,
		&a.staging.target,
		&a.staging.version,
		&a.staging.vars,
		&a.staging.domains,
		&cleanupRequestedAt,
		&cleanupRequestedBy,
		&createdAt,
		&createdBy,
	)

	a.created = shared.ActionFrom(createdBy, createdAt)

	if requestedAt, isSet := cleanupRequestedAt.TryGet(); isSet {
		a.cleanupRequested.Set(
			shared.ActionFrom(domain.UserID(cleanupRequestedBy.MustGet()), requestedAt),
		)
	}

	// vcs url has been set, reconstitute the vcs config
	if u, isSet := url.TryGet(); isSet {
		vcs := NewVersionControl(u)

		if tok, isSet := token.TryGet(); isSet {
			vcs.Authenticated(tok)
		}

		a.versionControl.Set(vcs)
	}

	return a, err
}

// Sets an app version control configuration.
func (a *App) UseVersionControl(config VersionControl) error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	if existing, isSet := a.versionControl.TryGet(); isSet && config == existing {
		return nil
	}

	a.apply(AppVersionControlConfigured{
		ID:     a.id,
		Config: config,
	})

	return nil
}

// Removes the version control configuration from the app.
func (a *App) RemoveVersionControl() error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	if !a.versionControl.HasValue() {
		return nil
	}

	a.apply(AppVersionControlRemoved{
		ID: a.id,
	})

	return nil
}

// Updates the production configuration for this application.
func (a *App) HasProductionConfig(configRequirement EnvironmentConfigRequirement) error {
	return a.tryUpdateEnvironmentConfig(Production, a.production, configRequirement, monad.None[EnvironmentChange]())
}

// Updates the staging configuration for this application.
func (a *App) HasStagingConfig(configRequirement EnvironmentConfigRequirement) error {
	return a.tryUpdateEnvironmentConfig(Staging, a.staging, configRequirement, monad.None[EnvironmentChange]())
}

// Same as HasProductionConfig, saying who changes it and whether volume data
// follows a change of target.
func (a *App) ChangeProductionConfig(configRequirement EnvironmentConfigRequirement, change EnvironmentChange) error {
	return a.tryUpdateEnvironmentConfig(Production, a.production, configRequirement, monad.Value(change))
}

// Same as HasStagingConfig, saying who changes it and whether volume data
// follows a change of target.
func (a *App) ChangeStagingConfig(configRequirement EnvironmentConfigRequirement, change EnvironmentChange) error {
	return a.tryUpdateEnvironmentConfig(Staging, a.staging, configRequirement, monad.Value(change))
}

// Request cleaning for this application. This marks the application for deletion.
func (a *App) RequestCleanup(requestedBy domain.UserID) {
	a.RequestCleanupWithBackup(requestedBy, false)
}

// Same as RequestCleanup, backing up the volumes before they are removed when asked.
func (a *App) RequestCleanupWithBackup(requestedBy domain.UserID, backupVolumes bool) {
	if a.cleanupRequested.HasValue() {
		return
	}

	a.apply(AppCleanupRequested{
		ID:               a.id,
		ProductionConfig: a.production,
		StagingConfig:    a.staging,
		Requested:        shared.NewAction(requestedBy),
		BackupVolumes:    backupVolumes,
	})
}

// Delete the application.
func (a *App) Delete(cleanedUp bool) error {
	if !a.cleanupRequested.HasValue() || !cleanedUp {
		return ErrAppCleanupNeeded
	}

	a.apply(AppDeleted{
		ID: a.id,
	})

	return nil
}

func (a *App) ID() AppID                                   { return a.id }
func (a *App) VersionControl() monad.Maybe[VersionControl] { return a.versionControl }

// Configuration of the given environment.
// Subdomain of the environment's main service on its target: the application name, plus
// "-<environment>" outside production (see ConfigSnapshot.subDomain).
func (a *App) DefaultSubdomain(env Environment) string {
	if env.IsProduction() {
		return string(a.name)
	}

	return string(a.name) + "-" + string(env)
}

func (a *App) Config(env Environment) EnvironmentConfig {
	if env.IsProduction() {
		return a.production
	}

	return a.staging
}

func (a *App) tryUpdateEnvironmentConfig(
	env Environment,
	existingConfig EnvironmentConfig,
	updatedConfigRequirement EnvironmentConfigRequirement,
	change monad.Maybe[EnvironmentChange],
) error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	updatedConfig, err := updatedConfigRequirement.Met()

	if err != nil {
		return err
	}

	// Same configuration, returns
	if updatedConfig.Equals(existingConfig) {
		return nil
	}

	updatedConfig.consolidate(existingConfig)

	evt := AppEnvChanged{
		ID:          a.id,
		Environment: env,
		Config:      updatedConfig,
		OldConfig:   existingConfig,
	}

	if c, isSet := change.TryGet(); isSet {
		evt.MoveVolumes = c.MoveVolumes
		evt.ChangedBy.Set(c.By)
	}

	a.apply(evt)

	return nil
}

func (a *App) apply(e event.Event) {
	switch evt := e.(type) {
	case AppCreated:
		a.id = evt.ID
		a.name = evt.Name
		a.production = evt.Production
		a.staging = evt.Staging
		a.created = evt.Created
	case AppEnvChanged:
		switch evt.Environment {
		case Production:
			a.production = evt.Config
		case Staging:
			a.staging = evt.Config
		}
	case AppVersionControlConfigured:
		a.versionControl.Set(evt.Config)
	case AppVersionControlRemoved:
		a.versionControl.Unset()
	case AppCleanupRequested:
		a.cleanupRequested.Set(evt.Requested)
	}

	event.Store(a, e)
}

func (a *App) Name() AppName { return a.name }

// Compose project name of an environment. Volumes created by a restore are named after it.
func (a *App) ProjectName(env Environment) (string, error) {
	snapshot, err := a.configSnapshotFor(env)

	if err != nil {
		return "", err
	}

	return snapshot.ProjectName(), nil
}

// Who created the application. Automatic backups with no other author are recorded under this user.
func (a *App) CreatedBy() domain.UserID { return a.created.By() }
