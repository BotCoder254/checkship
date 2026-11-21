package update_app

import (
	"context"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"

	"github.com/checkship-deploy/checkship/internal/deployment/app/create_app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/validate"
	"github.com/checkship-deploy/checkship/pkg/validate/strings"
)

type (
	// Update an existing application.
	Command struct {
		bus.Command[string]

		ID             string                         `json:"-"`
		VersionControl monad.Patch[VersionControl]    `json:"version_control"`
		Production     monad.Maybe[EnvironmentConfig] `json:"production"`
		Staging        monad.Maybe[EnvironmentConfig] `json:"staging"`
		MoveVolumes    monad.Maybe[bool]              `json:"move_volumes"` // When a target changes, copy volume data to the new one (true if not set)
	}

	EnvironmentConfig create_app.EnvironmentConfig

	VersionControl struct {
		Url   string              `json:"url"`
		Token monad.Patch[string] `json:"token"`
	}
)

func (Command) Name_() string { return "deployment.command.update_app" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
	rules create_app.DomainsRules,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var url domain.Url

		if err := validate.Struct(validate.Of{
			"version_control": validate.Patch(cmd.VersionControl, func(config VersionControl) error {
				return validate.Struct(validate.Of{
					"url":   validate.Value(config.Url, &url, domain.UrlFrom),
					"token": validate.Patch(config.Token, strings.Required),
				})
			}),
			"production": validate.Maybe(cmd.Production, func(conf EnvironmentConfig) error {
				return validate.Struct(validate.Of{
					"target": validate.Field(conf.Target, strings.Required),
				})
			}),
			"staging": validate.Maybe(cmd.Staging, func(conf EnvironmentConfig) error {
				return validate.Struct(validate.Of{
					"target": validate.Field(conf.Target, strings.Required),
				})
			}),
		}); err != nil {
			return "", err
		}

		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return "", err
		}

		// Determine the availability of updated targets
		var productionConfig, stagingConfig monad.Maybe[domain.EnvironmentConfig]

		// Custom domains: those given replace the environment's ones, omitted ones are kept.
		// Both environments are checked every time, against the target each will run on, so a
		// domain can't move twice into the application and a target change re-checks the domains.
		inputs := make([]create_app.DomainsInput, 0, 2)

		for _, item := range []struct {
			env  domain.Environment
			conf monad.Maybe[EnvironmentConfig]
		}{{domain.Production, cmd.Production}, {domain.Staging, cmd.Staging}} {
			current := app.Config(item.env)
			input := create_app.DomainsInput{
				Environment: item.env,
				Target:      current.Target(),
				Raw:         create_app.RawDomains(current.Domains()),
			}

			if conf, isUpdated := item.conf.TryGet(); isUpdated {
				input.Target = domain.TargetID(conf.Target)

				if raw, isSet := conf.Domains.TryGet(); isSet {
					input.Raw = raw
				}
			}

			inputs = append(inputs, input)
		}

		domains, err := create_app.BuildDomains(ctx, rules, app.ID(), inputs...)

		if err != nil {
			return "", err
		}

		if conf, isUpdated := cmd.Production.TryGet(); isUpdated {
			config := create_app.BuildEnvironmentConfig(domain.TargetID(conf.Target), conf.Vars)
			config.HasDomains(domains[domain.Production])
			productionConfig.Set(config)
		}

		if conf, isUpdated := cmd.Staging.TryGet(); isUpdated {
			config := create_app.BuildEnvironmentConfig(domain.TargetID(conf.Target), conf.Vars)
			config.HasDomains(domains[domain.Staging])
			stagingConfig.Set(config)
		}

		productionRequirement, stagingRequirement, err := reader.CheckAppNamingAvailabilityByID(ctx, app.ID(), productionConfig, stagingConfig)

		if err != nil {
			return "", err
		}

		if err = validate.Struct(validate.Of{
			"production.target": validate.If(productionConfig.HasValue(), productionRequirement.Error),
			"staging.target":    validate.If(stagingConfig.HasValue(), stagingRequirement.Error),
		}); err != nil {
			return "", err
		}

		if vcsPatch, isSet := cmd.VersionControl.TryGet(); isSet {
			if vcsUpdate, hasValue := vcsPatch.TryGet(); hasValue {
				// Take the existing vcs as a reference so the token is not modified if not provided at all
				vcs := app.VersionControl().Get(domain.NewVersionControl(url))
				vcs.HasUrl(url)

				if tokenPatch, isSet := vcsUpdate.Token.TryGet(); isSet {
					if token, hasValue := tokenPatch.TryGet(); hasValue {
						vcs.Authenticated(token)
					} else {
						vcs.Public()
					}
				}

				err = app.UseVersionControl(vcs)
			} else {
				err = app.RemoveVersionControl()
			}

			if err != nil {
				return "", err
			}
		}

		change := domain.EnvironmentChange{
			MoveVolumes: cmd.MoveVolumes.Get(true),
			By:          auth.CurrentUser(ctx).MustGet(),
		}

		if productionConfig.HasValue() {
			if err = app.ChangeProductionConfig(productionRequirement, change); err != nil {
				return "", err
			}
		}

		if stagingConfig.HasValue() {
			if err = app.ChangeStagingConfig(stagingRequirement, change); err != nil {
				return "", err
			}
		}

		if err = writer.Write(ctx, &app); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
