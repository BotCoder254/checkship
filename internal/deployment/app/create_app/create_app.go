package create_app

import (
	"context"

	auth "github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/bus"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/validate"
	"github.com/checkship-deploy/checkship/pkg/validate/strings"
)

type (
	// Create a new application.
	Command struct {
		bus.Command[string]

		Name           string                      `json:"name"`
		VersionControl monad.Maybe[VersionControl] `json:"version_control"`
		Production     EnvironmentConfig           `json:"production"`
		Staging        EnvironmentConfig           `json:"staging"`
	}

	EnvironmentConfig struct {
		Target  string                                    `json:"target"`
		Vars    monad.Maybe[map[string]map[string]string] `json:"vars"`
		Domains monad.Maybe[map[string][]string]          `json:"domains"` // Custom domains per service (seelf#5)
	}

	VersionControl struct {
		Url   string              `json:"url"`
		Token monad.Maybe[string] `json:"token"`
	}
)

func (Command) Name_() string { return "deployment.command.create_app" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
	rules DomainsRules,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var (
			appname          domain.AppName
			url              domain.Url
			productionTarget = domain.TargetID(cmd.Production.Target)
			stagingTarget    = domain.TargetID(cmd.Staging.Target)
		)

		if err := validate.Struct(validate.Of{
			"name": validate.Value(cmd.Name, &appname, domain.AppNameFrom),
			"version_control": validate.Maybe(cmd.VersionControl, func(config VersionControl) error {
				return validate.Struct(validate.Of{
					"url":   validate.Value(config.Url, &url, domain.UrlFrom),
					"token": validate.Maybe(config.Token, strings.Required),
				})
			}),
			"production": validate.Struct(validate.Of{
				"target": validate.Field(cmd.Production.Target, strings.Required),
			}),
			"staging": validate.Struct(validate.Of{
				"target": validate.Field(cmd.Staging.Target, strings.Required),
			}),
		}); err != nil {
			return "", err
		}

		domains, err := BuildDomains(ctx, rules, "",
			DomainsInput{Environment: domain.Production, Target: productionTarget, Raw: cmd.Production.Domains.Get(nil)},
			DomainsInput{Environment: domain.Staging, Target: stagingTarget, Raw: cmd.Staging.Domains.Get(nil)},
		)

		if err != nil {
			return "", err
		}

		productionConfig := BuildEnvironmentConfig(productionTarget, cmd.Production.Vars)
		productionConfig.HasDomains(domains[domain.Production])
		stagingConfig := BuildEnvironmentConfig(stagingTarget, cmd.Staging.Vars)
		stagingConfig.HasDomains(domains[domain.Staging])

		productionRequirement, stagingRequirement, err := reader.CheckAppNamingAvailability(
			ctx,
			appname,
			productionConfig,
			stagingConfig,
		)

		if err != nil {
			return "", err
		}

		// Returns early if the application name is not unique on both targets.
		if err = validate.Struct(validate.Of{
			"production.target": productionRequirement.Error(),
			"staging.target":    stagingRequirement.Error(),
		}); err != nil {
			return "", err
		}

		app, err := domain.NewApp(
			appname,
			productionRequirement,
			stagingRequirement,
			auth.CurrentUser(ctx).MustGet(),
		)

		if err != nil {
			return "", err
		}

		if cmdVCS, isSet := cmd.VersionControl.TryGet(); isSet {
			vcs := domain.NewVersionControl(url)

			if token, isSet := cmdVCS.Token.TryGet(); isSet {
				vcs.Authenticated(token)
			}

			_ = app.UseVersionControl(vcs)
		}

		if err := writer.Write(ctx, &app); err != nil {
			return "", err
		}

		return string(app.ID()), nil
	}
}

// Custom domains as typed, from parsed ones (to check the domains an environment keeps).
func RawDomains(domains monad.Maybe[domain.ServicesDomains]) map[string][]string {
	parsed, isSet := domains.TryGet()

	if !isSet {
		return nil
	}

	raw := make(map[string][]string, len(parsed))

	for service, hostnames := range parsed {
		for _, hostname := range hostnames {
			raw[service] = append(raw[service], string(hostname))
		}
	}

	return raw
}

// Helper method to build a domain.EnvironmentConfig from a raw command value.
func BuildEnvironmentConfig(target domain.TargetID, env monad.Maybe[map[string]map[string]string]) domain.EnvironmentConfig {
	config := domain.NewEnvironmentConfig(target)

	if vars, hasVars := env.TryGet(); hasVars {
		config.HasEnvironmentVariables(domain.ServicesEnvFrom(vars))
	}

	return config
}
