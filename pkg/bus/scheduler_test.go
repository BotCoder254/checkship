package bus_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/memory"
	"github.com/checkship-deploy/checkship/pkg/flag"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

const waitTimeout = 5 * time.Second

func TestScheduler(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	b := memory.NewBus()

	// Returns the inner error to test how the scheduler handles job results.
	bus.Register(b, func(_ context.Context, cmd returnCommand) (bus.UnitType, error) {
		return bus.Unit, cmd.err
	})

	// Blocks until its release channel is closed, to keep a worker busy.
	bus.Register(b, func(_ context.Context, cmd blockCommand) (bus.UnitType, error) {
		cmd.started <- cmd.id
		<-cmd.release
		return bus.Unit, nil
	})

	bus.Register(b, func(_ context.Context, cmd panicCommand) (bus.UnitType, error) {
		panic("boom")
	})

	t.Run("should queue and handle the job return appropriately", func(t *testing.T) {
		adapter := newAdapter()
		scheduler := bus.NewScheduler(adapter, logger, b)

		innerErr := errors.New("some error")

		withoutErr := returnCommand{}
		withUnwrapedErr := returnCommand{err: innerErr}
		withPreservedOrderErr := returnCommand{err: innerErr}

		assert.Nil(t, scheduler.Queue(context.Background(), withoutErr))
		assert.Nil(t, scheduler.Queue(context.Background(), withUnwrapedErr))
		assert.Nil(t, scheduler.Queue(context.Background(), withPreservedOrderErr, bus.WithPolicy(bus.JobPolicyRetryPreserveOrder)))
		assert.Nil(t, scheduler.Queue(context.Background(), addCommand{}))

		// A runner without messages handles every message
		scheduler.Start(bus.RunnerDefinition{Workers: 4})
		defer scheduler.Stop()

		adapter.waitFinished(t, 4)

		done, retried := adapter.results()

		assert.HasLength(t, 1, done)
		assert.Equal(t, 0, done[0].id)

		assert.HasLength(t, 3, retried)

		assert.Equal(t, 1, retried[0].id)
		assert.ErrorIs(t, innerErr, retried[0].err)
		assert.False(t, retried[0].preserveOrder)

		assert.Equal(t, 2, retried[1].id)
		assert.ErrorIs(t, innerErr, retried[1].err)
		assert.True(t, retried[1].preserveOrder)

		assert.Equal(t, 3, retried[2].id)
		assert.ErrorIs(t, bus.ErrNoHandlerRegistered, retried[2].err)
	})

	t.Run("should not delay jobs of a runner while another runner is busy", func(t *testing.T) {
		adapter := newAdapter()
		scheduler := bus.NewScheduler(adapter, logger, b)
		block := newBlockCommand(1)

		scheduler.Start(
			bus.RunnerDefinition{PollInterval: time.Hour, Workers: 1, Messages: []string{blockCommand{}.Name_()}},
			bus.RunnerDefinition{PollInterval: time.Hour, Workers: 1, Messages: []string{returnCommand{}.Name_()}},
		)
		defer scheduler.Stop()
		defer close(block.release) // Runs before Stop, which waits for the blocked job

		assert.Nil(t, scheduler.Queue(context.Background(), block))
		block.waitStarted(t)

		assert.Nil(t, scheduler.Queue(context.Background(), returnCommand{}))
		adapter.waitFinished(t, 1)

		done, _ := adapter.results()
		assert.HasLength(t, 1, done)
		assert.Equal(t, returnCommand{}.Name_(), done[0].msg.Name_())
	})

	t.Run("should only claim as many jobs as there are idle workers", func(t *testing.T) {
		adapter := newAdapter()
		scheduler := bus.NewScheduler(adapter, logger, b)
		block := newBlockCommand(3)

		for range 3 {
			assert.Nil(t, scheduler.Queue(context.Background(), block))
		}

		scheduler.Start(bus.RunnerDefinition{PollInterval: 10 * time.Millisecond, Workers: 1})

		block.waitStarted(t)
		time.Sleep(50 * time.Millisecond) // Leave time for a few polls

		assert.Equal(t, 1, adapter.claimedCount())
		assert.Equal(t, 2, adapter.pendingCount())

		close(block.release)
		adapter.waitFinished(t, 3)
		scheduler.Stop()
	})

	t.Run("should start a queued job without waiting for the poll interval", func(t *testing.T) {
		adapter := newAdapter()
		scheduler := bus.NewScheduler(adapter, logger, b)

		scheduler.Start(bus.RunnerDefinition{PollInterval: time.Hour, Workers: 1})
		defer scheduler.Stop()

		// The second job is picked up when the worker is done with the first one
		assert.Nil(t, scheduler.Queue(context.Background(), returnCommand{}))
		assert.Nil(t, scheduler.Queue(context.Background(), returnCommand{}))

		adapter.waitFinished(t, 2)
	})

	t.Run("should retry a job whose handler panicked", func(t *testing.T) {
		adapter := newAdapter()
		scheduler := bus.NewScheduler(adapter, logger, b)

		scheduler.Start(bus.RunnerDefinition{PollInterval: time.Hour, Workers: 1})
		defer scheduler.Stop()

		assert.Nil(t, scheduler.Queue(context.Background(), panicCommand{}))
		adapter.waitFinished(t, 1)

		_, retried := adapter.results()
		assert.HasLength(t, 1, retried)
		assert.ErrorIs(t, bus.ErrJobPanicked, retried[0].err)
	})

	t.Run("should stop while workers are busy and give back jobs not started", func(t *testing.T) {
		adapter := newAdapter()
		adapter.ignoreLimit = true // Hands over more jobs than idle workers to keep the poller blocked
		scheduler := bus.NewScheduler(adapter, logger, b)
		block := newBlockCommand(2)

		assert.Nil(t, scheduler.Queue(context.Background(), block))
		assert.Nil(t, scheduler.Queue(context.Background(), block))

		scheduler.Start(bus.RunnerDefinition{PollInterval: time.Hour, Workers: 1})
		block.waitStarted(t)

		stopped := make(chan struct{})

		go func() {
			scheduler.Stop()
			close(stopped)
		}()

		// Stop waits for the running job
		select {
		case <-stopped:
			t.Fatal("Stop returned before the running job finished")
		case <-time.After(50 * time.Millisecond):
		}

		close(block.release)

		select {
		case <-stopped:
		case <-time.After(waitTimeout):
			t.Fatal("Stop did not return")
		}

		done, _ := adapter.results()
		assert.HasLength(t, 1, done)
		assert.DeepEqual(t, []string{"1"}, adapter.releasedIDs())
		assert.Equal(t, 1, adapter.pendingCount())
	})

	t.Run("should be stoppable without having started", func(t *testing.T) {
		scheduler := bus.NewScheduler(newAdapter(), logger, b)
		scheduler.Stop()
	})
}

