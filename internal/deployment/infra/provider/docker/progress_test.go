package docker

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_ProgressFilter(t *testing.T) {
	bar := func(layer, action string, filled int) string {
		return " " + layer + " " + action + " [" + strings.Repeat("=", filled) + ">" + strings.Repeat(" ", 49-filled) + "]  1MB/10MB\n"
	}

	t.Run("should let every line which is not a progress bar go through", func(t *testing.T) {
		var out strings.Builder
		f, _ := clockedFilter(&out)

		input := "[STEP] launching docker compose project\n db Pulling \n a8a2131d1a02 Pulling fs layer \n a8a2131d1a02 Pull complete \n db Pulled \n"
		n, err := f.Write([]byte(input))

		assert.Nil(t, err)
		assert.Equal(t, len(input), n)
		assert.Equal(t, input, out.String())
	})

	t.Run("should drop a line identical to the last one printed for the same layer", func(t *testing.T) {
		var out strings.Builder
		f, _ := clockedFilter(&out)

		_, _ = f.Write([]byte(" aaa Extracting 1 s\n aaa Extracting 1 s\n bbb Extracting 1 s\n aaa Extracting 1 s\n aaa Extracting 2 s\n"))

		assert.Equal(t, " aaa Extracting 1 s\n bbb Extracting 1 s\n aaa Extracting 2 s\n", out.String())
	})

	t.Run("should keep a progress bar line only when its layer reaches another tenth", func(t *testing.T) {
		var out strings.Builder
		f, _ := clockedFilter(&out)

		for _, line := range []string{
			bar("aaa", "Downloading", 1),
			bar("aaa", "Downloading", 1),
			bar("aaa", "Downloading", 3),
			bar("bbb", "Downloading", 3),
			bar("aaa", "Extracting", 3),
			bar("aaa", "Downloading", 5),
			bar("aaa", "Downloading", 6),
			bar("aaa", "Downloading", 49),
		} {
			_, err := f.Write([]byte(line))
			assert.Nil(t, err)
		}

		assert.Equal(t, bar("aaa", "Downloading", 1)+
			bar("bbb", "Downloading", 3)+
			bar("aaa", "Extracting", 3)+
			bar("aaa", "Downloading", 5)+
			bar("aaa", "Downloading", 49), out.String())
	})

	t.Run("should keep a BuildKit layer download line only when it reaches another tenth", func(t *testing.T) {
		var out strings.Builder
		f, _ := clockedFilter(&out)

		input := []string{
			"#4 [1/2] FROM docker.io/library/python:3.13-alpine\n",
			"#4 sha256:aaa 0B / 12.67MB 0.2s\n",
			"#4 sha256:aaa 0B / 12.67MB 5.3s\n",
			"#4 sha256:aaa 1.05MB / 12.67MB 7.1s\n",
			"#4 sha256:aaa 2.10MB / 12.67MB 11.1s\n",
			"#4 sha256:bbb 412.00kB / 1.2MB 3.5s\n",
			"#4 sha256:aaa 2.10MB / 12.67MB 14.3s\n",
			"#4 sha256:aaa 12.67MB / 12.67MB 44.0s\n",
			"#4 sha256:aaa 12.67MB / 12.67MB 44.2s done\n",
			"#5 0.311 hello from the build\n",
			"#5 0.311 hello from the build\n",
		}

		_, _ = f.Write([]byte(strings.Join(input, "")))

		assert.Equal(t, input[0]+input[1]+input[4]+input[5]+input[7]+input[8]+input[9]+input[10], out.String())
	})

	t.Run("should handle lines split across writes and flush the last incomplete one", func(t *testing.T) {
		var out strings.Builder
		f, _ := clockedFilter(&out)

		_, _ = f.Write([]byte(" db Pul"))
		assert.Equal(t, "", out.String())

		_, _ = f.Write([]byte("ling \n db Pulled"))
		assert.Equal(t, " db Pulling \n", out.String())

		assert.Nil(t, f.Flush())
		assert.Equal(t, " db Pulling \n db Pulled\n", out.String())
	})

	t.Run("should keep a progress bar line once a second even within the same tenth", func(t *testing.T) {
		var out strings.Builder
		f, advance := clockedFilter(&out)

		_, _ = f.Write([]byte(bar("aaa", "Downloading", 11)))
		advance(400 * time.Millisecond)
		_, _ = f.Write([]byte(bar("aaa", "Downloading", 12)))
		advance(700 * time.Millisecond)
		_, _ = f.Write([]byte(bar("aaa", "Downloading", 13)))
		_, _ = f.Write([]byte(bar("aaa", "Downloading", 13))) // identical: still dropped
		advance(1500 * time.Millisecond)
		_, _ = f.Write([]byte(bar("bbb", "Downloading", 1)))

		assert.Equal(t, bar("aaa", "Downloading", 11)+
			bar("aaa", "Downloading", 13)+
			bar("bbb", "Downloading", 1), out.String())
	})
}

// A filter whose clock only moves when told to.
func clockedFilter(out io.Writer) (*progressFilter, func(time.Duration)) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	f := newProgressFilter(out)
	f.now = func() time.Time { return now }

	return f, func(d time.Duration) { now = now.Add(d) }
}
