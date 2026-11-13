package types_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/types"
)

type (
	type1 struct{}
	type2 struct{}
)

func Test_Is(t *testing.T) {
	t.Run("should be able to return if an object is of a given type", func(t *testing.T) {
		var (
			t1 any = type1{}
			t2 any = type2{}
		)

		assert.True(t, types.Is[type1](t1))
		assert.False(t, types.Is[type1](t2))
	})
}
