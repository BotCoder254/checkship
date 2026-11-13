//go:build !release

package fixture

import (
	"context"
	"os"
	"testing"

	"github.com/checkship-deploy/checkship/cmd/config"
	"github.com/checkship-deploy/checkship/internal/auth/domain"
	auth "github.com/checkship-deploy/checkship/internal/auth/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/bus/spy"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/ostools"
	"github.com/checkship-deploy/checkship/pkg/storage/sqlite"
)

type (
	seed struct {
		users []*domain.User
	}

	Context struct {
		Context    context.Context // If users has been seeded, will be authenticated as the first one
		Dispatcher spy.Dispatcher
		UsersStore auth.UsersStore
	}

	SeedBuilder func(*seed)
)

func PrepareDatabase(t testing.TB, options ...SeedBuilder) *Context {
	cfg := config.Default(config.WithTestDefaults())

	if err := ostools.MkdirAll(cfg.DataDir()); err != nil {
		t.Fatal(err)
	}

	result := Context{
		Context:    context.Background(),
		Dispatcher: spy.NewDispatcher(),
	}

	db, err := sqlite.Open(cfg.ConnectionString(), must.Panic(log.NewLogger()), result.Dispatcher)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(cfg.DataDir())
	})

	if err = db.Migrate(auth.Migrations); err != nil {
		t.Fatal(err)
	}

	result.UsersStore = auth.NewUsersStore(db)

	// Seed the database
	var s seed

	for _, o := range options {
		o(&s)
	}

	if len(s.users) > 0 {
		if err := result.UsersStore.Write(result.Context, s.users...); err != nil {
			t.Fatal(err)
		}

		result.Context = domain.WithUserID(result.Context, s.users[0].ID()) // The first created user will be used as the authenticated one
	}

	// Reset the dispatcher after seeding
	result.Dispatcher.Reset()

	return &result
}

func WithUsers(users ...*domain.User) SeedBuilder {
	return func(s *seed) {
		s.users = users
	}
}
