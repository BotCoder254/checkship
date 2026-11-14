package docker_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/provider/docker"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

func Test_Logs(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))

	arrange := func(t *testing.T, fake *logsAPI) docker.Docker {
		cli := &logsCli{api: fake}
		return docker.New(logger, docker.WithTestConfig(cli, cli, filepath.Join(t.TempDir(), "config")))
	}

	t.Run("should return ErrNoContainers when nothing ever ran", func(t *testing.T) {
		provider := arrange(t, newLogsAPI())

		err := provider.Logs(context.Background(), target, "app", domain.Production, domain.LogsOptions{Tail: 10}, newConsumer())

		assert.ErrorIs(t, domain.ErrNoContainers, err)
	})

	t.Run("should filter containers on the application, target and environment labels", func(t *testing.T) {
		fake := newLogsAPI()
		provider := arrange(t, fake)

		_ = provider.Logs(context.Background(), target, "app", domain.Staging, domain.LogsOptions{}, newConsumer())

		labels := fake.listed.Get("label")
		assert.ArrayEqual(t, []string{
			docker.AppLabel + "=app",
			docker.TargetLabel + "=" + string(target.ID()),
			docker.EnvironmentLabel + "=staging",
		}, labels)
	})

	t.Run("should merge the lines read back in time order, then follow started containers", func(t *testing.T) {
		fake := newLogsAPI()
		fake.add("c1", "/web-1", "web", false,
			frame(1, "2026-11-14T23:41:02.000000000Z GET / 200\n"),
			frame(1, "2026-11-14T23:41:05.000000000Z GET /healthz 200\n"))
		fake.add("c2", "/db-1", "db", false,
			frame(2, "2026-11-14T23:41:03.000000000Z FATAL: too many connections\n"))
		provider := arrange(t, fake)
		consumer := newConsumer()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)

		go func() {
			done <- provider.Logs(ctx, target, "app", domain.Production, domain.LogsOptions{Tail: 50}, consumer)
		}()

		lines := consumer.waitLines(t, 3)
		consumer.mu.Lock()
		assert.Equal(t, 1, consumer.ready)
		consumer.mu.Unlock()
		assert.Equal(t, "GET / 200", lines[0].Text)
		assert.Equal(t, "FATAL: too many connections", lines[1].Text)
		assert.Equal(t, domain.LogStreamStderr, lines[1].Stream)
		assert.Equal(t, "db", lines[1].Service)
		assert.Equal(t, "db-1", lines[1].Container)
		assert.Equal(t, "GET /healthz 200", lines[2].Text)
		assert.Equal(t, "50", fake.logOptions("c1").Tail)

		// A redeploy: the old container dies, a new one starts.
		fake.add("c3", "/web-2", "web", true, []byte("2026-11-14T23:50:00.000000000Z \x1b[32mready\x1b[0m\r\n"))
		fake.events <- events.Message{Action: "die", Actor: events.Actor{ID: "c1", Attributes: map[string]string{"name": "web-1", api.ServiceLabel: "web", "exitCode": "137"}}, TimeNano: time.Now().UnixNano()}
		fake.events <- events.Message{Action: "start", Actor: events.Actor{ID: "c3", Attributes: map[string]string{"name": "web-2", api.ServiceLabel: "web"}}, TimeNano: time.Now().UnixNano()}
		fake.events <- events.Message{Action: "health_status: unhealthy", Actor: events.Actor{ID: "c3", Attributes: map[string]string{"name": "web-2", api.ServiceLabel: "web"}}, TimeNano: time.Now().UnixNano()}

		lines = consumer.waitLines(t, 4)
		assert.Equal(t, "\x1b[32mready\x1b[0m", lines[3].Text)
		assert.Equal(t, "all", fake.logOptions("c3").Tail)
		assert.NotZero(t, fake.logOptions("c3").Since)

		evts := consumer.waitEvents(t, 3)
		assert.Equal(t, domain.ServiceActionDie, evts[0].Action)
		assert.Equal(t, 137, evts[0].ExitCode.Get(0))
		assert.Equal(t, domain.ServiceActionStart, evts[1].Action)
		assert.Equal(t, domain.ServiceActionHealth, evts[2].Action)
		assert.Equal(t, "unhealthy", evts[2].Health.Get(""))

		cancel()

		select {
		case err := <-done:
			assert.Nil(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Logs did not return after the context was cancelled")
		}
	})

	t.Run("should read back from the given time when resuming", func(t *testing.T) {
		fake := newLogsAPI()
		fake.add("c1", "/web-1", "web", false)
		provider := arrange(t, fake)
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		since := time.Date(2026, 11, 14, 23, 41, 2, 118000000, time.UTC)

		var options domain.LogsOptions
		options.Tail = 200
		options.Since.Set(since)

		assert.Nil(t, provider.Logs(ctx, target, "app", domain.Production, options, newConsumer()))
		assert.Equal(t, "2026-11-14T23:41:02.118Z", fake.logOptions("c1").Since)
		assert.Equal(t, "all", fake.logOptions("c1").Tail)
	})

	t.Run("should return ErrLogsUnavailable when the logging driver can't be read", func(t *testing.T) {
		fake := newLogsAPI()
		fake.add("c1", "/web-1", "web", false)
		fake.logsErr = errdefs.NotImplemented(errors.New("configured logging driver does not support reading"))
		provider := arrange(t, fake)

		err := provider.Logs(context.Background(), target, "app", domain.Production, domain.LogsOptions{}, newConsumer())

		assert.ErrorIs(t, domain.ErrLogsUnavailable, err)
	})
}

