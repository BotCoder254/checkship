// Package live fans small "something changed" hints out to connected clients (server-sent
// events). Hints carry a topic and an id, never data: clients refetch what they show with their
// own credentials, so a hint can't leak anything and a lost or spurious hint is harmless.
package live

import (
	"errors"
	"sync"
)

var (
	ErrTooManyStreams = errors.New("too_many_streams")
	ErrClosed         = errors.New("live_closed")
)

type (
	Hint struct {
		Topic string `json:"topic"`        // deployments, targets or apps
		ID    string `json:"id,omitempty"` // the app or target concerned, when known
	}

	Hub struct {
		mu         sync.Mutex
		clients    map[*client]struct{}
		perUser    map[string]int
		maxPerUser int
		maxTotal   int
		buffer     int
		closed     bool
	}

	client struct {
		user string
		ch   chan Hint
	}
)

// Build a hub accepting at most maxPerUser streams per user and maxTotal in all. Each client
// buffers up to buffer hints; when full, new hints for that client are dropped (it will catch up
// with the next hint or its periodic refresh).
func NewHub(maxPerUser, maxTotal, buffer int) *Hub {
	return &Hub{
		clients:    map[*client]struct{}{},
		perUser:    map[string]int{},
		maxPerUser: maxPerUser,
		maxTotal:   maxTotal,
		buffer:     buffer,
	}
}

// Open a stream for the given user. The returned function must be called once the stream ends.
// The channel is closed when the hub closes.
func (h *Hub) Subscribe(user string) (<-chan Hint, func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, nil, ErrClosed
	}

	if len(h.clients) >= h.maxTotal || h.perUser[user] >= h.maxPerUser {
		return nil, nil, ErrTooManyStreams
	}

	c := &client{user: user, ch: make(chan Hint, h.buffer)}
	h.clients[c] = struct{}{}
	h.perUser[user]++

	var once sync.Once

	return c.ch, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()

			if _, ok := h.clients[c]; !ok {
				return // already removed by Close
			}

			delete(h.clients, c)
			h.release(c.user)
			close(c.ch)
		})
	}, nil
}

// Send a hint to every client without ever blocking: it is called from inside write transactions.
func (h *Hub) Publish(hint Hint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c := range h.clients {
		select {
		case c.ch <- hint:
		default: // slow client: drop, it refreshes on its own
		}
	}
}

// Close every stream and refuse new ones (server shutdown).
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	h.closed = true

	for c := range h.clients {
		delete(h.clients, c)
		h.release(c.user)
		close(c.ch)
	}
}

// Number of open streams, for tests and diagnostics.
func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.clients)
}

func (h *Hub) release(user string) {
	if h.perUser[user] <= 1 {
		delete(h.perUser, user)
		return
	}
	h.perUser[user]--
}
