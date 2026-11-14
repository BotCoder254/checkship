package serve

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_logs"
	deployment "github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/memory"
	"github.com/checkship-deploy/checkship/pkg/live"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/gin-gonic/gin"
)

func Test_AppLogs(t *testing.T) {
	type arranged struct {
		url      string
		queries  func() []get_app_logs.Query
		shutdown context.CancelFunc
	}

	// The stream function given here runs for every request.
	arrange := func(t *testing.T, limiter *live.Limiter, stream deployment.LogsStream, prepareErr error) arranged {
		var (
			mu      sync.Mutex
			queries []get_app_logs.Query
			b       = memory.NewBus()
		)

		bus.Register(b, func(_ context.Context, q get_app_logs.Query) (deployment.LogsStream, error) {
			mu.Lock()
			queries = append(queries, q)
			mu.Unlock()
			return stream, prepareErr
		})

		shutdown, stop := context.WithCancel(context.Background())
		s := &server{bus: b, logger: must.Panic(log.NewLogger()), logsLimiter: limiter, shutdown: shutdown, stopStreams: stop}
		router := gin.New()
		router.GET("/apps/:id/logs", func(c *gin.Context) {
			c.Request = c.Request.WithContext(domain.WithUserID(c.Request.Context(), "u1"))
		}, s.appLogsHandler())
		srv := httptest.NewServer(router)
		t.Cleanup(srv.Close)
		t.Cleanup(stop)

		return arranged{
			url:      srv.URL + "/apps/app1/logs",
			shutdown: stop,
			queries: func() []get_app_logs.Query {
				mu.Lock()
				defer mu.Unlock()
				return append([]get_app_logs.Query(nil), queries...)
			},
		}
	}

	at := func(sec int) time.Time { return time.Date(2026, 11, 14, 23, 41, sec, 118000000, time.UTC) }

	t.Run("should stream lines in batches and state events", func(t *testing.T) {
		a := arrange(t, live.NewLimiter(3, 20), func(ctx context.Context, c deployment.LogsConsumer) error {
			c.Ready()
			c.Line(deployment.LogLine{Time: at(2), Service: "web", Stream: deployment.LogStreamStdout, Text: "GET / 200"})
			c.Line(deployment.LogLine{Time: at(3), Service: "db", Stream: deployment.LogStreamStderr, Text: "FATAL"})
			c.Event(deployment.ServiceEvent{Time: at(4), Service: "web", Action: deployment.ServiceActionDie})
			<-ctx.Done()
			return nil
		}, nil)

		resp, err := http.Get(a.url + "?environment=staging")
		assert.Nil(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

		reader := bufio.NewReader(resp.Body)
		body := readUntil(t, reader, "event: state")
		body += readUntil(t, reader, "\n\n")
		assert.Match(t, `id: 2026-11-14T23:41:03.118Z\nevent: logs\ndata: \[\{"time":"2026-11-14T23:41:02.118Z","service":"web","container":"","stream":"stdout","text":"GET / 200"\},\{[^]]*"stream":"stderr","text":"FATAL"\}\]`, body)
		assert.Match(t, `event: state\ndata: \{"time":"2026-11-14T23:41:04.118Z","service":"web","container":"","action":"die","exit_code":null,"health":null\}`, body)

		queries := a.queries()
		assert.HasLength(t, 1, queries)
		assert.Equal(t, "app1", queries[0].AppID)
		assert.Equal(t, "staging", queries[0].Environment)
		assert.Equal(t, logsDefaultTail, queries[0].Tail)
		assert.False(t, queries[0].Since.HasValue())
	})

	t.Run("should resume from Last-Event-ID and bound the tail", func(t *testing.T) {
		a := arrange(t, live.NewLimiter(3, 20), func(ctx context.Context, c deployment.LogsConsumer) error {
			return deployment.ErrNoContainers
		}, nil)

		req, _ := http.NewRequest(http.MethodGet, a.url+"?tail=99999", nil)
		req.Header.Set("Last-Event-ID", "2026-11-14T23:41:03.118Z")
		resp, err := http.DefaultClient.Do(req)
		assert.Nil(t, err)
		resp.Body.Close()

		queries := a.queries()
		assert.Equal(t, "production", queries[0].Environment)
		assert.Equal(t, logsMaxTail, queries[0].Tail)
		assert.Equal(t, at(3), queries[0].Since.Get(time.Time{}))
	})

	t.Run("should answer with a JSON error when the stream can't start", func(t *testing.T) {
		for _, tt := range []struct {
			err    error
			status int
			code   string
		}{
			{deployment.ErrNoContainers, http.StatusNotFound, "no_containers"},
			{deployment.ErrLogsUnavailable, http.StatusConflict, "logs_unavailable"},
		} {
			a := arrange(t, live.NewLimiter(3, 20), func(context.Context, deployment.LogsConsumer) error { return tt.err }, nil)

			resp, err := http.Get(a.url)
			assert.Nil(t, err)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Match(t, `"code":"`+tt.code+`"`, string(body))
		}
	})

	t.Run("should end a started stream with an error event", func(t *testing.T) {
		a := arrange(t, live.NewLimiter(3, 20), func(_ context.Context, c deployment.LogsConsumer) error {
			c.Ready()
			c.Line(deployment.LogLine{Time: at(2), Text: "last line"})
			return deployment.ErrLogsUnavailable
		}, nil)

		resp, err := http.Get(a.url)
		assert.Nil(t, err)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		assert.Match(t, `"text":"last line"`, string(body))
		assert.Match(t, "event: error\ndata: \\{\"code\":\"logs_unavailable\"\\}", string(body))
	})

	t.Run("should refuse a stream past the limit", func(t *testing.T) {
		a := arrange(t, live.NewLimiter(1, 20), func(ctx context.Context, c deployment.LogsConsumer) error {
			c.Ready()
			<-ctx.Done()
			return nil
		}, nil)

		first, err := http.Get(a.url)
		assert.Nil(t, err)
		defer first.Body.Close()
		assert.Equal(t, http.StatusOK, first.StatusCode)

		second, err := http.Get(a.url)
		assert.Nil(t, err)
		second.Body.Close()
		assert.Equal(t, http.StatusTooManyRequests, second.StatusCode)
	})

	t.Run("should end open streams when the server shuts down", func(t *testing.T) {
		a := arrange(t, live.NewLimiter(3, 20), func(ctx context.Context, c deployment.LogsConsumer) error {
			c.Ready()
			<-ctx.Done()
			return nil
		}, nil)

		resp, err := http.Get(a.url)
		assert.Nil(t, err)
		defer resp.Body.Close()
		reader := bufio.NewReader(resp.Body)
		readUntil(t, reader, "retry:")

		ended := make(chan struct{})
		go func() {
			_, _ = io.ReadAll(reader)
			close(ended)
		}()

		a.shutdown()

		select {
		case <-ended:
		case <-time.After(5 * time.Second):
			t.Fatal("the stream did not end on shutdown")
		}
	})
}

// Reads the body until it contains the given text, failing after a few seconds.
func readUntil(t *testing.T, reader *bufio.Reader, text string) string {
	t.Helper()

	var (
		read strings.Builder
		done = make(chan struct{})
	)

	go func() {
		defer close(done)
		for !strings.Contains(read.String(), text) {
			b, err := reader.ReadByte()
			if err != nil {
				return
			}
			read.WriteByte(b)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%q never came", text)
	}

	return read.String()
}
