package crypto_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/crypto"
)

func Test_RandomKey(t *testing.T) {
	key, err := crypto.RandomKey[string](32)
	assert.Nil(t, err)
	assert.HasNRunes(t, 32, key)
}
