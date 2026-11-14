package bus

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

const (
	JobPolicyRetryPreserveOrder      JobPolicy = 1 << iota // Retry the job but preserve the order among the group
	JobPolicyWaitForOthersResourceID                       // Wait for other jobs on the same resource id to finish before processing
	JobPolicyCancellable                                   // The job can be cancellable by a user
	JobPolicyMerge                                         // If another job for the same resource and the same message name exists and is pending, replace it's payload
)

type (
	JobPolicy uint8

	// Represents a schedulable request, one that can be queued for later dispatching.
	Schedulable interface {
		Request
		ResourceID() string // Resource id for which the task will be created
	}

	// Enable scheduled dispatching of a message.
	Scheduler interface {
		// Queue a request to be dispatched asynchronously at a later time.
		Queue(context.Context, Schedulable, ...JobOptions) error
	}

	// Job option passed down to adapter.
	CreateOptions struct {
		Group  monad.Maybe[string]
		Policy JobPolicy
	}

	JobOptions func(*CreateOptions)

	// Represents a scheduler that can be started and stopped. Jobs can be queued before
	// Start, they are processed once the runners are started.
	RunnableScheduler interface {
		Scheduler
		Start(...RunnerDefinition)
		Stop()
	}

	// One runner polls the store for the given message names and runs up to Workers
	// jobs at once. A runner without Messages handles every message.
	RunnerDefinition struct {
		PollInterval time.Duration // Fallback interval, runners are also woken when a job is queued or finishes
		Workers      int           // Number of jobs processed at the same time, at least 1
		Messages     []string      // Message names handled by this runner
	}

	// Represents a request that has been queued for dispatching.
	ScheduledJob interface {
		ID() string
		Message() Request
		Policy() JobPolicy
	}

	GetJobsFilters struct {
		Page   monad.Maybe[int]       `form:"page"`
		Status monad.Maybe[JobStatus] `form:"status"`
	}

	// Where a job stands in the queue, used to filter the list of jobs.
	JobStatus string

	// Adapter used to store scheduled jobs. Could be anything from a database to a file or
	// an in-memory store.
	ScheduledJobsStore interface {
		Setup() error                                                                        // Setup the store
		Create(context.Context, Schedulable, CreateOptions) error                            // Create a new scheduled job
		Delete(context.Context, string) error                                                // Try to delete a job from the store
		GetAllJobs(context.Context, GetJobsFilters) (storage.Paginated[ScheduledJob], error) // Retrieve all jobs from the store
		GetNextPendingJobs(context.Context, int, ...string) ([]ScheduledJob, error)          // Claim up to n pending jobs among the given message names (all if none)
		Release(context.Context, ...string) error                                            // Give claimed jobs back without running them
		Retry(context.Context, ScheduledJob, error) error                                    // Retry the given job with the given reason
		Done(context.Context, ScheduledJob) error                                            // Mark the given job as done
		AfterCommit(context.Context, func())                                                 // Run a function once what was written with the context is visible to runners
	}

	defaultScheduler struct {
		dispatcher Dispatcher
		logger     log.Logger
		store      ScheduledJobsStore

		mu       sync.RWMutex
		started  bool
		runners  []*runner
		byName   map[string]*runner
		catchAll *runner
	}

	runner struct {
		scheduler *defaultScheduler
		def       RunnerDefinition
		busy      atomic.Int32  // Jobs handed to workers and not finished yet
		wakeCh    chan struct{} // Buffered (1), asks the poller to check for jobs now
		done      chan struct{} // Closed on stop
		jobs      chan ScheduledJob
		exitGroup sync.WaitGroup
	}
)

var ErrJobPanicked = errors.New("job panicked")

// Builds up a new scheduler used to queue messages for later dispatching using the
// provided adapter. Runners are given to Start.
func NewScheduler(adapter ScheduledJobsStore, log log.Logger, dispatcher Dispatcher) RunnableScheduler {
	return &defaultScheduler{
		dispatcher: dispatcher,
		logger:     log,
		store:      adapter,
	}
}

func (s *defaultScheduler) Queue(
	ctx context.Context,
	msg Schedulable,
	options ...JobOptions,
) error {
	var opts CreateOptions

	for _, opt := range options {
		opt(&opts)
	}

	if err := s.store.Create(ctx, msg, opts); err != nil {
		return err
	}

	if r := s.runnerFor(msg.Name_()); r != nil {
		s.store.AfterCommit(ctx, r.wake)
	}

	return nil
}

func (s *defaultScheduler) Start(definitions ...RunnerDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return
	}

	s.started = true
	s.runners = make([]*runner, len(definitions))
	s.byName = make(map[string]*runner)
	s.catchAll = nil

	for i, def := range definitions {
		r := newRunner(s, def)
		s.runners[i] = r

		if len(def.Messages) == 0 {
			s.catchAll = r
		}

		for _, name := range def.Messages {
			s.byName[name] = r
		}
	}

	for _, r := range s.runners {
		r.start()
	}
}

// Stops polling and waits for running jobs to finish. Jobs claimed but not started
// yet are given back to the store.
func (s *defaultScheduler) Stop() {
	s.mu.Lock()

	if !s.started {
		s.mu.Unlock()
		return
	}

	// Running jobs may queue other jobs (Queue takes the read lock), so the lock is
	// released before waiting for them.
	s.started = false
	runners := s.runners
	s.mu.Unlock()

	s.logger.Info("waiting for current jobs to finish")

	for _, r := range runners {
		close(r.done)
	}

	for _, r := range runners {
		r.exitGroup.Wait()
	}
}

