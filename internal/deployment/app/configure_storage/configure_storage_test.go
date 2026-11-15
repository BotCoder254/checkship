package configure_storage_test

import (
	"context"
	"errors"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/configure_storage"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	deploymentsqlite "github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
)

type options struct{}

func (options) BuiltinStorageImage() string { return "versity/versitygw:v1.11.0" }

type fakeProvider struct {
	domain.Provider
	err       error
	endpoints []domain.StorageEndpoint
}

func (p *fakeProvider) SetupStorage(context.Context, domain.Target, domain.BuiltinStorageSpec) ([]domain.StorageEndpoint, error) {
	return p.endpoints, p.err
}

// Answers only on the given endpoint.
type fakeChecker struct{ answers string }

func (c fakeChecker) Check(_ context.Context, access domain.BackupStorageAccess) error {
	if access.Endpoint == c.answers {
		return nil
	}
	return errors.New("connection refused")
}

type noMove struct{}

func (noMove) Move(context.Context, domain.BackupStorageAccess, domain.BackupStorageAccess) (int, error) {
	return 0, nil
}

func Test_ConfigureStorage(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	user := authfixture.User()
	target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))

	run := func(t *testing.T, provider *fakeProvider, checker fakeChecker) domain.BackupStorage {
		ctx := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&target))
		store := deploymentsqlite.NewBackupStorageStore(ctx.DB)
		settings := must.Panic(store.Get(ctx.Context))
		settings.UseBuiltin(target.ID(), "key", "secret")
		assert.Nil(t, store.Write(ctx.Context, &settings))
		settings = must.Panic(store.Get(ctx.Context))

		handler := configure_storage.Handler(store, store, ctx.TargetsStore, provider, checker, noMove{}, options{}, logger)
		_, err := handler(ctx.Context, configure_storage.Command{Version: settings.Version()})
		assert.Nil(t, err)

		return must.Panic(store.Get(ctx.Context))
	}

	t.Run("should keep the first endpoint that answers", func(t *testing.T) {
		got := run(t, &fakeProvider{endpoints: []domain.StorageEndpoint{
			{Address: "checkship-storage:7070"},
			{Address: "127.0.0.1:7070"},
		}}, fakeChecker{answers: "127.0.0.1:7070"})

		assert.Equal(t, domain.BackupStorageReady, got.Status())
		assert.Equal(t, "127.0.0.1:7070", got.Access().Endpoint)
	})

	t.Run("should record a failed setup as failed", func(t *testing.T) {
		got := run(t, &fakeProvider{err: errors.New("network not found")}, fakeChecker{})

		assert.Equal(t, domain.BackupStorageFailed, got.Status())
		assert.Equal(t, "", got.Access().Endpoint)
		assert.True(t, got.ErrorCode().HasValue())
	})
}
