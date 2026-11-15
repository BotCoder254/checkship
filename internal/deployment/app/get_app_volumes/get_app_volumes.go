package get_app_volumes

import (
	"context"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

type (
	// Named volumes of an application environment, read live from its target, and
	// the mounts a backup does not include.
	Query struct {
		bus.Query[Volumes]

		AppID       string `json:"-"`
		Environment string `json:"-"`
	}

	Volumes struct {
		Target  string                       `json:"target"`
		Volumes []Volume                     `json:"volumes"`
		Skipped []volumearchive.SkippedMount `json:"skipped"`
	}

	Volume struct {
		Key        string   `json:"key"`
		DockerName string   `json:"docker_name"`
		Driver     string   `json:"driver"`
		Services   []string `json:"services"`
	}
)

func (Query) Name_() string { return "deployment.query.get_app_volumes" }

func Handler(
	apps domain.AppsReader,
	targets domain.TargetsReader,
	provider domain.Provider,
) bus.RequestHandler[Volumes, Query] {
	return func(ctx context.Context, query Query) (Volumes, error) {
		env, err := domain.EnvironmentFrom(query.Environment)

		if err != nil {
			return Volumes{}, err
		}

		app, err := apps.GetByID(ctx, domain.AppID(query.AppID))

		if err != nil {
			return Volumes{}, err
		}

		target, err := targets.GetByID(ctx, app.Config(env).Target())

		if err != nil {
			return Volumes{}, err
		}

		infos, skipped, err := provider.Volumes(ctx, target, app.ID(), env)

		if err != nil {
			return Volumes{}, err
		}

		result := Volumes{
			Target:  string(target.ID()),
			Volumes: make([]Volume, len(infos)),
			Skipped: skipped,
		}

		if result.Skipped == nil {
			result.Skipped = []volumearchive.SkippedMount{}
		}

		for i, v := range infos {
			services := v.Services

			if services == nil {
				services = []string{}
			}

			result.Volumes[i] = Volume{Key: v.Key, DockerName: v.DockerName, Driver: v.Driver, Services: services}
		}

		return result, nil
	}
}
