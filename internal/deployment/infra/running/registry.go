// Package running keeps track, in memory, of the deployments this process is deploying so
// they can be cancelled while they run. checkship is a single process: the scheduler runs
// every deploy job, so an in-memory map is the whole picture.
package running

import (
	"context"
	"sync"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
)

type (
	registry struct {
		mu      sync.Mutex
		running map[domain.DeploymentID]*entry
	}

	entry struct {
		cancel context.CancelCauseFunc
	}
)

// Build an empty registry. Its size is bounded by the number of deploy workers.
func New() domain.RunningDeployments {
	return &registry{running: make(map[domain.DeploymentID]*entry)}
}

func (r *registry) Track(ctx context.Context, id domain.DeploymentID) (context.Context, func()) {
	tracked, cancel := context.WithCancelCause(ctx)
	e := &entry{cancel: cancel}

	r.mu.Lock()
	// Should not happen (jobs of the same deployment never run together), but if it does the
	// newest job wins and the older one keeps its own context.
	r.running[id] = e
	r.mu.Unlock()

	var once sync.Once

	return tracked, func() {
		once.Do(func() {
			r.mu.Lock()
			if r.running[id] == e {
				delete(r.running, id)
			}
			r.mu.Unlock()
			cancel(context.Canceled) // Release the context resources, the job is done
		})
	}
}

func (r *registry) Cancel(id domain.DeploymentID) bool {
	r.mu.Lock()
	e, found := r.running[id]
	r.mu.Unlock()

	if !found {
		return false
	}

	e.cancel(domain.ErrDeploymentCancelled)

	return true
}
