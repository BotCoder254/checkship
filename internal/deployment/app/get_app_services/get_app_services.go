package get_app_services

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
)

// State of the containers of an application environment.
type Query struct {
	bus.Query[[]domain.ServiceState]

	AppID       string `json:"-"`
	Environment string `json:"-"`
}

func (Query) Name_() string { return "deployment.query.get_app_services" }

func Handler(
	appsReader domain.AppsReader,
	targetsReader domain.TargetsReader,
	provider domain.Provider,
) bus.RequestHandler[[]domain.ServiceState, Query] {
	return func(ctx context.Context, query Query) ([]domain.ServiceState, error) {
		env, err := domain.EnvironmentFrom(query.Environment)

		if err != nil {
			return nil, err
		}

		app, err := appsReader.GetByID(ctx, domain.AppID(query.AppID))

		if err != nil {
			return nil, err
		}

		target, err := targetsReader.GetByID(ctx, app.Config(env).Target())

		if err != nil {
			return nil, err
		}

		return provider.Services(ctx, target, app.ID(), env)
	}
}
