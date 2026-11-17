package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_dashboard"
	"github.com/checkship-deploy/checkship/pkg/bus"
	httputils "github.com/checkship-deploy/checkship/pkg/http"
	"github.com/checkship-deploy/checkship/pkg/live"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/gin-gonic/gin"
)

// Dashboard routes: read only, session only (never the CI API key). Every parameter is read as
// text and mapped to an allowed value, so a bad value falls back instead of failing.

const (
	liveRetry     = 5 * time.Second
	liveHeartbeat = 15 * time.Second
	liveBatch     = 500 * time.Millisecond
)

func dashboardFilters(ctx *gin.Context) get_dashboard.Filters {
	f := get_dashboard.Filters{
		Days:        atoi(ctx.Query("range")),
		Environment: environmentParam(ctx.Query("environment")),
		Offset:      atoi(ctx.Query("tz")),
		Until:       time.Now().UTC(),
	}

	return f.Normalized()
}

func environmentParam(value string) (m monad.Maybe[string]) {
	if value == "production" || value == "staging" {
		m.Set(value)
	}
	return m
}

// An application or target id (a ksuid: 27 letters and digits); anything else means all of them.
func idParam(value string) (m monad.Maybe[string]) {
	value = strings.TrimSpace(value)
	if len(value) == 27 && strings.IndexFunc(value, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) == -1 {
		m.Set(value)
	}
	return m
}

func atoi(value string) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return n
}

// The data is the user's: never let a shared cache keep it.
func privateOk[T any](ctx *gin.Context, data T) error {
	ctx.Header("Cache-Control", "private, no-store")
	ctx.JSON(http.StatusOK, data)
	return nil
}

func (s *server) dashboardSummaryHandler() gin.HandlerFunc {
	return httputils.Send(s, func(ctx *gin.Context) error {
		summary, err := bus.Send(s.bus, ctx.Request.Context(), get_dashboard.SummaryQuery{
			Filters: dashboardFilters(ctx),
			App:     idParam(ctx.Query("app")),
			Target:  idParam(ctx.Query("target")),
		})

		if err != nil {
			return err
		}

		return privateOk(ctx, summary)
	})
}

func (s *server) dashboardAppsHandler() gin.HandlerFunc {
	return httputils.Send(s, func(ctx *gin.Context) error {
		sort := ctx.Query("sort")
		if sort != "failures" && sort != "recent" {
			sort = "deploys"
		}

		apps, err := bus.Send(s.bus, ctx.Request.Context(), get_dashboard.AppsQuery{
			Filters: dashboardFilters(ctx),
			Sort:    sort,
			Page:    max(1, atoi(ctx.Query("page"))),
			PerPage: max(0, min(atoi(ctx.Query("per_page")), get_dashboard.MaxPerPage)),
		})

		if err != nil {
			return err
		}

		return privateOk(ctx, apps)
	})
}

func (s *server) dashboardActivityHandler() gin.HandlerFunc {
	return httputils.Send(s, func(ctx *gin.Context) error {
		query := get_dashboard.ActivityQuery{
			Environment: environmentParam(ctx.Query("environment")),
			Limit:       max(0, min(atoi(ctx.Query("limit")), get_dashboard.MaxActivity)),
		}

		if status, err := strconv.ParseUint(ctx.Query("status"), 10, 8); err == nil && status <= 3 {
			query.Status.Set(uint8(status))
		}

		// With a range (7, 30 or 90 days), only that period, as on the summary.
		if ctx.Query("range") != "" {
			_, from, _ := dashboardFilters(ctx).Bounds()
			query.Since.Set(from)
		}

		if app := strings.TrimSpace(ctx.Query("app")); app != "" && len(app) <= 64 {
			query.AppID.Set(app)
		}

		if value := ctx.Query("cursor"); value != "" {
			cursor, err := get_dashboard.DecodeCursor(value)
			if err != nil {
				return err
			}
			query.After.Set(cursor)
		}

		activity, err := bus.Send(s.bus, ctx.Request.Context(), query)

		if err != nil {
			return err
		}

		return privateOk(ctx, activity)
	})
}

// Server-sent events: hints that something changed, grouped every half second, a heartbeat
// comment every 15 s. The stream ends when the client leaves or the server shuts down.
func (s *server) eventsHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uid, _ := domain.CurrentUser(ctx.Request.Context()).TryGet()

		hints, unsubscribe, err := s.live.Subscribe(string(uid))

		if err != nil {
			status := http.StatusServiceUnavailable
			if errors.Is(err, live.ErrTooManyStreams) {
				status = http.StatusTooManyRequests
			}
			ctx.AbortWithStatusJSON(status, gin.H{"code": err.Error()})
			return
		}

		defer unsubscribe()

		w := ctx.Writer
		header := w.Header()
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-store")
		header.Set("Connection", "keep-alive")
		header.Set("X-Accel-Buffering", "no") // proxies such as nginx must not buffer the stream
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "retry: %d\n\n", liveRetry.Milliseconds())
		w.Flush()

		heartbeat := time.NewTicker(liveHeartbeat)
		batch := time.NewTicker(liveBatch)
		defer heartbeat.Stop()
		defer batch.Stop()

		var (
			id      uint64
			pending = map[live.Hint]struct{}{}
			done    = ctx.Request.Context().Done()
		)

		for {
			select {
			case <-done:
				return
			case hint, open := <-hints:
				if !open {
					return
				}
				pending[hint] = struct{}{}
			case <-batch.C:
				if len(pending) == 0 {
					continue
				}
				for hint := range pending {
					data, _ := json.Marshal(hint)
					id++
					fmt.Fprintf(w, "id: %d\nevent: hint\ndata: %s\n\n", id, data)
				}
				clear(pending)
				w.Flush()
			case <-heartbeat.C:
				fmt.Fprint(w, ": ping\n\n")
				w.Flush()
			}
		}
	}
}
