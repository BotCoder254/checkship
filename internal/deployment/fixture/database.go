//go:build !release

package fixture

import (
	"context"
	"os"
	"testing"

	"github.com/checkship-deploy/checkship/cmd/config"
	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	authsqlite "github.com/checkship-deploy/checkship/internal/auth/infra/sqlite"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	deployment "github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/bus/spy"
	scheduler "github.com/checkship-deploy/checkship/pkg/bus/sqlite"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/ostools"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
)

type (
	seed struct {
		users       []*auth.User
		targets     []*domain.Target
		apps        []*domain.App
		deployments []*domain.Deployment
		registries  []*domain.Registry
		backups     []*domain.Backup
	}

	Context struct {
		Config           config.Configuration
		Context          context.Context // If users has been seeded, will be authenticated as the first one
		Dispatcher       spy.Dispatcher
		TargetsStore     deployment.TargetsStore
		AppsStore        deployment.AppsStore
		DeploymentsStore deployment.DeploymentsStore
		RegistriesStore  deployment.RegistriesStore
		BackupsStore     deployment.BackupsStore
		RestoresStore    deployment.RestoresStore
		DB               *sqlite.Database // For tests that read through the query gateway
	}

	SeedBuilder func(*seed)
)

func PrepareDatabase(t testing.TB, options ...SeedBuilder) *Context {
	result := Context{
		Config:     config.Default(config.WithTestDefaults()),
		Context:    context.Background(),
		Dispatcher: spy.NewDispatcher(),
	}

	if err := ostools.MkdirAll(result.Config.DataDir()); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.Open(result.Config.ConnectionString(), must.Panic(log.NewLogger()), result.Dispatcher)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(result.Config.DataDir())
	})

	// FIXME: scheduler migrations are needed because some migrations may queue a job by inserting inside
	// the scheduled_jobs table. That's a mistake from my side and I should fix it later.
	if err = db.Migrate(scheduler.Migrations, authsqlite.Migrations, deployment.Migrations); err != nil {
		t.Fatal(err)
	}

	result.DB = db
	result.AppsStore = deployment.NewAppsStore(db)
	result.TargetsStore = deployment.NewTargetsStore(db)
	result.DeploymentsStore = deployment.NewDeploymentsStore(db)
	result.RegistriesStore = deployment.NewRegistriesStore(db)
	result.BackupsStore = deployment.NewBackupsStore(db)
	result.RestoresStore = deployment.NewRestoresStore(db)

	// Seed the database
	var s seed

	for _, o := range options {
		o(&s)
	}

	if len(s.users) > 0 {
		if err := authsqlite.NewUsersStore(db).Write(result.Context, s.users...); err != nil {
			t.Fatal(err)
		}
		result.Context = auth.WithUserID(result.Context, s.users[0].ID()) // The first created user will be used as the authenticated one
	}

	if err := result.RegistriesStore.Write(result.Context, s.registries...); err != nil {
		t.Fatal(err)
	}

	if err := result.TargetsStore.Write(result.Context, s.targets...); err != nil {
		t.Fatal(err)
	}

	if err := result.AppsStore.Write(result.Context, s.apps...); err != nil {
		t.Fatal(err)
	}

	if err := result.DeploymentsStore.Write(result.Context, s.deployments...); err != nil {
		t.Fatal(err)
	}

	if err := result.BackupsStore.Write(result.Context, s.backups...); err != nil {
		t.Fatal(err)
	}

	// Reset the dispatcher after seeding
	result.Dispatcher.Reset()

	return &result
}

func WithUsers(users ...*auth.User) SeedBuilder {
	return func(s *seed) {
		s.users = users
	}
}

func WithTargets(targets ...*domain.Target) SeedBuilder {
	return func(s *seed) {
		s.targets = targets
	}
}

func WithApps(apps ...*domain.App) SeedBuilder {
	return func(s *seed) {
		s.apps = apps
	}
}

func WithDeployments(deployments ...*domain.Deployment) SeedBuilder {
	return func(s *seed) {
		s.deployments = deployments
	}
}

func WithRegistries(registries ...*domain.Registry) SeedBuilder {
	return func(s *seed) {
		s.registries = registries
	}
}

func WithBackups(backups ...*domain.Backup) SeedBuilder {
	return func(s *seed) {
		s.backups = backups
	}
}
