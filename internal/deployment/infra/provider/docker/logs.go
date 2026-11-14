package docker

import (
	"context"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dclient "github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

// Lines of every container arriving within this window after the stream opens are
// sorted by time before being passed on, so the services read back interleave. The
// consumer is told the stream is ready at the end of it, and errors found before (no
// readable logs) are returned instead.
const initialMergeWindow = 300 * time.Millisecond

// Labels identifying the containers of an application environment on a target.
func appFilters(app domain.AppID, target domain.Target, env domain.Environment) filters.Args {
	return filters.NewArgs(
		filters.Arg("label", AppLabel+"="+string(app)),
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
		filters.Arg("label", EnvironmentLabel+"="+string(env)),
	)
}

func (d *docker) Logs(
	ctx context.Context,
	target domain.Target,
	app domain.AppID,
	env domain.Environment,
	options domain.LogsOptions,
	consumer domain.LogsConsumer,
) error {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	criteria := appFilters(app, target, env)
	containers, err := client.api.ContainerList(ctx, container.ListOptions{All: true, Filters: criteria})

	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return domain.ErrNoContainers
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	f := &follower{
		ctx:       ctx,
		cancel:    cancel,
		api:       client.api,
		consumer:  consumer,
		options:   options,
		following: make(map[string]bool),
		merging:   true,
	}

	// Events first, so a container starting while the others are read back is not missed.
	eventsSince := time.Now()
	f.watch(criteria, eventsSince)

	for _, c := range containers {
		f.follow(c.ID, containerName(c.Names), c.Labels[api.ServiceLabel], options.Since)
	}

	f.wg.Add(1)

	go func() {
		defer f.wg.Done()

		select {
		case <-ctx.Done():
		case <-time.After(initialMergeWindow):
			f.endMerge()
		}
	}()

	<-ctx.Done()
	f.wg.Wait()

	return f.err
}

func (d *docker) Services(
	ctx context.Context,
	target domain.Target,
	app domain.AppID,
	env domain.Environment,
) ([]domain.ServiceState, error) {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return nil, err
	}

	defer client.Close()

	containers, err := client.api.ContainerList(ctx, container.ListOptions{All: true, Filters: appFilters(app, target, env)})

	if err != nil {
		return nil, err
	}

	states := make([]domain.ServiceState, 0, len(containers))

	for _, c := range containers {
		state := domain.ServiceState{
			Service:   c.Labels[api.ServiceLabel],
			Container: containerName(c.Names),
			Image:     c.Image,
			State:     c.State,
			Status:    c.Status,
		}

		// Restarts, health and exit code are only known by inspecting the container.
		if inspected, err := client.api.ContainerInspect(ctx, c.ID); err == nil && inspected.ContainerJSONBase != nil {
			state.RestartCount = inspected.RestartCount

			if s := inspected.State; s != nil {
				if s.Health != nil && s.Health.Status != "" && s.Health.Status != "none" {
					state.Health.Set(s.Health.Status)
				}

				if s.Status == "exited" || s.Status == "dead" {
					state.ExitCode.Set(s.ExitCode)
				}

				if startedAt, err := time.Parse(time.RFC3339Nano, s.StartedAt); err == nil && !startedAt.IsZero() && startedAt.Year() > 1 {
					state.StartedAt.Set(startedAt)
				}
			}
		}

		states = append(states, state)
	}

	slices.SortFunc(states, func(a, b domain.ServiceState) int {
		return strings.Compare(a.Service+a.Container, b.Service+b.Container)
	})

	return states, nil
}

// Follows the logs of every container of an application environment, and the
// containers started later on (redeploys, restarts, new services).
type follower struct {
	ctx      context.Context
	cancel   context.CancelFunc
	api      dclient.APIClient
	consumer domain.LogsConsumer
	options  domain.LogsOptions
	wg       sync.WaitGroup

	mu        sync.Mutex
	following map[string]bool // Containers whose logs are being read
	merging   bool            // Initial window: lines and events are held, lines to be sorted
	held      []domain.LogLine
	heldEvts  []domain.ServiceEvent
	err       error // First error that ended the stream
}

// Ends the stream with the given error. Only the first one is kept.
func (f *follower) fail(err error) {
	f.mu.Lock()
	if f.err == nil {
		f.err = err
	}
	f.mu.Unlock()
	f.cancel()
}

func (f *follower) line(l domain.LogLine) {
	f.mu.Lock()

	if f.merging {
		f.held = append(f.held, l)
		f.mu.Unlock()
		return
	}

	f.mu.Unlock()
	f.consumer.Line(l)
}

