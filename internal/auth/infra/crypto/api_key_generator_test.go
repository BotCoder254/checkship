package crypto_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/internal/auth/infra/crypto"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_KeyGenerator(t *testing.T) {
	t.Run("should generate an API key", func(t *testing.T) {
		generator := crypto.NewKeyGenerator()
		key, err := generator.Generate()

		assert.Nil(t, err)
		assert.HasNRunes(t, 64, key)
	})
}
