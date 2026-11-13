package get_targets

import (
	"github.com/checkship-deploy/checkship/internal/deployment/app/get_target"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// Retrieve all available targets.
type Query struct {
	bus.Query[[]get_target.Target]

	ActiveOnly bool `form:"active_only"`
}

func (Query) Name_() string { return "deployment.query.get_targets" }
