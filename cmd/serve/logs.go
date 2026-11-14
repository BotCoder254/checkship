package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_logs"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_app_services"
	deployment "github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	httputils "github.com/checkship-deploy/checkship/pkg/http"
	"github.com/checkship-deploy/checkship/pkg/live"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/gin-gonic/gin"
)

const (
	logsDefaultTail    = 200
	logsMaxTail        = 5000
	logsBatchInterval  = 100 * time.Millisecond
	logsBatchSize      = 500
	logsStreamsPerUser = 3 // Each stream holds connections to the target, so they are few
	logsStreamsTotal   = 20
	logsBuffer         = 1024
	logsFailedCode     = "logs_failed"
)

type (
	logsParams struct {
		Environment string `form:"environment"`
		Tail        int    `form:"tail"`
		Since       string `form:"since"`
	}

	// What the stream goroutine hands to the handler.
	logsItem struct {
		ready bool
		line  *deployment.LogLine
		event *deployment.ServiceEvent
	}

	// Passes the provider's calls to the handler through a channel. A send gives up when
	// the handler is gone.
	logsConsumer struct {
		ctx   context.Context
		items chan logsItem
	}
)

func (c *logsConsumer) send(item logsItem) {
	select {
	case c.items <- item:
	case <-c.ctx.Done():
	}
}

func (c *logsConsumer) Ready()                          { c.send(logsItem{ready: true}) }
func (c *logsConsumer) Line(l deployment.LogLine)       { c.send(logsItem{line: &l}) }
func (c *logsConsumer) Event(e deployment.ServiceEvent) { c.send(logsItem{event: &e}) }

func environmentOrProduction(value string) string {
	if value == "" {
		return string(deployment.Production)
	}

	return value
}

// Where to resume a stream: the Last-Event-ID header the browser sends on reconnect, or
// the since parameter.
func logsSince(ctx *gin.Context, params logsParams) monad.Maybe[time.Time] {
	var since monad.Maybe[time.Time]

	for _, value := range []string{ctx.GetHeader("Last-Event-ID"), params.Since} {
		if at, err := time.Parse(time.RFC3339Nano, value); err == nil {
			since.Set(at)
			break
		}
	}

	return since
}

func logsTail(value int) int {
	if value <= 0 {
		return logsDefaultTail
	}

	return min(value, logsMaxTail)
}

// Server-sent events with the logs of an application environment's containers. Lines come
// in batches (event "logs", id = the last line's time, so a reconnect resumes from there),
// container changes as "state", and a stream ending on its own error as "error".
// Errors found before the containers are read are plain JSON responses.
func (s *server) appLogsHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var params logsParams
		_ = ctx.ShouldBindQuery(&params)

		uid, _ := domain.CurrentUser(ctx.Request.Context()).TryGet()
		release, err := s.logsLimiter.Acquire(string(uid))

		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": err.Error()})
			return
		}

		defer release()

		stream, err := bus.Send(s.bus, ctx.Request.Context(), get_app_logs.Query{
			AppID:       ctx.Param("id"),
			Environment: environmentOrProduction(params.Environment),
			Tail:        logsTail(params.Tail),
			Since:       logsSince(ctx, params),
		})

		if err != nil {
			s.logsError(ctx, err)
			return
		}

		// Ends with the request, or when the server shuts down (Shutdown waits for open
		// requests and this one would never end otherwise).
		streamCtx, cancel := context.WithCancel(ctx.Request.Context())
		defer cancel()
		stopOnShutdown := context.AfterFunc(s.shutdownContext(), cancel)
		defer stopOnShutdown()

		consumer := &logsConsumer{ctx: streamCtx, items: make(chan logsItem, logsBuffer)}
		result := make(chan error, 1)

		go func() { result <- stream(streamCtx, consumer) }()

		s.writeLogs(ctx, streamCtx, consumer, result)
	}
}

