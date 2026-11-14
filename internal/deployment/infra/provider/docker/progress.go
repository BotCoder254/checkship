package docker

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Compose prints one line per pull event when its output is not a terminal, which is always
// the case for deployment logs, so a single layer download can write thousands of identical
// lines like:
//
//	03e7aa2637ca Downloading [=======>                                   ]  16.78MB/115.5MB
//
// and timers such as "03e7aa2637ca Extracting 2 s" many times per second. BuildKit, which
// builds images, prints base image layer downloads every few seconds:
//
//	#4 sha256:f9abc654947e09e316e7703544bd7e68123d383aeb70c5a9cf15be03e71721ce 4.19MB / 12.67MB 16.7s
//
// progressFilter sits between compose and the deployment log. It keeps a progress line only
// when its layer reaches another tenth of its size and drops a line identical to the last one
// printed for the same layer or service. Every other line goes through as is.
type progressFilter struct {
	mu       sync.Mutex
	out      io.Writer
	pending  []byte            // Last incomplete line, written once its newline arrives
	printed  map[string]int    // Last tenth printed per "<layer> <action>"
	lastLine map[string]string // Last line printed per layer or service
}

var (
	progressBarLine = regexp.MustCompile(`^\s*(\S+) (\S+) \[([=> ]+)\]`)
	composeLine     = regexp.MustCompile(`^ (\S+) \S`) // Compose prefixes its lines with a space and the layer or service
	buildkitLayer   = regexp.MustCompile(`^#\d+ (sha256:\S+) ([\d.]+)\s?([kKMGT]?i?B) / ([\d.]+)\s?([kKMGT]?i?B) [\d.]+s\s*$`)
)

var sizeUnits = map[string]float64{
	"B": 1, "kB": 1e3, "KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12,
	"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40,
}

func newProgressFilter(out io.Writer) *progressFilter {
	return &progressFilter{
		out:      out,
		printed:  make(map[string]int),
		lastLine: make(map[string]string),
	}
}

func (f *progressFilter) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pending = append(f.pending, p...)

	var kept []byte

	for {
		idx := bytes.IndexByte(f.pending, '\n')

		if idx < 0 {
			break
		}

		line := f.pending[:idx+1]

		if f.keep(string(line)) {
			kept = append(kept, line...)
		}

		f.pending = f.pending[idx+1:]
	}

	// Don't keep growing the backing array when everything has been consumed
	if len(f.pending) == 0 {
		f.pending = nil
	}

	if len(kept) > 0 {
		if _, err := f.out.Write(kept); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

// Writes what's left of an incomplete last line, if any.
func (f *progressFilter) Flush() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.pending) == 0 {
		return nil
	}

	_, err := f.out.Write(append(f.pending, '\n'))
	f.pending = nil

	return err
}

func (f *progressFilter) keep(line string) bool {
	if subject := composeLine.FindStringSubmatch(line); subject != nil {
		if f.lastLine[subject[1]] == line {
			return false
		}

		f.lastLine[subject[1]] = line
	}

	if match := progressBarLine.FindStringSubmatch(line); match != nil {
		bar := match[3]
		return f.reached(match[1]+" "+match[2], strings.Count(bar, "=")*10/len(bar))
	}

	// Lines ending with "done" don't match: the final size of a layer is always kept.
	if match := buildkitLayer.FindStringSubmatch(line); match != nil {
		current, total := size(match[2], match[3]), size(match[4], match[5])

		if total <= 0 {
			return true
		}

		return f.reached(match[1], min(int(current*10/total), 10))
	}

	return true
}

// Tells whether the given layer progress reached another tenth since the last printed line.
func (f *progressFilter) reached(key string, tenth int) bool {
	if last, found := f.printed[key]; found && last == tenth {
		return false
	}

	f.printed[key] = tenth

	return true
}

func size(value, unit string) float64 {
	n, err := strconv.ParseFloat(value, 64)

	if err != nil {
		return 0
	}

	return n * sizeUnits[unit]
}