func Test_Services(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))

	t.Run("should describe each container, sorted by service", func(t *testing.T) {
		fake := newLogsAPI()
		fake.add("c1", "/web-1", "web", false)
		fake.add("c2", "/migrate-1", "migrate", false)
		fake.inspect["c1"] = inspected("running", 2, "healthy", 0, "2026-11-14T20:00:00Z")
		fake.inspect["c2"] = inspected("exited", 0, "", 3, "2026-11-14T19:00:00Z")
		cli := &logsCli{api: fake}
		provider := docker.New(logger, docker.WithTestConfig(cli, cli, filepath.Join(t.TempDir(), "config")))

		states, err := provider.Services(context.Background(), target, "app", domain.Production)

		assert.Nil(t, err)
		assert.HasLength(t, 2, states)
		assert.Equal(t, "migrate", states[0].Service)
		assert.Equal(t, 3, states[0].ExitCode.Get(-1))
		assert.False(t, states[0].Health.HasValue())
		assert.Equal(t, "web", states[1].Service)
		assert.Equal(t, "web-1", states[1].Container)
		assert.Equal(t, 2, states[1].RestartCount)
		assert.Equal(t, "healthy", states[1].Health.Get(""))
		assert.False(t, states[1].ExitCode.HasValue())
		assert.True(t, states[1].StartedAt.HasValue())
	})
}

// Builds a multiplexed frame: stream byte, 3 zero bytes, big-endian size, payload.
func frame(stream byte, payload string) []byte {
	header := make([]byte, 8)
	header[0] = stream
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, payload...)
}

func inspected(status string, restarts int, health string, exitCode int, startedAt string) dockertypes.ContainerJSON {
	state := &dockertypes.ContainerState{Status: status, ExitCode: exitCode, StartedAt: startedAt}

	if health != "" {
		state.Health = &dockertypes.Health{Status: health}
	}

	return dockertypes.ContainerJSON{
		ContainerJSONBase: &dockertypes.ContainerJSONBase{ID: "x", RestartCount: restarts, State: state},
		Config:            &container.Config{},
	}
}

