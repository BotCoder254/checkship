package http

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/gin-gonic/gin"
)

var ErrUnexpected = apperr.New("unexpected_error") // Error returned when an infrastructure error occurs

// Tiny interface to represents needed contrat in order to use helpers provided by this package.
type Server interface {
	IsSecure() bool
	Logger() log.Logger
}

// Bind the request to the given TIn and handle errors if any by setting the appropriate status.
func Bind[TIn any](s Server, handler func(*gin.Context, TIn) error) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var cmd TIn

		if err := ctx.ShouldBind(&cmd); err != nil {
			ctx.AbortWithError(http.StatusUnprocessableEntity, err)
			return
		}

		if err := handler(ctx, cmd); err != nil {
			HandleError(s, ctx, err)
		}
	}
}

// Call the given handler and handle any returned error by setting the appropriate status.
func Send(s Server, handler func(*gin.Context) error) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if err := handler(ctx); err != nil {
			HandleError(s, ctx, err)
		}
	}
}

// Sets the location header and the response status
func Created[TOut any](s Server, ctx *gin.Context, data TOut, location string, args ...any) error {
	scheme := "http://"

	if s.IsSecure() {
		scheme = "https://"
	}

	addCommonResponseHeaders(ctx)
	ctx.Header("Location", fmt.Sprintf(scheme+path.Join(ctx.Request.Host, location), args...))
	ctx.JSON(http.StatusCreated, data)
	return nil
}

// Mark the request has succeeded with no data.
func NoContent(ctx *gin.Context) error {
	addCommonResponseHeaders(ctx)
	ctx.Status(http.StatusNoContent)
	return nil
}

// Returns the text file at the given path, for files which are still being written such as
// deployment logs.
//
// The content type is always text/plain: guessing it from the extension depends on the host
// mime database (".log" is text/x-log on many Linux desktops) and clients only read text/plain
// as text. The response is never cached and has no Last-Modified header, whose one second
// precision made conditional requests answer 304 with outdated content while the file grew.
func TextFile(ctx *gin.Context, filepath string) error {
	ctx.Header("Content-Type", "text/plain; charset=utf-8")
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("X-Content-Type-Options", "nosniff")

	file, err := os.Open(filepath)

	// Not written yet (a queued deployment for example): that's an empty file, not an error.
	if errors.Is(err, os.ErrNotExist) {
		ctx.Status(http.StatusOK)
		return nil
	}

	if err != nil {
		return err
	}

	defer file.Close()

	http.ServeContent(ctx.Writer, ctx.Request, path.Base(filepath), time.Time{}, file)

	return nil
}

// Streams a file to download. Nothing is buffered, so archives of any size go
// straight from their source to the client.
func Attachment(ctx *gin.Context, filename, contentType string, size int64, content io.Reader) error {
	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("X-Content-Type-Options", "nosniff")

	if size >= 0 {
		ctx.Header("Content-Length", strconv.FormatInt(size, 10))
	}

	ctx.Status(http.StatusOK)

	_, err := io.Copy(ctx.Writer, content)

	return err
}

// Mark the request has succeeded with the given data.
func Ok[TOut any](ctx *gin.Context, data TOut) error {
	addCommonResponseHeaders(ctx)
	ctx.JSON(http.StatusOK, data)
	return nil
}

// Handle the given non-nil error and sets the status code based on error type.
func HandleError(s Server, ctx *gin.Context, err error) {
	var (
		status int = http.StatusInternalServerError
		data   any = ErrUnexpected
	)

	// Translates the error type to the appropriate HTTP status code
	if _, isAppErr := apperr.As[apperr.Error](err); isAppErr {
		status = http.StatusBadRequest // Default to HTTP 400
		data = err

		if errors.Is(err, apperr.ErrNotFound) {
			status = http.StatusNotFound // But if it's a not found, that's an HTTP 404
		}
	} else {
		s.Logger().Errorw(err.Error(), "error", err)
	}

	ctx.Error(err)
	ctx.AbortWithStatusJSON(status, data)
}

func addCommonResponseHeaders(ctx *gin.Context) {
	ctx.Header("Cache-Control", "public, max-age=0, must-revalidate")
}
