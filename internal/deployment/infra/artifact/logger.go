package artifact

import (
	"fmt"
	"io"
	"sync"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
)

type stepLogger struct {
	mu      sync.Mutex // Compose events and steps are written from several goroutines
	writer  io.WriteCloser
	written func() // Called after each write, so live readers of the log wake up
}

// Instantiates a new step logger to provide a simple way to build a deployment logfile.
func newLogger(writer io.WriteCloser, written func()) domain.DeploymentLogger {
	if written == nil {
		written = func() {}
	}

	return &stepLogger{writer: writer, written: written}
}

func (l *stepLogger) Stepf(format string, args ...any) {
	l.print("[STEP]", format, args)
}

func (l *stepLogger) Warnf(format string, args ...any) {
	l.print("[WARN]", format, args)
}

func (l *stepLogger) Infof(format string, args ...any) {
	l.print("[INFO]", format, args)
}

func (l *stepLogger) Error(err error) {
	l.print("[ERROR]", err.Error(), nil)
}

func (l *stepLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	n, err = l.writer.Write(p)
	l.mu.Unlock()

	l.written()

	return n, err
}

func (l *stepLogger) Close() error {
	l.mu.Lock()
	err := l.writer.Close()
	l.mu.Unlock()

	l.written()

	return err
}

func (l *stepLogger) print(prefix string, format string, args []any) {
	_, _ = l.Write([]byte(prefix + " " + fmt.Sprintf(format, args...) + "\n"))
}