var (
	_ bus.ScheduledJob       = (*job)(nil)
	_ bus.ScheduledJobsStore = (*adapter)(nil)
	_ bus.Request            = (*returnCommand)(nil)
)

type (
	job struct {
		id            int
		msg           bus.Request
		policy        bus.JobPolicy
		err           error
		preserveOrder bool
		claimed       bool
	}

	// In-memory store, safe for concurrent use.
	adapter struct {
		mu          sync.Mutex
		ignoreLimit bool
		jobs        []*job
		done        []*job
		retried     []*job
		released    []string
		finished    chan struct{}
	}

	returnCommand struct {
		bus.Command[bus.UnitType]

		err error
	}

	blockCommand struct {
		bus.Command[bus.UnitType]

		id      string
		started chan string
		release chan struct{}
	}

	panicCommand struct {
		bus.Command[bus.UnitType]
	}
)

func (r returnCommand) Name_() string      { return "returnCommand" }
func (r returnCommand) ResourceID() string { return "" }

func (blockCommand) Name_() string        { return "blockCommand" }
func (c blockCommand) ResourceID() string { return c.id }

func (panicCommand) Name_() string      { return "panicCommand" }
func (panicCommand) ResourceID() string { return "" }

func newBlockCommand(capacity int) blockCommand {
	return blockCommand{
		started: make(chan string, capacity),
		release: make(chan struct{}),
	}
}

