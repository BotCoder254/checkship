package fixture_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/auth/infra/crypto"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_User(t *testing.T) {
	t.Run("should be able to create a random user", func(t *testing.T) {
		user := fixture.User()

		assert.NotZero(t, user.ID())
	})

	t.Run("should be able to create a user with a given email", func(t *testing.T) {
		user := fixture.User(fixture.WithEmail("an@email.com"))

		registered := assert.EventIs[domain.UserRegistered](t, &user, 0)
		assert.Equal(t, "an@email.com", registered.Email)
	})

	t.Run("should be able to create a user with a given password hash", func(t *testing.T) {
		user := fixture.User(fixture.WithPasswordHash("somePassword"))

		registered := assert.EventIs[domain.UserRegistered](t, &user, 0)
		assert.Equal(t, "somePassword", registered.Password)
	})

	t.Run("should be able to create a user with a given password", func(t *testing.T) {
		hasher := crypto.NewBCryptHasher()
		user := fixture.User(fixture.WithPassword("somePassword", hasher))

		registered := assert.EventIs[domain.UserRegistered](t, &user, 0)
		assert.Nil(t, hasher.Compare("somePassword", registered.Password))
	})

	t.Run("should be able to create a user with a given api key", func(t *testing.T) {
		user := fixture.User(fixture.WithAPIKey("someapikey"))

		registered := assert.EventIs[domain.UserRegistered](t, &user, 0)
		assert.Equal(t, "someapikey", registered.Key)
	})
}
