package numbers

import (
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

var (
	ErrMin = apperr.New("min")
	ErrMax = apperr.New("max")
)

func Min(minValue int) validate.Validator[int] {
	return func(value int) error {
		if value < minValue {
			return ErrMin
		}

		return nil
	}
}

func Max(maxValue int) validate.Validator[int] {
	return func(value int) error {
		if value > maxValue {
			return ErrMax
		}

		return nil
	}
}
