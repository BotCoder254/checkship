package numbers_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/validate/numbers"
)

func Test_Min(t *testing.T) {
	t.Run("should fail on value lesser than the required min", func(t *testing.T) {
		assert.ErrorIs(t, numbers.ErrMin, numbers.Min(3)(2))
		assert.ErrorIs(t, numbers.ErrMin, numbers.Min(3)(1))
	})

	t.Run("should succeed on value greater then the required min", func(t *testing.T) {
		assert.Nil(t, numbers.Min(3)(4))
		assert.Nil(t, numbers.Min(3)(3))
	})
}
