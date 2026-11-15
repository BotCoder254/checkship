package domain

import (
	"context"
	"io"

	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
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
		// Named volumes of an application environment on a target, and the mounts that cannot be backed up.
		Volumes(context.Context, Target, AppID, Environment) ([]VolumeInfo, []volumearchive.SkippedMount, error)
		// Write an archive of an application environment's volumes to the given writer.
		BackupVolumes(context.Context, Target, AppID, Environment, BackupVolumesOptions, io.Writer) (BackupVolumesResult, error)
		// Replace volumes of an application environment with the content of an archive.
		// Returns the keys of the volumes restored.
		RestoreVolumes(context.Context, Target, AppID, Environment, RestoreVolumesOptions, io.Reader) ([]string, error)
		// Runs the built-in backup storage on a target and returns the ways to reach it, best first.
		SetupStorage(context.Context, Target, BuiltinStorageSpec) ([]StorageEndpoint, error)
		// Stops the built-in backup storage on a target and removes its data.
		RemoveStorage(context.Context, Target) error
	}
)
