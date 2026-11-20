package live_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/live"
)

func Test_Signals(t *testing.T) {
	t.Run("should wake every waiter of the key", func(t *testing.T) {
		s := live.NewSignals()
		a, doneA := s.Wait("k")
		b, doneB := s.Wait("k")
		other, doneOther := s.Wait("other")
		defer doneA()
		defer doneB()
		defer doneOther()

		s.Notify("k")

		assert.True(t, isClosed(a))
		assert.True(t, isClosed(b))
		assert.False(t, isClosed(other))
	})

	t.Run("should not block when nobody waits", func(t *testing.T) {
		s := live.NewSignals()

		s.Notify("k")

		assert.Equal(t, 0, s.Count())
	})

	t.Run("should give a new channel after a notification", func(t *testing.T) {
		s := live.NewSignals()
		first, done := s.Wait("k")
		s.Notify("k")
		done()

		second, done := s.Wait("k")
		defer done()

		assert.True(t, isClosed(first))
		assert.False(t, isClosed(second))
	})

	t.Run("should free the key once nobody waits", func(t *testing.T) {
		s := live.NewSignals()
		_, doneA := s.Wait("k")
		_, doneB := s.Wait("k")

		doneA()
		assert.Equal(t, 1, s.Count())
		doneB()
		doneB() // Calling it twice is harmless
		assert.Equal(t, 0, s.Count())
	})
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
