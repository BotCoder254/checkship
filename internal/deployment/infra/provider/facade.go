package provider

import (
	"context"
	"io"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/volumearchive"
)

type (
	Provider interface {
		domain.Provider
		CanPrepare(any) bool
		CanHandle(domain.ProviderConfig) bool
	}

	facade struct {
		providers []Provider
	}
)

// Creates a new facade which will call the appropriate provider.
func NewFacade(providers ...Provider) domain.Provider {
	return &facade{providers}
}

func (f *facade) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	for _, p := range f.providers {
		if p.CanPrepare(payload) {
			return p.Prepare(ctx, payload, existing...)
		}
	}

	return nil, domain.ErrNoValidProviderFound
}

func (f *facade) Deploy(ctx context.Context, info domain.DeploymentContext, deployment domain.Deployment, target domain.Target, registries []domain.Registry) (domain.Services, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.Deploy(ctx, info, deployment, target, registries)
}

func (f *facade) Setup(ctx context.Context, target domain.Target) (domain.TargetEntrypointsAssigned, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.Setup(ctx, target)
}

func (f *facade) RemoveConfiguration(ctx context.Context, target domain.Target) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.RemoveConfiguration(ctx, target)
}

func (f *facade) CleanupTarget(ctx context.Context, target domain.Target, strategy domain.CleanupStrategy) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.CleanupTarget(ctx, target, strategy)
}

func (f *facade) Cleanup(ctx context.Context, app domain.AppID, target domain.Target, env domain.Environment, strategy domain.CleanupStrategy) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.Cleanup(ctx, app, target, env, strategy)
}

func (f *facade) Logs(ctx context.Context, target domain.Target, app domain.AppID, env domain.Environment, options domain.LogsOptions, consumer domain.LogsConsumer) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.Logs(ctx, target, app, env, options, consumer)
}

func (f *facade) Services(ctx context.Context, target domain.Target, app domain.AppID, env domain.Environment) ([]domain.ServiceState, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.Services(ctx, target, app, env)
}

func (f *facade) Volumes(ctx context.Context, target domain.Target, app domain.AppID, env domain.Environment) ([]domain.VolumeInfo, []volumearchive.SkippedMount, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, nil, err
	}

	return provider.Volumes(ctx, target, app, env)
}

func (f *facade) BackupVolumes(ctx context.Context, target domain.Target, app domain.AppID, env domain.Environment, options domain.BackupVolumesOptions, w io.Writer) (domain.BackupVolumesResult, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return domain.BackupVolumesResult{}, err
	}

	return provider.BackupVolumes(ctx, target, app, env, options, w)
}

func (f *facade) RestoreVolumes(ctx context.Context, target domain.Target, app domain.AppID, env domain.Environment, options domain.RestoreVolumesOptions, r io.Reader) ([]string, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.RestoreVolumes(ctx, target, app, env, options, r)
}

func (f *facade) SetupStorage(ctx context.Context, target domain.Target, spec domain.BuiltinStorageSpec) ([]domain.StorageEndpoint, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.SetupStorage(ctx, target, spec)
}

func (f *facade) RemoveStorage(ctx context.Context, target domain.Target) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.RemoveStorage(ctx, target)
}

func (f *facade) providerForTarget(target domain.Target) (Provider, error) {
	config := target.Provider()

	for _, p := range f.providers {
		if p.CanHandle(config) {
			return p, nil
		}
	}

	return nil, domain.ErrNoValidProviderFound
}
