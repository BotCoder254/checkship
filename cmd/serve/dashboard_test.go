package serve

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_dashboard"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/live"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func Test_DashboardParams(t *testing.T) {
	t.Run("should fall back on unknown values", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/?range=365&environment=prod%27--&tz=99999", nil)

		f := dashboardFilters(ctx)

		assert.Equal(t, get_dashboard.DefaultDays, f.Days)
		assert.False(t, f.Environment.HasValue())
		assert.Equal(t, get_dashboard.MaxOffset, f.Offset)
	})

	t.Run("should keep allowed values", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/?range=7&environment=staging&tz=-300", nil)

		f := dashboardFilters(ctx)

		assert.Equal(t, 7, f.Days)
		assert.Equal(t, "staging", f.Environment.MustGet())
		assert.Equal(t, -300, f.Offset)
	})
}

func Test_DashboardIDParam(t *testing.T) {
	assert.Equal(t, "3MDVPFzEqkIJs51zinbSnHpG3LL", idParam("3MDVPFzEqkIJs51zinbSnHpG3LL").MustGet())
	assert.False(t, idParam("").HasValue())
	assert.False(t, idParam("' OR 1=1 --").HasValue())
	assert.False(t, idParam("short").HasValue())
}

// The dashboard routes sit behind the session-only group: the CI API key is refused.
func Test_DashboardRequiresASession(t *testing.T) {
	s := &server{router: gin.New(), live: live.NewHub(5, 100, 4)}
	s.router.Use(sessions.Sessions(sessionName, cookie.NewStore([]byte("secret"))))
	secured := s.router.Group("/api/v1", s.authenticate(false))
	secured.GET("/dashboard/summary", func(c *gin.Context) { c.Status(http.StatusOK) })
	secured.GET("/events", s.eventsHandler())

	for _, path := range []string{"/api/v1/dashboard/summary", "/api/v1/events"} {
		for _, auth := range []string{"", "Bearer some-api-key"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			if auth != "" {
				req.Header.Set("Authorization", auth)
			}
			rec := httptest.NewRecorder()

			s.router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s with %q", path, auth)
		}
	}
}

func Test_Events(t *testing.T) {
	arrange := func(hub *live.Hub) *httptest.Server {
		s := &server{live: hub}
		router := gin.New()
		router.GET("/events", func(c *gin.Context) {
			c.Request = c.Request.WithContext(domain.WithUserID(c.Request.Context(), "u1"))
		}, s.eventsHandler())
		srv := httptest.NewServer(router)
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("should stream grouped hints with ids", func(t *testing.T) {
		hub := live.NewHub(5, 100, 8)
		srv := arrange(hub)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
		res, err := http.DefaultClient.Do(req)
		assert.Nil(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "text/event-stream", res.Header.Get("Content-Type"))
		assert.Equal(t, "no-store", res.Header.Get("Cache-Control"))

		reader := bufio.NewReader(res.Body)
		line, _ := reader.ReadString('\n')
		assert.Equal(t, "retry: 5000\n", line)

		for hub.Count() == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		hub.Publish(live.Hint{Topic: "deployments", ID: "app"})
		hub.Publish(live.Hint{Topic: "deployments", ID: "app"}) // same hint: sent once

		var got []string
		for len(got) < 3 {
			line, err := reader.ReadString('\n')
			assert.Nil(t, err)
			if line = strings.TrimSpace(line); line != "" {
				got = append(got, line)
			}
		}
		assert.DeepEqual(t, []string{"id: 1", "event: hint", `data: {"topic":"deployments","id":"app"}`}, got)
	})

	t.Run("should refuse streams over the limit and end them on shutdown", func(t *testing.T) {
		hub := live.NewHub(1, 100, 8)
		srv := arrange(hub)

		first, err := http.Get(srv.URL + "/events")
		assert.Nil(t, err)
		defer first.Body.Close()
		for hub.Count() == 0 {
			time.Sleep(10 * time.Millisecond)
		}

		second, err := http.Get(srv.URL + "/events")
		assert.Nil(t, err)
		second.Body.Close()
		assert.Equal(t, http.StatusTooManyRequests, second.StatusCode)

		hub.Close()

		done := make(chan struct{})
		go func() {
			reader := bufio.NewReader(first.Body)
			for {
				if _, err := reader.ReadString('\n'); err != nil {
					close(done)
					return
				}
			}
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("the stream should end when the hub closes")
		}
	})
}
