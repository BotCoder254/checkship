// Package bytesize parses human sizes such as "512mb" or "5 GB".
package bytesize

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

var ErrInvalidSize = errors.New("invalid_size")

var units = []struct {
	suffix     string
	multiplier int64
}{
	{"tb", 1 << 40},
	{"gb", 1 << 30},
	{"mb", 1 << 20},
	{"kb", 1 << 10},
	{"b", 1},
}

// Parses a size in bytes. Units are powers of 1024, case does not matter and a
// bare number is a count of bytes.
func Parse(value string) (int64, error) {
	s := strings.ToLower(strings.TrimSpace(value))
	multiplier := int64(1)

	for _, u := range units {
		if rest, ok := strings.CutSuffix(s, u.suffix); ok {
			s, multiplier = strings.TrimSpace(rest), u.multiplier
			break
		}
	}

	n, err := strconv.ParseFloat(s, 64)

	if err != nil || n <= 0 || math.IsInf(n, 0) || n*float64(multiplier) > math.MaxInt64 {
		return 0, ErrInvalidSize
	}

	return int64(n * float64(multiplier)), nil
}
