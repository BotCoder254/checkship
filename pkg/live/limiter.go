package live

import "sync"

// Counts long-lived streams per user and in all, and refuses new ones past the limits.
// Used for streams that hold resources elsewhere, such as runtime logs which keep
// connections open to a target.
type Limiter struct {
	mu         sync.Mutex
	perUser    map[string]int
	total      int
	maxPerUser int
	maxTotal   int
}

func NewLimiter(maxPerUser, maxTotal int) *Limiter {
	return &Limiter{
		perUser:    map[string]int{},
		maxPerUser: maxPerUser,
		maxTotal:   maxTotal,
	}
}

// Takes a slot for the given user. The returned function gives it back and can be
// called more than once.
func (l *Limiter) Acquire(user string) (func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.total >= l.maxTotal || l.perUser[user] >= l.maxPerUser {
		return nil, ErrTooManyStreams
	}

	l.total++
	l.perUser[user]++

	var once sync.Once

	return func() {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()

			l.total--

			if l.perUser[user] <= 1 {
				delete(l.perUser, user)
			} else {
				l.perUser[user]--
			}
		})
	}, nil
}

// Number of slots taken, for tests and diagnostics.
func (l *Limiter) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.total
}
