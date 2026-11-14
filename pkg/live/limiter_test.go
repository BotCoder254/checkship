package live_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/live"
)

func Test_Limiter(t *testing.T) {
	t.Run("should refuse a stream past the per user limit", func(t *testing.T) {
		limiter := live.NewLimiter(2, 10)

		_, err := limiter.Acquire("a")
		assert.Nil(t, err)
		_, err = limiter.Acquire("a")
		assert.Nil(t, err)

		_, err = limiter.Acquire("a")
		assert.ErrorIs(t, live.ErrTooManyStreams, err)

		_, err = limiter.Acquire("b")
		assert.Nil(t, err, "another user is not affected")
	})

	t.Run("should refuse a stream past the total limit", func(t *testing.T) {
		limiter := live.NewLimiter(5, 2)

		_, _ = limiter.Acquire("a")
		_, _ = limiter.Acquire("b")

		_, err := limiter.Acquire("c")
		assert.ErrorIs(t, live.ErrTooManyStreams, err)
	})

	t.Run("should give a slot back once, however many times release is called", func(t *testing.T) {
		limiter := live.NewLimiter(1, 2)

		release, err := limiter.Acquire("a")
		assert.Nil(t, err)
		other, err := limiter.Acquire("b")
		assert.Nil(t, err)

		release()
		release()

		assert.Equal(t, 1, limiter.Count())

		_, err = limiter.Acquire("a")
		assert.Nil(t, err)

		other()
		assert.Equal(t, 1, limiter.Count())
	})
}