func (s *server) writeLogs(ctx *gin.Context, streamCtx context.Context, consumer *logsConsumer, result <-chan error) {
	var (
		w         = ctx.Writer
		started   bool
		batch     []deployment.LogLine
		heartbeat = time.NewTicker(liveHeartbeat)
		flushing  = time.NewTicker(logsBatchInterval)
		flush     func()
	)

	defer heartbeat.Stop()
	defer flushing.Stop()

	// Lines and events sent before the stream returned are still in the channel.
	drain := func() {
		for {
			select {
			case item := <-consumer.items:
				if item.line != nil {
					batch = append(batch, *item.line)
				} else if item.event != nil {
					flush()
					data, _ := json.Marshal(item.event)
					fmt.Fprintf(w, "event: state\ndata: %s\n\n", data)
				}
			default:
				return
			}
		}
	}

	flush = func() {
		if len(batch) == 0 {
			return
		}

		data, _ := json.Marshal(batch)
		id := batch[len(batch)-1].Time.UTC().Format(time.RFC3339Nano)
		fmt.Fprintf(w, "id: %s\nevent: logs\ndata: %s\n\n", id, data)
		w.Flush()
		batch = batch[:0]
	}

	for {
		select {
		case <-streamCtx.Done():
			return
		case err := <-result:
			if !started {
				if err == nil {
					err = deployment.ErrNoContainers
				}
				s.logsError(ctx, err)
				return
			}

			drain()
			flush()

			if err != nil {
				data, _ := json.Marshal(gin.H{"code": s.logsErrorCode(err)})
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
				w.Flush()
			}

			return
		case item := <-consumer.items:
			switch {
			case item.ready:
				started = true
				header := w.Header()
				header.Set("Content-Type", "text/event-stream")
				header.Set("Cache-Control", "no-store")
				header.Set("Connection", "keep-alive")
				header.Set("X-Accel-Buffering", "no") // proxies such as nginx must not buffer the stream
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "retry: %d\n\n", liveRetry.Milliseconds())
				w.Flush()
			case item.line != nil:
				batch = append(batch, *item.line)

				if len(batch) >= logsBatchSize {
					flush()
				}
			case item.event != nil:
				flush()
				data, _ := json.Marshal(item.event)
				fmt.Fprintf(w, "event: state\ndata: %s\n\n", data)
				w.Flush()
			}
		case <-flushing.C:
			if started {
				flush()
			}
		case <-heartbeat.C:
			if started {
				fmt.Fprint(w, ": ping\n\n")
				w.Flush()
			}
		}
	}
}

// JSON error for a stream that couldn't start.
func (s *server) logsError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, deployment.ErrNoContainers):
		ctx.AbortWithStatusJSON(http.StatusNotFound, err)
	case errors.Is(err, deployment.ErrLogsUnavailable):
		ctx.AbortWithStatusJSON(http.StatusConflict, err)
	case errors.Is(err, live.ErrTooManyStreams):
		ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": err.Error()})
	default:
		httputils.HandleError(s, ctx, err)
	}
}

// Code sent in the "error" event of a stream that had started.
func (s *server) logsErrorCode(err error) string {
	if appErr, ok := apperr.As[apperr.Error](err); ok {
		return appErr.Code
	}

	s.logger.Errorw("runtime logs stream failed", "error", err)

	return logsFailedCode
}

func (s *server) appServicesHandler() gin.HandlerFunc {
	return httputils.Send(s, func(ctx *gin.Context) error {
		states, err := bus.Send(s.bus, ctx.Request.Context(), get_app_services.Query{
			AppID:       ctx.Param("id"),
			Environment: environmentOrProduction(ctx.Query("environment")),
		})

		if err != nil {
			return err
		}

		return httputils.Ok(ctx, states)
	})
}

func (s *server) shutdownContext() context.Context {
	if s.shutdown == nil {
		return context.Background()
	}

	return s.shutdown
}
