package live_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/live"
)

func Test_Hub(t *testing.T) {
	t.Run("should fan hints out to every stream", func(t *testing.T) {
		hub := live.NewHub(5, 100, 4)
		a, closeA, _ := hub.Subscribe("u1")
		b, closeB, _ := hub.Subscribe("u2")
		defer closeA()
		defer closeB()

		hub.Publish(live.Hint{Topic: "deployments", ID: "app"})

		assert.Equal(t, live.Hint{Topic: "deployments", ID: "app"}, <-a)
		assert.Equal(t, live.Hint{Topic: "deployments", ID: "app"}, <-b)
	})

	t.Run("should drop hints for a slow client instead of blocking", func(t *testing.T) {
		hub := live.NewHub(5, 100, 1)
		ch, unsubscribe, _ := hub.Subscribe("u1")
		defer unsubscribe()

		hub.Publish(live.Hint{Topic: "apps"})
		hub.Publish(live.Hint{Topic: "targets"}) // buffer full: dropped, and Publish returns

		assert.Equal(t, live.Hint{Topic: "apps"}, <-ch)
		select {
		case h := <-ch:
			t.Fatalf("unexpected hint %v", h)
		default:
		}
	})

	t.Run("should limit streams per user and in all", func(t *testing.T) {
		hub := live.NewHub(2, 3, 1)
		_, c1, _ := hub.Subscribe("u1")
		_, _, _ = hub.Subscribe("u1")

		_, _, err := hub.Subscribe("u1")
		assert.ErrorIs(t, live.ErrTooManyStreams, err)

		_, _, err = hub.Subscribe("u2")
		assert.Nil(t, err)
		_, _, err = hub.Subscribe("u3")
		assert.ErrorIs(t, live.ErrTooManyStreams, err)

		c1()
		c1() // idempotent
		_, _, err = hub.Subscribe("u1")
		assert.Nil(t, err)
		assert.Equal(t, 3, hub.Count())
	})

	t.Run("should close every stream and refuse new ones", func(t *testing.T) {
		hub := live.NewHub(5, 100, 1)
		ch, unsubscribe, _ := hub.Subscribe("u1")

		hub.Close()
		unsubscribe() // safe after Close

		_, open := <-ch
		assert.False(t, open)
		_, _, err := hub.Subscribe("u1")
		assert.ErrorIs(t, live.ErrClosed, err)
		assert.Equal(t, 0, hub.Count())
	})
}
