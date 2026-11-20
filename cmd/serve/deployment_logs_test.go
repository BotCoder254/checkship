package serve

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment_log"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/bus/memory"
	"github.com/checkship-deploy/checkship/pkg/live"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/gin-gonic/gin"
)

func Test_DeploymentLogsStream(t *testing.T) {
	type arranged struct {
		url      string
		logpath  string
		finished *atomic.Bool
		signals  *live.Signals
	}

	arrange := func(t *testing.T, limiter *live.Limiter) arranged {
		var (
			b        = memory.NewBus()
			logpath  = filepath.Join(t.TempDir(), "1-app-1.deployment.log")
			finished = &atomic.Bool{}
			signals  = live.NewSignals()
		)

		bus.Register(b, func(_ context.Context, q get_deployment_log.Query) (string, error) {
			if q.AppID != "app" || q.DeploymentNumber != 1 {
				return "", apperr.ErrNotFound
			}
			return logpath, nil
		})

		bus.Register(b, func(_ context.Context, q get_deployment.Query) (get_deployment.Deployment, error) {
			var d get_deployment.Deployment
			if finished.Load() {
				d.State.FinishedAt.Set(time.Now())
			}
			return d, nil
		})

		if limiter == nil {
			limiter = live.NewLimiter(deploymentLogsStreamsPerUser, deploymentLogsStreamsTotal)
		}

		shutdown, stop := context.WithCancel(context.Background())
		s := &server{
			bus:                   b,
			logger:                must.Panic(log.NewLogger()),
			deploymentLogsLimiter: limiter,
			logSignals:            signals,
			shutdown:              shutdown,
			stopStreams:           stop,
		}
		router := gin.New()
		router.GET("/apps/:id/deployments/:number/logs/stream", func(c *gin.Context) {
			c.Request = c.Request.WithContext(domain.WithUserID(c.Request.Context(), "u1"))
		}, s.deploymentLogsStreamHandler())
		srv := httptest.NewServer(router)
		t.Cleanup(srv.Close)
		t.Cleanup(stop)

		return arranged{
			url:      srv.URL + "/apps/app/deployments/1/logs/stream",
			logpath:  logpath,
			finished: finished,
			signals:  signals,
		}
	}

	write := func(t *testing.T, a arranged, text string) {
		f, err := os.OpenFile(a.logpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		assert.Nil(t, err)
		_, err = f.WriteString(text)
		assert.Nil(t, err)
		assert.Nil(t, f.Close())
		a.signals.Notify(a.logpath)
	}

	t.Run("should send the whole log then end once the deployment has finished", func(t *testing.T) {
		a := arrange(t, nil)
		write(t, a, "[STEP] one\n[STEP] two\npartial")
		a.finished.Store(true)

		events := readEvents(t, a.url, nil)

		assert.DeepEqual(t, []sseEvent{
			{event: "logs", id: "22", text: "[STEP] one\n[STEP] two\n"},
			{event: "logs", id: "29", text: "partial"},
			{event: "end"},
		}, events)
	})

	t.Run("should resume from the Last-Event-ID offset", func(t *testing.T) {
		a := arrange(t, nil)
		write(t, a, "[STEP] one\n[STEP] two\n")
		a.finished.Store(true)

		events := readEvents(t, a.url, map[string]string{"Last-Event-ID": "11"})

		assert.DeepEqual(t, []sseEvent{
			{event: "logs", id: "22", text: "[STEP] two\n"},
			{event: "end"},
		}, events)
	})

	t.Run("should reset an offset past the end of the log", func(t *testing.T) {
		a := arrange(t, nil)
		write(t, a, "[STEP] one\n")
		a.finished.Store(true)

		events := readEvents(t, a.url+"?offset=500", nil)

		assert.DeepEqual(t, []sseEvent{
			{event: "reset", id: "0"},
			{event: "logs", id: "11", text: "[STEP] one\n"},
			{event: "end"},
		}, events)
	})

	t.Run("should stream lines as they are written while the deployment runs", func(t *testing.T) {
		a := arrange(t, nil)
		write(t, a, "[STEP] one\n")

		got := make(chan []sseEvent, 1)
		go func() { got <- readEvents(t, a.url, nil) }()

		time.Sleep(150 * time.Millisecond)
		write(t, a, "[INFO] two\n[INFO] thr")
		time.Sleep(150 * time.Millisecond)
		write(t, a, "ee\n")
		time.Sleep(150 * time.Millisecond)
		a.finished.Store(true)

		select {
		case events := <-got:
			assert.DeepEqual(t, []sseEvent{
				{event: "logs", id: "11", text: "[STEP] one\n"},
				{event: "logs", id: "22", text: "[INFO] two\n"},
				{event: "logs", id: "35", text: "[INFO] three\n"},
				{event: "end"},
			}, events)
		case <-time.After(5 * time.Second):
			t.Fatal("the stream did not end")
		}
	})

	t.Run("should end at once for a deployment cancelled before it wrote anything", func(t *testing.T) {
		a := arrange(t, nil)
		a.finished.Store(true)

		assert.DeepEqual(t, []sseEvent{{event: "end"}}, readEvents(t, a.url, nil))
	})

	t.Run("should answer 404 for an unknown or invalid deployment", func(t *testing.T) {
		a := arrange(t, nil)

		for _, url := range []string{
			strings.Replace(a.url, "/1/", "/2/", 1),
			strings.Replace(a.url, "/1/", "/abc/", 1),
			strings.Replace(a.url, "/1/", "/-1/", 1),
		} {
			res, err := http.Get(url)
			assert.Nil(t, err)
			res.Body.Close()
			assert.Equal(t, http.StatusNotFound, res.StatusCode)
		}
	})

	t.Run("should refuse a stream over the limit", func(t *testing.T) {
		limiter := live.NewLimiter(1, 1)
		release, _ := limiter.Acquire("u1")
		defer release()
		a := arrange(t, limiter)

		res, err := http.Get(a.url)
		assert.Nil(t, err)
		res.Body.Close()

		assert.Equal(t, http.StatusTooManyRequests, res.StatusCode)
	})
}

type sseEvent struct {
	event, id, text string
}

// Reads a stream until it closes and returns its events (pings and retry lines left out).
func readEvents(t *testing.T, url string, headers map[string]string) []sseEvent {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("unexpected response %d %s", res.StatusCode, res.Header.Get("Content-Type"))
		return nil
	}

	var (
		events  []sseEvent
		current sseEvent
		scanner = bufio.NewScanner(res.Body)
	)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case line == "":
			if current.event != "" {
				events = append(events, current)
			}
			current = sseEvent{}
		case strings.HasPrefix(line, "event: "):
			current.event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "id: "):
			current.id = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "data: "):
			var chunk deploymentLogsChunk
			_ = json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk)
			current.text = chunk.Text
		}
	}

	return events
}
