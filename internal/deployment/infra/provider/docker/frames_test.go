package docker

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

type collectedLine struct {
	stream    domain.LogStream
	at        string
	text      string
	truncated bool
}

func collect(lines *[]collectedLine) lineFunc {
	return func(s domain.LogStream, at time.Time, text string, truncated bool) {
		*lines = append(*lines, collectedLine{s, at.Format("15:04:05.000"), text, truncated})
	}
}

// Builds a multiplexed frame: stream byte, 3 zero bytes, big-endian size, payload.
func frame(stream byte, payload string) []byte {
	header := make([]byte, 8)
	header[0] = stream
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, payload...)
}

func Test_Frames(t *testing.T) {
	const (
		t1 = "2026-11-14T23:41:02.118000000Z"
		t2 = "2026-11-14T23:41:03.377000000Z"
	)

	t.Run("should read stdout and stderr frames into lines with their time", func(t *testing.T) {
		var (
			lines  []collectedLine
			stream bytes.Buffer
		)

		stream.Write(frame(1, t1+" GET / 200\n"))
		stream.Write(frame(2, t2+" Error: connect ETIMEDOUT\n"))

		assert.Nil(t, readMultiplexedLogs(&stream, collect(&lines)))
		assert.DeepEqual(t, []collectedLine{
			{domain.LogStreamStdout, "23:41:02.118", "GET / 200", false},
			{domain.LogStreamStderr, "23:41:03.377", "Error: connect ETIMEDOUT", false},
		}, lines)
	})

	t.Run("should join the pieces of a long line and drop the timestamps inside it", func(t *testing.T) {
		var (
			lines  []collectedLine
			stream bytes.Buffer
			first  = strings.Repeat("a", dockerPieceSize)
			second = strings.Repeat("b", 100)
		)

		stream.Write(frame(1, t1+" "+first))
		stream.Write(frame(1, t2+" "+second+"\n"))

		assert.Nil(t, readMultiplexedLogs(&stream, collect(&lines)))
		assert.HasLength(t, 1, lines)
		assert.Equal(t, first+second, lines[0].text)
		assert.Equal(t, "23:41:02.118", lines[0].at, "the line keeps its first timestamp")
	})

	t.Run("should cut lines longer than the limit and mark them", func(t *testing.T) {
		var (
			lines  []collectedLine
			stream bytes.Buffer
			piece  = strings.Repeat("x", dockerPieceSize)
		)

		for range 5 {
			stream.Write(frame(1, t1+" "+piece))
		}
		stream.Write(frame(1, t1+" end\n"))
		stream.Write(frame(1, t2+" next\n"))

		assert.Nil(t, readMultiplexedLogs(&stream, collect(&lines)))
		assert.HasLength(t, 2, lines)
		assert.Equal(t, maxLogLineSize, len(lines[0].text))
		assert.True(t, lines[0].truncated)
		assert.Equal(t, "next", lines[1].text)
		assert.False(t, lines[1].truncated)
	})

	t.Run("should keep what a carriage return left on screen and drop CRLF endings", func(t *testing.T) {
		var (
			lines  []collectedLine
			stream bytes.Buffer
		)

		stream.Write(frame(1, t1+" downloading 10%\rdownloading 100%\r\n"))

		assert.Nil(t, readMultiplexedLogs(&stream, collect(&lines)))
		assert.Equal(t, "downloading 100%", lines[0].text)
	})

	t.Run("should emit an unfinished last line when the stream ends", func(t *testing.T) {
		var (
			lines  []collectedLine
			stream bytes.Buffer
		)

		stream.Write(frame(1, t1+" no newline"))

		assert.Nil(t, readMultiplexedLogs(&stream, collect(&lines)))
		assert.DeepEqual(t, []collectedLine{{domain.LogStreamStdout, "23:41:02.118", "no newline", false}}, lines)
	})

	t.Run("should return the error frame sent by Docker", func(t *testing.T) {
		var (
			lines  []collectedLine
			stream bytes.Buffer
		)

		stream.Write(frame(3, "configured logging driver does not support reading\n"))

		err := readMultiplexedLogs(&stream, collect(&lines))
		assert.ErrorIs(t, errLogStreamFailed, err)
		assert.Match(t, "does not support reading", err.Error())
	})

	t.Run("should read a raw TTY stream with a timestamp on each line", func(t *testing.T) {
		var lines []collectedLine

		stream := strings.NewReader(t1 + " \x1b[32mready\x1b[0m\r\n" + t2 + " listening on :3000\r\n")

		assert.Nil(t, readRawLogs(stream, collect(&lines)))
		assert.DeepEqual(t, []collectedLine{
			{domain.LogStreamStdout, "23:41:02.118", "\x1b[32mready\x1b[0m", false},
			{domain.LogStreamStdout, "23:41:03.377", "listening on :3000", false},
		}, lines)
	})

	t.Run("should join the pieces of a long line in a raw stream", func(t *testing.T) {
		var (
			lines  []collectedLine
			first  = strings.Repeat("a", dockerPieceSize)
			stream = strings.NewReader(t1 + " " + first + t2 + " tail\n")
		)

		assert.Nil(t, readRawLogs(stream, collect(&lines)))
		assert.HasLength(t, 1, lines)
		assert.Equal(t, first+"tail", lines[0].text)
	})

	t.Run("should keep text that doesn't start with a timestamp", func(t *testing.T) {
		var lines []collectedLine

		assert.Nil(t, readRawLogs(strings.NewReader("plain line\n"), collect(&lines)))
		assert.Equal(t, "plain line", lines[0].text)
	})
}