type (
	logsCli struct {
		command.Cli
		api.Service
		api *logsAPI
	}

	fakeContainer struct {
		summary dockertypes.Container
		tty     bool
		logs    []byte
	}

	logsAPI struct {
		client.APIClient

		mu         sync.Mutex
		containers []fakeContainer
		inspect    map[string]dockertypes.ContainerJSON
		options    map[string]container.LogsOptions
		listed     filtersArgs
		logsErr    error
		events     chan events.Message
	}

	filtersArgs interface{ Get(string) []string }

	consumer struct {
		mu     sync.Mutex
		ready  int
		lines  []domain.LogLine
		events []domain.ServiceEvent
	}
)

func (c *logsCli) Client() client.APIClient { return c.api }

func newLogsAPI() *logsAPI {
	return &logsAPI{
		inspect: make(map[string]dockertypes.ContainerJSON),
		options: make(map[string]container.LogsOptions),
		events:  make(chan events.Message, 10),
	}
}

func (a *logsAPI) add(id, name, service string, tty bool, logs ...[]byte) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.containers = append(a.containers, fakeContainer{
		summary: dockertypes.Container{ID: id, Names: []string{name}, Labels: map[string]string{api.ServiceLabel: service}, State: "running", Status: "Up 3 hours"},
		tty:     tty,
		logs:    bytes.Join(logs, nil),
	})
}

func (a *logsAPI) logOptions(id string) container.LogsOptions {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.options[id]
}

func (a *logsAPI) Close() error { return nil }

func (a *logsAPI) ContainerList(_ context.Context, options container.ListOptions) ([]dockertypes.Container, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.listed = options.Filters
	result := make([]dockertypes.Container, 0, len(a.containers))

	// Containers added after the stream opened are only found through events.
	for _, c := range a.containers {
		result = append(result, c.summary)
	}

	return result, nil
}

func (a *logsAPI) ContainerInspect(_ context.Context, id string) (dockertypes.ContainerJSON, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if inspected, found := a.inspect[id]; found {
		return inspected, nil
	}

	for _, c := range a.containers {
		if c.summary.ID == id {
			return dockertypes.ContainerJSON{
				ContainerJSONBase: &dockertypes.ContainerJSONBase{ID: id},
				Config:            &container.Config{Tty: c.tty},
			}, nil
		}
	}

	return dockertypes.ContainerJSON{}, errdefs.NotFound(errors.New("no such container"))
}

func (a *logsAPI) ContainerLogs(_ context.Context, id string, options container.LogsOptions) (io.ReadCloser, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.logsErr != nil {
		return nil, a.logsErr
	}

	a.options[id] = options

	for _, c := range a.containers {
		if c.summary.ID == id {
			return io.NopCloser(bytes.NewReader(c.logs)), nil
		}
	}

	return nil, errdefs.NotFound(errors.New("no such container"))
}

func (a *logsAPI) Events(ctx context.Context, _ events.ListOptions) (<-chan events.Message, <-chan error) {
	return a.events, make(chan error)
}

func newConsumer() *consumer { return &consumer{} }

func (c *consumer) Ready() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready++
}

func (c *consumer) Line(l domain.LogLine) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = append(c.lines, l)
}

func (c *consumer) Event(e domain.ServiceEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *consumer) waitLines(t *testing.T, count int) []domain.LogLine {
	t.Helper()

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		c.mu.Lock()
		if len(c.lines) >= count {
			lines := append([]domain.LogLine(nil), c.lines...)
			c.mu.Unlock()
			return lines
		}
		c.mu.Unlock()
	}

	t.Fatalf("expected %d lines", count)
	return nil
}

func (c *consumer) waitEvents(t *testing.T, count int) []domain.ServiceEvent {
	t.Helper()

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		c.mu.Lock()
		if len(c.events) >= count {
			evts := append([]domain.ServiceEvent(nil), c.events...)
			c.mu.Unlock()
			return evts
		}
		c.mu.Unlock()
	}

	t.Fatalf("expected %d events", count)
	return nil
}
