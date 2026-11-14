package domain

import (
	"context"
	"time"

	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

var (
	ErrNoContainers    = apperr.New("no_containers")    // Nothing runs for this application environment (never deployed or cleaned up)
	ErrLogsUnavailable = apperr.New("logs_unavailable") // The logging driver of the containers can't be read
)

const (
	LogStreamStdout LogStream = "stdout"
	LogStreamStderr LogStream = "stderr"

	ServiceActionStart   ServiceAction = "start"
	ServiceActionDie     ServiceAction = "die"
	ServiceActionOOM     ServiceAction = "oom"
	ServiceActionHealth  ServiceAction = "health"
	ServiceActionDestroy ServiceAction = "destroy"
)

type (
	LogStream     string
	ServiceAction string

	// Options of a runtime logs stream.
	LogsOptions struct {
		Tail  int                    // Lines to read back from each container before following
		Since monad.Maybe[time.Time] // Only lines written from this time, used to resume a stream
	}

	// A line written by an application container.
	LogLine struct {
		Time      time.Time `json:"time"`
		Service   string    `json:"service"`
		Container string    `json:"container"`
		Stream    LogStream `json:"stream"`
		Text      string    `json:"text"`
		Truncated bool      `json:"truncated,omitempty"`
	}

	// Something that happened to a container while its logs were followed.
	ServiceEvent struct {
		Time      time.Time           `json:"time"`
		Service   string              `json:"service"`
		Container string              `json:"container"`
		Action    ServiceAction       `json:"action"`
		ExitCode  monad.Maybe[int]    `json:"exit_code"`
		Health    monad.Maybe[string] `json:"health"`
	}

	// Receives a runtime logs stream. Ready is called once, when the containers have been
	// found and their logs are being read; lines and events follow it. Calls are made
	// from several goroutines and must be safe for concurrent use.
	LogsConsumer interface {
		Ready()
		Line(LogLine)
		Event(ServiceEvent)
	}

	// Runs a runtime logs stream until ctx is done or an error occurs.
	LogsStream func(context.Context, LogsConsumer) error

	// State of an application container at a given time.
	ServiceState struct {
		Service      string                 `json:"service"`
		Container    string                 `json:"container"`
		Image        string                 `json:"image"`
		State        string                 `json:"state"`  // created, running, paused, restarting, removing, exited or dead
		Status       string                 `json:"status"` // Docker's summary, such as "Up 3 minutes"
		Health       monad.Maybe[string]    `json:"health"`
		RestartCount int                    `json:"restart_count"`
		ExitCode     monad.Maybe[int]       `json:"exit_code"`
		StartedAt    monad.Maybe[time.Time] `json:"started_at"`
	}
)