func (f *follower) endMerge() {
	f.mu.Lock()
	held, heldEvts := f.held, f.heldEvts
	f.held, f.heldEvts = nil, nil
	f.merging = false
	failed := f.err != nil
	f.mu.Unlock()

	if failed || f.ctx.Err() != nil {
		return
	}

	sort.SliceStable(held, func(i, j int) bool { return held[i].Time.Before(held[j].Time) })

	f.consumer.Ready()

	for _, l := range held {
		f.consumer.Line(l)
	}

	for _, e := range heldEvts {
		f.consumer.Event(e)
	}
}

func (f *follower) passEvent(e domain.ServiceEvent) {
	f.mu.Lock()

	if f.merging {
		f.heldEvts = append(f.heldEvts, e)
		f.mu.Unlock()
		return
	}

	f.mu.Unlock()
	f.consumer.Event(e)
}

// Reads a container's logs in a goroutine until they end (container stopped) or the
// stream is closed. Does nothing if the container is already followed.
func (f *follower) follow(id, name, service string, since monad.Maybe[time.Time]) {
	f.mu.Lock()

	if f.following[id] {
		f.mu.Unlock()
		return
	}

	f.following[id] = true
	f.mu.Unlock()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       strconv.Itoa(max(f.options.Tail, 0)),
	}

	if at, isSet := since.TryGet(); isSet {
		options.Since = at.UTC().Format(time.RFC3339Nano)
		options.Tail = "all"
	}

	f.wg.Add(1)

	go func() {
		defer f.wg.Done()
		defer func() {
			f.mu.Lock()
			delete(f.following, id)
			f.mu.Unlock()
		}()

		err := f.read(id, name, service, options)

		if err == nil || f.ctx.Err() != nil || errdefs.IsNotFound(err) {
			return
		}

		if errdefs.IsNotImplemented(err) {
			err = domain.ErrLogsUnavailable
		}

		f.fail(err)
	}()
}

func (f *follower) read(id, name, service string, options container.LogsOptions) error {
	inspected, err := f.api.ContainerInspect(f.ctx, id)

	if err != nil {
		return err
	}

	stream, err := f.api.ContainerLogs(f.ctx, id, options)

	if err != nil {
		return err
	}

	defer stream.Close()

	emit := func(s domain.LogStream, at time.Time, text string, truncated bool) {
		f.line(domain.LogLine{
			Time:      at,
			Service:   service,
			Container: name,
			Stream:    s,
			Text:      text,
			Truncated: truncated,
		})
	}

	if inspected.Config != nil && inspected.Config.Tty {
		return readRawLogs(stream, emit)
	}

	return readMultiplexedLogs(stream, emit)
}

// Watches the containers' events: starts are followed, and state changes are passed on.
func (f *follower) watch(criteria filters.Args, since time.Time) {
	criteria = criteria.Clone()
	criteria.Add("type", string(events.ContainerEventType))

	messages, errs := f.api.Events(f.ctx, events.ListOptions{
		Filters: criteria,
		Since:   strconv.FormatInt(since.Unix(), 10),
	})

	f.wg.Add(1)

	go func() {
		defer f.wg.Done()

		for {
			select {
			case <-f.ctx.Done():
				return
			case err := <-errs:
				if err != nil && f.ctx.Err() == nil {
					f.fail(err)
				}
				return
			case msg := <-messages:
				f.event(msg)
			}
		}
	}()
}

func (f *follower) event(msg events.Message) {
	var (
		attrs   = msg.Actor.Attributes
		at      = time.Unix(0, msg.TimeNano).UTC()
		service = attrs[api.ServiceLabel]
		name    = attrs["name"]
		action  = string(msg.Action)
		evt     = domain.ServiceEvent{Time: at, Service: service, Container: name}
	)

	switch {
	case action == "start":
		evt.Action = domain.ServiceActionStart
		f.follow(msg.Actor.ID, name, service, monad.Value(at))
	case action == "die":
		evt.Action = domain.ServiceActionDie

		if code, err := strconv.Atoi(attrs["exitCode"]); err == nil {
			evt.ExitCode.Set(code)
		}
	case action == "oom":
		evt.Action = domain.ServiceActionOOM
	case action == "destroy":
		evt.Action = domain.ServiceActionDestroy
	case strings.HasPrefix(action, "health_status"):
		evt.Action = domain.ServiceActionHealth

		if _, status, found := strings.Cut(action, ": "); found {
			evt.Health.Set(status)
		}
	default:
		return
	}

	f.passEvent(evt)
}

func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}

	return strings.TrimPrefix(names[0], "/")
}
