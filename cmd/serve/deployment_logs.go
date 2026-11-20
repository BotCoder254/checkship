package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment"
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_deployment_log"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/bus"
	httputils "github.com/checkship-deploy/checkship/pkg/http"
	"github.com/gin-gonic/gin"
)

const (
	deploymentLogsStreamsPerUser = 5  // A stream only reads a local file, they are cheap
	deploymentLogsStreamsTotal   = 40 //
	deploymentLogsChunkSize      = 64 << 10
	deploymentLogsMaxPerWake     = 1 << 20 // Read at most this much before checking the request again
	deploymentLogsPoll           = time.Second
)

type deploymentLogsChunk struct {
	Text string `json:"text"`
}

// Parses the deployment number of the route, anything invalid being a not found.
func deploymentNumber(ctx *gin.Context) (int, error) {
	number, err := strconv.Atoi(ctx.Param("number"))

	if err != nil || number <= 0 {
		return 0, apperr.ErrNotFound
	}

	return number, nil
}

// Where to resume a deployment log stream: the Last-Event-ID header the browser sends on
// reconnect, or the offset parameter. Both are a byte offset in the log file.
func deploymentLogsOffset(ctx *gin.Context) int64 {
	for _, value := range []string{ctx.GetHeader("Last-Event-ID"), ctx.Query("offset")} {
		if offset, err := strconv.ParseInt(value, 10, 64); err == nil && offset >= 0 {
			return offset
		}
	}

	return 0
}

// Server-sent events with a deployment log as it is written (seelf#7 companion: logs used to
// be polled whole every 2.5s). Events:
//
//   - "logs": {"text": "..."} with the new whole lines, id = the byte offset reached, so a
//     reconnect resumes from there
//   - "reset": the requested offset is past the end of the file, the client must clear what
//     it shows; what follows starts at 0
//   - "end": the deployment has finished and everything was sent; the stream closes
//
// The log path comes from the deployment, never from the request.
func (s *server) deploymentLogsStreamHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		appid := ctx.Param("id")
		number, err := deploymentNumber(ctx)

		if err != nil {
			httputils.HandleError(s, ctx, err)
			return
		}

		uid, _ := domain.CurrentUser(ctx.Request.Context()).TryGet()
		release, err := s.deploymentLogsLimiter.Acquire(string(uid))

		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": err.Error()})
			return
		}

		defer release()

		logpath, err := bus.Send(s.bus, ctx.Request.Context(), get_deployment_log.Query{
			AppID:            appid,
			DeploymentNumber: number,
		})

		if err != nil {
			httputils.HandleError(s, ctx, err)
			return
		}

		streamCtx, cancel := context.WithCancel(ctx.Request.Context())
		defer cancel()
		stopOnShutdown := context.AfterFunc(s.shutdownContext(), cancel)
		defer stopOnShutdown()

		finished := func() (bool, error) {
			depl, err := bus.Send(s.bus, streamCtx, get_deployment.Query{
				AppID:            appid,
				DeploymentNumber: number,
			})

			return depl.State.FinishedAt.HasValue(), err
		}

		w := ctx.Writer
		header := w.Header()
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-store")
		header.Set("Connection", "keep-alive")
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Accel-Buffering", "no") // proxies such as nginx must not buffer the stream
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "retry: %d\n\n", liveRetry.Milliseconds())
		w.Flush()

		tail := &logTail{path: logpath, offset: deploymentLogsOffset(ctx)}
		poll := time.NewTicker(deploymentLogsPoll)
		heartbeat := time.NewTicker(liveHeartbeat)
		defer poll.Stop()
		defer heartbeat.Stop()

		send := func(event string, id string, data any) {
			payload, _ := json.Marshal(data)

			if id != "" {
				fmt.Fprintf(w, "id: %s\n", id)
			}

			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
			w.Flush()
		}

		// Sends what's new; returns false when the stream must stop.
		forward := func(all bool) (sent bool, ok bool) {
			for read := 0; read < deploymentLogsMaxPerWake; {
				chunk, reset, err := tail.next(all)

				if err != nil {
					s.logger.Errorw("could not read a deployment log", "error", err)
					send("error", "", gin.H{"code": logsFailedCode})
					return sent, false
				}

				if reset {
					send("reset", "0", struct{}{})
				}

				if len(chunk) == 0 {
					return sent, true
				}

				send("logs", strconv.FormatInt(tail.offset, 10), deploymentLogsChunk{Text: string(chunk)})
				sent = true
				read += len(chunk)
			}

			return sent, true
		}

		for {
			// Registered before reading so a write happening meanwhile still wakes us up.
			written, stopWaiting := s.logSignals.Wait(logpath)

			sent, ok := forward(false)

			if !ok {
				stopWaiting()
				return
			}

			// Nothing new: if the deployment has ended, its log is complete (the log is closed
			// before the final state is written), send what's left and stop.
			if !sent {
				done, err := finished()

				if err != nil {
					stopWaiting()

					if !errors.Is(err, context.Canceled) {
						send("error", "", gin.H{"code": logsFailedCode})
					}

					return
				}

				if done {
					stopWaiting()

					if _, ok = forward(true); ok {
						send("end", "", struct{}{})
					}

					return
				}
			}

			select {
			case <-streamCtx.Done():
				stopWaiting()
				return
			case <-written:
			case <-poll.C:
			case <-heartbeat.C:
				fmt.Fprint(w, ": ping\n\n")
				w.Flush()
			}

			stopWaiting()
		}
	}
}

// Reads a log file from an offset.
type logTail struct {
	path   string
	offset int64
}

// Returns the bytes after the offset, up to deploymentLogsChunkSize, and moves the offset past
// them. Unless all is true, only whole lines are returned (but a line longer than a chunk is
// cut). A missing file is an empty log (a deployment which hasn't started). reset is true when
// the offset was past the end of the file and was moved back to 0.
func (t *logTail) next(all bool) (chunk []byte, reset bool, err error) {
	file, err := os.Open(t.path)

	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	defer file.Close()

	info, err := file.Stat()

	if err != nil {
		return nil, false, err
	}

	if t.offset > info.Size() {
		t.offset = 0
		reset = true
	}

	available := info.Size() - t.offset

	if available <= 0 {
		return nil, reset, nil
	}

	buf := make([]byte, min(available, deploymentLogsChunkSize))
	n, err := file.ReadAt(buf, t.offset)

	if err != nil && !errors.Is(err, io.EOF) {
		return nil, reset, err
	}

	buf = buf[:n]

	if !all {
		if idx := bytes.LastIndexByte(buf, '\n'); idx >= 0 {
			buf = buf[:idx+1]
		} else if len(buf) < deploymentLogsChunkSize {
			return nil, reset, nil // An incomplete line, wait for the rest of it
		}
	}

	t.offset += int64(len(buf))

	return buf, reset, nil
}
