package docker

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
)

// Turning a Docker logs stream (requested with timestamps) into lines.
//
// Docker stores a log message per line, and splits lines longer than 16 KiB into
// pieces. With timestamps on, every piece starts with its own timestamp, so the
// timestamp ends up in the middle of long lines (docker/cli#4941). The logs endpoint
// writes one message per frame for containers without a TTY (an 8-byte header gives
// the stream and the size), and everything as a raw stream for TTY containers.

const (
	maxLogLineSize   = 64 * 1024   // Longer lines are cut and marked as truncated
	dockerPieceSize  = 16 * 1024   // Size of the pieces Docker splits long lines into
	maxLogFrameSize  = 1024 * 1024 // A frame bigger than this is read but not kept past this size
	maxTimestampSize = len("2006-01-02T15:04:05.999999999Z07:00")
	frameHeaderSize  = 8
)

var errLogStreamFailed = errors.New("docker reported an error while reading the logs")

type (
	// Receives a complete line.
	lineFunc func(stream domain.LogStream, at time.Time, text string, truncated bool)

	// Builds the lines of one stream (stdout or stderr) from Docker log messages.
	lineAssembler struct {
		stream    domain.LogStream
		emit      lineFunc
		buf       []byte
		at        time.Time
		open      bool // A message without a trailing \n left a line unfinished
		truncated bool
	}
)

func newLineAssembler(stream domain.LogStream, emit lineFunc) *lineAssembler {
	return &lineAssembler{stream: stream, emit: emit}
}

// Handles one Docker log message: "<timestamp> <text>", ending with \n unless it
// is a piece of a longer line.
func (a *lineAssembler) message(msg []byte) {
	at, rest, hasTime := splitTimestamp(msg)

	if !hasTime {
		rest = msg
		at = time.Now().UTC()
	}

	// A piece continuing an open line keeps the line's first timestamp.
	if !a.open {
		a.at = at
	}

	for {
		idx := bytes.IndexByte(rest, '\n')

		if idx < 0 {
			break
		}

		a.append(rest[:idx])
		a.flushLine()
		a.at = at
		rest = rest[idx+1:]
	}

	if len(rest) > 0 {
		a.append(rest)
		a.open = true
	}
}

// Emits an unfinished line, used when the stream ends.
func (a *lineAssembler) close() {
	if a.open {
		a.flushLine()
	}
}

func (a *lineAssembler) append(b []byte) {
	if a.truncated {
		return
	}

	if room := maxLogLineSize - len(a.buf); len(b) > room {
		b = b[:room]
		a.truncated = true
	}

	a.buf = append(a.buf, b...)
}

func (a *lineAssembler) flushLine() {
	a.emit(a.stream, a.at, cleanLine(a.buf), a.truncated)
	a.buf = a.buf[:0]
	a.open = false
	a.truncated = false
}

// Removes the \r of TTY line endings. Text before a carriage return in the middle of
// a line has been overwritten on a terminal (progress bars), so only the last part is kept.
func cleanLine(b []byte) string {
	b = bytes.TrimRight(b, "\r")

	if idx := bytes.LastIndexByte(b, '\r'); idx >= 0 {
		b = b[idx+1:]
	}

	return string(b)
}

// Splits "<RFC3339Nano timestamp> <rest>". Returns false if the message doesn't start
// with a timestamp.
func splitTimestamp(msg []byte) (time.Time, []byte, bool) {
	limit := min(len(msg), maxTimestampSize+1)
	idx := bytes.IndexByte(msg[:limit], ' ')

	if idx <= 0 {
		return time.Time{}, msg, false
	}

	at, err := time.Parse(time.RFC3339Nano, string(msg[:idx]))

	if err != nil {
		return time.Time{}, msg, false
	}

	return at, msg[idx+1:], true
}

// Reads a multiplexed stream (containers without a TTY) until it ends.
func readMultiplexedLogs(r io.Reader, emit lineFunc) error {
	var (
		header = make([]byte, frameHeaderSize)
		stdout = newLineAssembler(domain.LogStreamStdout, emit)
		stderr = newLineAssembler(domain.LogStreamStderr, emit)
	)

	defer stdout.close()
	defer stderr.close()

	for {
		if _, err := io.ReadFull(r, header); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}

			return err
		}

		size := int(binary.BigEndian.Uint32(header[4:]))
		payload, err := readFrame(r, size)

		if err != nil {
			return err
		}

		switch header[0] {
		case 1:
			stdout.message(payload)
		case 2:
			stderr.message(payload)
		case 3:
			return fmt.Errorf("%w: %s", errLogStreamFailed, bytes.TrimSpace(payload))
		}
	}
}

// Reads a frame payload, keeping at most maxLogFrameSize bytes of it.
func readFrame(r io.Reader, size int) ([]byte, error) {
	keep := min(size, maxLogFrameSize)
	payload := make([]byte, keep)

	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	if size > keep {
		if _, err := io.CopyN(io.Discard, r, int64(size-keep)); err != nil {
			return nil, err
		}
	}

	return payload, nil
}

// Reads a raw stream (TTY containers) until it ends. Messages are rebuilt from the
// stream: a timestamp, then text up to a \n or up to Docker's piece size.
func readRawLogs(r io.Reader, emit lineFunc) error {
	var (
		reader = bufio.NewReaderSize(r, dockerPieceSize+maxTimestampSize+2)
		lines  = newLineAssembler(domain.LogStreamStdout, emit)
	)

	defer lines.close()

	for {
		msg, err := readRawMessage(reader)

		if len(msg) > 0 {
			lines.message(msg)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}
	}
}

func readRawMessage(r *bufio.Reader) ([]byte, error) {
	var msg []byte

	// The timestamp and its space
	if head, err := r.Peek(maxTimestampSize + 1); err == nil || len(head) > 0 {
		if _, _, ok := splitTimestamp(head); ok {
			idx := bytes.IndexByte(head, ' ')
			msg = append(msg, head[:idx+1]...)
			_, _ = r.Discard(idx + 1)
		}
	}

	// The text, up to a \n or a full piece
	for content := 0; content < dockerPieceSize; {
		b, err := r.ReadByte()

		if err != nil {
			return msg, err
		}

		msg = append(msg, b)
		content++

		if b == '\n' {
			break
		}
	}

	return msg, nil
}
