package live

import "sync"

type (
	// Wakes up whoever waits on a key when something happens to it, like a deployment log being
	// written. A notification carries nothing and is never queued: waiters read what changed
	// themselves, so a missed one only delays them until their own fallback timer.
	Signals struct {
		mu      sync.Mutex
		signals map[string]*signal
	}

	signal struct {
		ch      chan struct{}
		waiters int
	}
)

func NewSignals() *Signals {
	return &Signals{signals: make(map[string]*signal)}
}

// Returns a channel closed on the next Notify for the key, and a function to call when no
// longer waiting on it (it frees the channel when nobody else waits).
func (s *Signals) Wait(key string) (<-chan struct{}, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sig, found := s.signals[key]

	if !found {
		sig = &signal{ch: make(chan struct{})}
		s.signals[key] = sig
	}

	sig.waiters++

	var once sync.Once

	return sig.ch, func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			sig.waiters--

			if sig.waiters <= 0 && s.signals[key] == sig {
				delete(s.signals, key)
			}
		})
	}
}

// Wakes up everyone waiting on the key. Never blocks, cheap when nobody waits.
func (s *Signals) Notify(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sig, found := s.signals[key]; found {
		close(sig.ch)
		delete(s.signals, key)
	}
}

// Number of keys waited on, for tests.
func (s *Signals) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.signals)
}
