package numbers

import (
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

var ErrMin = apperr.New("min")

func Min(minValue int) validate.Validator[int] {
	return func(value int) error {
		if value < minValue {
			return ErrMin
		}

		return nil
	}
}
