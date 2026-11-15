package bytesize_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/bytesize"
)

func Test_Parse(t *testing.T) {
	for value, expected := range map[string]int64{
		"512":     512,
		"10b":     10,
		"1kb":     1024,
		"32mb":    32 << 20,
		"5GB":     5 << 30,
		" 1.5 gb": 3 << 29,
		"2tb":     2 << 40,
	} {
		got, err := bytesize.Parse(value)
		assert.Nil(t, err, value)
		assert.Equal(t, expected, got, value)
	}

	for _, value := range []string{"", "mb", "-1mb", "0", "abc", "5 parsecs", "99999999999tb"} {
		_, err := bytesize.Parse(value)
		assert.ErrorIs(t, bytesize.ErrInvalidSize, err, value)
	}
}