func (s *defaultScheduler) runnerFor(name string) *runner {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil
	}

	if r, found := s.byName[name]; found {
		return r
	}

	return s.catchAll
}

func newRunner(s *defaultScheduler, def RunnerDefinition) *runner {
	if def.Workers < 1 {
		def.Workers = 1
	}

	if def.PollInterval <= 0 {
		def.PollInterval = time.Second
	}

	return &runner{
		scheduler: s,
		def:       def,
		wakeCh:    make(chan struct{}, 1),
		done:      make(chan struct{}),
		jobs:      make(chan ScheduledJob),
	}
}

func (r *runner) start() {
	r.scheduler.logger.Debugw("starting runner",
		"poll", r.def.PollInterval,
		"workers", r.def.Workers,
		"jobs", r.def.Messages)

	for range r.def.Workers {
		r.exitGroup.Add(1)
		go r.work()
	}

	r.exitGroup.Add(1)
	go r.poll()
}

// Non-blocking: a pending wake-up already covers this one.
func (r *runner) wake() {
	select {
	case r.wakeCh <- struct{}{}:
	default:
	}
}

// The poller is the only sender on r.jobs and closes it when it returns, so the
// workers stop once they finish their current job.
func (r *runner) poll() {
	defer r.exitGroup.Done()
	defer close(r.jobs)

	ticker := time.NewTicker(r.def.PollInterval)
	defer ticker.Stop()

	for {
		if !r.dispatchPendingJobs() {
			return
		}

		select {
		case <-r.done:
			return
		case <-ticker.C:
		case <-r.wakeCh:
		}
	}
}

// Claims as many jobs as there are idle workers and hands them over. Returns false
// if the runner has been stopped meanwhile.
func (r *runner) dispatchPendingJobs() bool {
	idle := r.def.Workers - int(r.busy.Load())

	if idle <= 0 {
		return true
	}

	ctx := context.Background()
	jobs, err := r.scheduler.store.GetNextPendingJobs(ctx, idle, r.def.Messages...)

	if err != nil {
		r.scheduler.logger.Errorw("error while retrieving pending jobs",
			"error", err)
		return true
	}

	for i, job := range jobs {
		// Counted before the hand-over so the next poll never asks for more than
		// the idle workers.
		r.busy.Add(1)

		select {
		case r.jobs <- job:
		case <-r.done:
			r.busy.Add(-1)
			r.release(ctx, jobs[i:])
			return false
		}
	}

	return true
}

func (r *runner) release(ctx context.Context, jobs []ScheduledJob) {
	ids := make([]string, len(jobs))

	for i, job := range jobs {
		ids[i] = job.ID()
	}

	if err := r.scheduler.store.Release(ctx, ids...); err != nil {
		r.scheduler.logger.Errorw("error while releasing jobs not started before stopping",
			"jobs", ids,
			"error", err)
	}
}

func (r *runner) work() {
	defer r.exitGroup.Done()

	for job := range r.jobs {
		ctx := context.Background()
		err := r.run(ctx, job)

		r.scheduler.handleJobReturn(ctx, job, err)
		r.busy.Add(-1)
		r.wake() // A worker is free, check for the next job now
	}
}

// Sends the job message on the bus. A panic in the handler is turned into an error
// so the job is retried and the process keeps running.
func (r *runner) run(ctx context.Context, job ScheduledJob) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: %v", ErrJobPanicked, recovered)

			r.scheduler.logger.Errorw("job panicked",
				"job", job.ID(),
				"name", job.Message().Name_(),
				"panic", recovered,
				"stack", string(debug.Stack()))
		}
	}()

	_, err = r.scheduler.dispatcher.Send(ctx, job.Message())

	return err
}

func (s *defaultScheduler) handleJobReturn(ctx context.Context, job ScheduledJob, err error) {
	if err == nil {
		if err = s.store.Done(ctx, job); err != nil {
			s.logger.Errorw("error while marking job as done",
				"job", job.ID(),
				"name", job.Message().Name_(),
				"error", err)
		}
		return
	}

	s.logger.Warnw("error while processing job, it will be retried later",
		"job", job.ID(),
		"name", job.Message().Name_(),
		"error", err)

	if err = s.store.Retry(ctx, job, err); err != nil {
		s.logger.Errorw("error while retrying job",
			"job", job.ID(),
			"name", job.Message().Name_(),
			"error", err)
	}
}

const (
	JobStatusRunning JobStatus = "running" // Retrieved by a worker
	JobStatusFailed  JobStatus = "failed"  // Waiting for a retry after an error
	JobStatusPending JobStatus = "pending" // Waiting for its turn, never failed
)

// Attach the job being queued to a specific group meaning only one job of a group
// can be processed at a time.
func WithGroup(name string) JobOptions {
	return func(o *CreateOptions) {
		o.Group.Set(name)
	}
}

// Attach given policies to the job being queued. It will determine how the job
// will be handled.
func WithPolicy(policy JobPolicy) JobOptions {
	return func(o *CreateOptions) {
		o.Policy = policy
	}
}
