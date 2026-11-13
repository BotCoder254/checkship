package delete_registry_test

import (
	"context"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/delete_registry"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/spy"
)

func Test_DeleteRegistry(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, delete_registry.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return delete_registry.Handler(context.RegistriesStore, context.RegistriesStore), context.Dispatcher
	}

	t.Run("should require an existing registry", func(t *testing.T) {
		handler, _ := arrange(t)

		_, err := handler(context.Background(), delete_registry.Command{
			ID: "non-existing-id",
		})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should delete the registry", func(t *testing.T) {
		user := authfixture.User()
		registry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		handler, dispatcher := arrange(t, fixture.WithUsers(&user), fixture.WithRegistries(&registry))

		_, err := handler(context.Background(), delete_registry.Command{
			ID: string(registry.ID()),
		})

		assert.Nil(t, err)
		assert.HasLength(t, 1, dispatcher.Signals())

		deleted := assert.Is[domain.RegistryDeleted](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.RegistryDeleted{
			ID: registry.ID(),
		}, deleted)
	})
}