func (c blockCommand) waitStarted(t testing.TB) {
	t.Helper()

	select {
	case <-c.started:
	case <-time.After(waitTimeout):
		t.Fatal("blocking job did not start")
	}
}

func (j *job) ID() string            { return strconv.Itoa(j.id) }
func (j *job) Message() bus.Request  { return j.msg }
func (j *job) Policy() bus.JobPolicy { return j.policy }

func newAdapter() *adapter {
	return &adapter{finished: make(chan struct{}, 100)}
}

func (a *adapter) Setup() error { return nil }

func (a *adapter) GetAllJobs(context.Context, bus.GetJobsFilters) (storage.Paginated[bus.ScheduledJob], error) {
	return storage.Paginated[bus.ScheduledJob]{}, nil
}

func (a *adapter) Create(_ context.Context, msg bus.Schedulable, opts bus.CreateOptions) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.jobs = append(a.jobs, &job{id: len(a.jobs), msg: msg, policy: opts.Policy})
	return nil
}

func (a *adapter) Delete(context.Context, string) error { return nil }

func (a *adapter) AfterCommit(_ context.Context, fn func()) { fn() }

func (a *adapter) GetNextPendingJobs(_ context.Context, limit int, names ...string) ([]bus.ScheduledJob, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var result []bus.ScheduledJob

	for _, j := range a.jobs {
		if !a.ignoreLimit && len(result) >= limit {
			break
		}

		if j.claimed || (len(names) > 0 && !slices.Contains(names, j.msg.Name_())) {
			continue
		}

		j.claimed = true
		result = append(result, j)
	}

	return result, nil
}

func (a *adapter) Release(_ context.Context, ids ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.released = append(a.released, ids...)

	for _, j := range a.jobs {
		if slices.Contains(ids, j.ID()) {
			j.claimed = false
		}
	}

	return nil
}

func (a *adapter) Retry(_ context.Context, j bus.ScheduledJob, jobErr error) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	jo := j.(*job)
	jo.err = jobErr
	jo.preserveOrder = flag.IsSet(j.Policy(), bus.JobPolicyRetryPreserveOrder)

	a.retried = append(a.retried, jo)
	a.remove(jo)
	a.finished <- struct{}{}

	return nil
}

func (a *adapter) Done(_ context.Context, j bus.ScheduledJob) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.done = append(a.done, j.(*job))
	a.remove(j.(*job))
	a.finished <- struct{}{}

	return nil
}

// Retried jobs are removed too so they are not picked up again during the test.
func (a *adapter) remove(j *job) {
	a.jobs = slices.DeleteFunc(a.jobs, func(other *job) bool { return other == j })
}

func (a *adapter) waitFinished(t testing.TB, count int) {
	t.Helper()

	for i := range count {
		select {
		case <-a.finished:
		case <-time.After(waitTimeout):
			t.Fatalf("only %d of %d jobs finished", i, count)
		}
	}
}

// Done and retried jobs sorted by id.
func (a *adapter) results() (done, retried []*job) {
	a.mu.Lock()
	defer a.mu.Unlock()

	byID := func(x, y *job) int { return x.id - y.id }

	return slices.SortedFunc(slices.Values(a.done), byID), slices.SortedFunc(slices.Values(a.retried), byID)
}

func (a *adapter) claimedCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.claimedCountLocked()
}

func (a *adapter) pendingCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return len(a.jobs) - a.claimedCountLocked()
}

func (a *adapter) claimedCountLocked() (count int) {
	for _, j := range a.jobs {
		if j.claimed {
			count++
		}
	}

	return count
}

func (a *adapter) releasedIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return slices.Clone(a.released)
}
