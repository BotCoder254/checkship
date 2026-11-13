package flag_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/flag"
)

type flagType uint

const (
	flagA flagType = 1 << iota
	flagB
	flagC
)

func Test_IsSet(t *testing.T) {
	assert.True(t, flag.IsSet(flagA, flagA))
	assert.False(t, flag.IsSet(flagA, flagB))
	assert.True(t, flag.IsSet(flagA|flagB, flagA))
	assert.True(t, flag.IsSet(flagA|flagB, flagB|flagA))
	assert.True(t, flag.IsSet(flagA|flagB|flagC, flagB|flagA))
	assert.False(t, flag.IsSet(flagA, flagB|flagA))
	assert.False(t, flag.IsSet(flagA|flagC, flagB|flagA))
}
