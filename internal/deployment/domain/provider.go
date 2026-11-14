package domain

import (
	"context"

	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/storage"
)

var (
	ErrNoValidProviderFound   = apperr.New("no_valid_provider_found")
	ErrInvalidProviderPayload = apperr.New("invalid_provider_payload")

	ProviderConfigTypes = storage.NewDiscriminatedMapper(func(c ProviderConfig) string { return c.Kind() })
)

type (
	// Provider specific configuration used by a target
	ProviderConfig interface {
		Kind() string
		Equals(ProviderConfig) bool // Compare the provider configuration with another one
		Fingerprint() string        // Used for unicity of a provider configuration (such as one per host)
		String() string             // User friendly representation, mostly for logs
	}

	// Provider used to run an application services.
	Provider interface {
		// Prepare the given payload representing a Provider specific configuration.
		Prepare(ctx context.Context, payload any, existing ...ProviderConfig) (ProviderConfig, error)
		// Deploy a deployment on the specified target and return services that has been deployed.
		Deploy(context.Context, DeploymentContext, Deployment, Target, []Registry) (Services, error)
		// Setup a target by deploying the needed stuff to actually serve deployments.
		Setup(context.Context, Target) (TargetEntrypointsAssigned, error)
		// Remove target related configuration.
		RemoveConfiguration(context.Context, Target) error
		// Cleanup a target, removing every resources managed by checkship on it.
		CleanupTarget(context.Context, Target, CleanupStrategy) error
		// Cleanup an application on the specified target and environment, which means removing every possible stuff related to it
		Cleanup(context.Context, AppID, Target, Environment, CleanupStrategy) error
		// Follow the logs of an application environment's containers on a target, until ctx is done.
		// Returns ErrNoContainers when nothing has ever run there.
		Logs(context.Context, Target, AppID, Environment, LogsOptions, LogsConsumer) error
		// State of an application environment's containers on a target.
		Services(context.Context, Target, AppID, Environment) ([]ServiceState, error)
	}
)
