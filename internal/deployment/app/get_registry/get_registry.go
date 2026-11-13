package get_registry

import (
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

type (
	// Retrieve one registry
	Query struct {
		bus.Query[Registry]

		ID string `json:"id"`
	}

	Registry struct {
		ID          string                   `json:"id"`
		Name        string                   `json:"name"`
		Url         string                   `json:"url"`
		Credentials monad.Maybe[Credentials] `json:"credentials"`
		CreatedAt   time.Time                `json:"created_at"`
		CreatedBy   app.UserSummary          `json:"created_by"`
	}

	Credentials struct {
		Username string               `json:"username"`
		Password storage.SecretString `json:"password"`
	}
)

func (Query) Name_() string { return "deployment.query.get_registry" }
