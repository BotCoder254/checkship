package get_app_logs

import (
	"context"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

// Prepares a stream of the logs written by an application environment's containers.
// The stream itself runs when the caller calls it, with the context that should end it.
type Query struct {
	bus.Query[domain.LogsStream]

	AppID       string                 `json:"-"`
	Environment string                 `json:"-"`
	Tail        int                    `json:"-"`
	Since       monad.Maybe[time.Time] `json:"-"`
}

func (Query) Name_() string { return "deployment.query.get_app_logs" }

func Handler(
	appsReader domain.AppsReader,
	targetsReader domain.TargetsReader,
	provider domain.Provider,
) bus.RequestHandler[domain.LogsStream, Query] {
	return func(ctx context.Context, query Query) (domain.LogsStream, error) {
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

		options := domain.LogsOptions{Tail: query.Tail, Since: query.Since}

		return func(ctx context.Context, consumer domain.LogsConsumer) error {
			return provider.Logs(ctx, target, app.ID(), env, options, consumer)
		}, nil
	}
}
