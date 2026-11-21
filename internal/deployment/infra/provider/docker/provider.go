package docker

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/app/expose_checkship_container"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/provider"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/ssh"
	ptypes "github.com/checkship-deploy/checkship/pkg/types"
	"github.com/checkship-deploy/checkship/pkg/validate"
	"github.com/checkship-deploy/checkship/pkg/validate/numbers"
	vstrings "github.com/checkship-deploy/checkship/pkg/validate/strings"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/progress"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/errdefs"
)

var (
	ErrLoadProjectFailed     = errors.New("compose_file_malformed")
	ErrOpenComposeFileFailed = errors.New("compose_file_open_failed")
	ErrComposeFailed         = errors.New("compose_failed")
	ErrTargetConnectFailed   = errors.New("target_connect_failed")

	sshConfigPath = filepath.Join(must.Panic(os.UserHomeDir()), ".ssh", "config")
)

const (
	AppLabel               = "app.checkship.application"        // ID of the application
	EnvironmentLabel       = "app.checkship.environment"        // Environment of the application
	TargetLabel            = "app.checkship.target"             // ID of the target
	ExposedLabel           = "app.checkship.exposed"            // Force the exposure of a service (used when exposing checkship itself for example)
	SubdomainLabel         = "app.checkship.subdomain"          // Subdomain to use for the service, only for http entrypoints
	CustomEntrypointsLabel = "app.checkship.custom_entrypoints" // Boolean representing wether or not a service use custom entrypoints
	StorageLabel           = "app.checkship.storage"            // Volume of the built-in backup storage, never removed by a target cleanup
)

type (
	DockerOptions func(*docker)

	Docker interface {
		provider.Provider
		expose_checkship_container.LocalProvider
		domain.TunnelConnector
	}

	docker struct {
		client    *client // Client to use, mostly for testing
		logger    log.Logger
		sshConfig ssh.Configurator
	}
)

// Creates a docker provider with given options. The configuration is mostly used to
// ease the testing of some internals.
//
// Multiple goroutine can use the same provider at the same time.
func New(logger log.Logger, configuration ...DockerOptions) Docker {
	d := &docker{
		logger:    logger,
		sshConfig: ssh.NewFileConfigurator(sshConfigPath),
	}

	for _, opt := range configuration {
		opt(d)
	}

	return d
}

// Use the given compose service and cli instead of creating new ones. Used for testing.
func WithTestConfig(cli command.Cli, composeService api.Service, sshConfigPath string) DockerOptions {
	return func(d *docker) {
		d.client = &client{
			cli:     cli,
			api:     cli.Client(),
			compose: composeService,
		}
		d.sshConfig = ssh.NewFileConfigurator(sshConfigPath)
	}
}

func (d *docker) CanPrepare(payload any) bool                 { return ptypes.Is[Body](payload) }
func (d *docker) CanHandle(config domain.ProviderConfig) bool { return ptypes.Is[Data](config) }

func (d *docker) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	config, ok := payload.(Body)

	if !ok {
		return nil, domain.ErrInvalidProviderPayload
	}

	var (
		host    ssh.Host
		privKey ssh.PrivateKey
	)

	if err := validate.Struct(validate.Of{
		"docker.host": validate.Maybe(config.Host, func(s string) error {
			return validate.Value(s, &host, ssh.ParseHost)
		}),
		"docker.user": validate.Maybe(config.User, vstrings.Required),
		"docker.port": validate.Maybe(config.Port, numbers.Min(0)),
		"docker.private_key": validate.Patch(config.PrivateKey, func(s string) error {
			return validate.Value(s, &privKey, ssh.ParsePrivateKey)
		}),
	}); err != nil {
		return nil, err
	}

	var data Data

	// No host, we're done
	if !config.Host.HasValue() {
		return data, nil
	}

	data.Host.Set(host)
	data.User.Set(config.User.Get(defaultUser))
	data.Port.Set(config.Port.Get(defaultPort))

	// Private key nil, we're done
	if config.PrivateKey.IsNil() {
		return data, nil
	}

	// Private key set, we're done
	if config.PrivateKey.HasValue() {
		data.PrivateKey.Set(privKey)
		return data, nil
	}

	// Else try to retrieve it from the existing one.
	// (This is needed because the existing private key is never exposed to the end user)
	for _, existingConfig := range existing {
		d, isDockerData := existingConfig.(Data)

		if !isDockerData {
			continue
		}

		data.PrivateKey = d.PrivateKey
	}

	return data, nil
}

func (d *docker) Setup(ctx context.Context, target domain.Target) (domain.TargetEntrypointsAssigned, error) {
	config, ok := target.Provider().(Data)

	if !ok {
		return nil, domain.ErrInvalidProviderPayload
	}

	if err := d.configureTargetSSH(target.ID(), config); err != nil {
		return nil, err
	}

	client, err := d.tryConnect(ctx, nil, config.Host)

	if err != nil {
		return nil, err
	}

	defer client.Close()

	if target.IsManual() {
		return nil, client.compose.Down(ctx, targetProjectName(target.ID()), api.DownOptions{
			RemoveOrphans: true,
			Images:        "all",
			Volumes:       true,
		})
	}

	project, assigned, err := newProxyProjectBuilder(client, target).Build(ctx)

	if err != nil {
		return nil, err
	}

	return assigned, client.compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			QuietPull:     true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	})
}

func (d *docker) RemoveConfiguration(_ context.Context, target domain.Target) error {
	return d.sshConfig.Remove(string(target.ID()))
}

func (d *docker) PrepareLocal(context.Context) (domain.ProviderConfig, error) {
	return Data{}, nil
}

func (d *docker) Expose(ctx context.Context, target domain.Target, container string) error {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	err = client.api.NetworkConnect(ctx, targetPublicNetworkName(target.ID()), container, nil)

	// Catch the forbidden error when the container is already attached
	if errdefs.IsForbidden(err) {
		return nil
	}

	if err != nil {
		return err
	}

	// Without the restart, traefik may pick the wrong container ip... This will force checkship to restart right now
	return client.api.ContainerRestart(ctx, container, dockercontainer.StopOptions{})
}

func (d *docker) Deploy(
	ctx context.Context,
	deploymentCtx domain.DeploymentContext,
	deployment domain.Deployment,
	target domain.Target,
	registries []domain.Registry,
) (domain.Services, error) {
	logger := deploymentCtx.Logger()
	composeOutput := newProgressFilter(logger)
	defer composeOutput.Flush()

	client, err := d.connect(ctx, composeOutput, target, registries...)

	if err != nil {
		logger.Error(err)
		return nil, ErrTargetConnectFailed
	}

	defer client.Close()

	// A cancelled deployment (seelf#7) also drops the idle connections to the Docker daemon
	// right away; requests in flight end with the context.
	stopClosing := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stopClosing()

	logger.Stepf("successfully connected to docker version %s", client.version)

	if len(client.registries) > 0 {
		logger.Infof("using custom registries: %s", strings.Join(client.registries, ", "))
	}

	project, services, err := newDeploymentProjectBuilder(deploymentCtx, deployment, target).Build(ctx)

	if err != nil {
		return nil, err
	}

	logger.Stepf("launching docker compose project (pulling, building and running)")

	if built := servicesToBuild(project); len(built) > 0 {
		logger.Infof("image(s) to build for %s", strings.Join(built, ", "))
	}

	if err = client.compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			// Plain BuildKit progress, written to the deployment log by the patched compose
			// (third_party/compose) like the rest of the compose output.
			Build: &api.BuildOptions{
				Progress: progress.ModePlain,
			},
			RemoveOrphans: true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	}); err != nil {
		// Stopped on purpose: the deploy job says so in the log, "context canceled" would only
		// look like a failure.
		if ctx.Err() == nil {
			logger.Error(err)
		}

		return nil, ErrComposeFailed
	}

	if url, isManagedByCheckship := target.Url().TryGet(); isManagedByCheckship {
		if url.UseSSL() {
			logger.Infof("you may have to wait for certificates to be generated before your app is available")
		}

		if len(services.CustomEntrypoints()) > 0 {
			logger.Infof("this deployment uses custom entrypoints. If this is the first time, you may have to wait a few seconds for the target to find available ports and expose them appropriately")
		}
	}

	prunedCount, err := client.PruneImages(ctx, filters.NewArgs(
		filters.Arg("dangling", "true"),
		filters.Arg("label", AppLabel+"="+string(deployment.ID().AppID())),
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
		filters.Arg("label", EnvironmentLabel+"="+string(deployment.Config().Environment())),
	))

	if err != nil {
		logger.Warnf(err.Error())
	} else if prunedCount > 0 {
		logger.Infof("pruned %d dangling image(s)", prunedCount)
	}

	return services, nil
}

func (d *docker) CleanupTarget(ctx context.Context, target domain.Target, strategy domain.CleanupStrategy) (err error) {
	if strategy == domain.CleanupStrategySkip {
		return nil
	}

	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	// TODO: We should probably prune all images and docker builder cache to free up some space
	return client.RemoveResources(ctx, filters.NewArgs(
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
	))
}

func (d *docker) Cleanup(ctx context.Context, app domain.AppID, target domain.Target, env domain.Environment, strategy domain.CleanupStrategy) error {
	if strategy == domain.CleanupStrategySkip {
		return nil
	}

	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	return client.RemoveResources(ctx, filters.NewArgs(
		filters.Arg("label", AppLabel+"="+string(app)),
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
		filters.Arg("label", EnvironmentLabel+"="+string(env)),
	))
}

func (d *docker) tryConnect(ctx context.Context, out io.Writer, host monad.Maybe[ssh.Host], registries ...domain.Registry) (*client, error) {
	// For tests, bypass the initialization and use the provided one
	if d.client != nil {
		return d.client, nil
	}

	return connect(ctx, out, host, registries...)
}

// Connect to the docker daemon and return a new docker cli and compose service.
func (d *docker) connect(ctx context.Context, out io.Writer, target domain.Target, registries ...domain.Registry) (*client, error) {
	data, ok := target.Provider().(Data)

	if !ok {
		return nil, domain.ErrInvalidProviderPayload
	}

	return d.tryConnect(ctx, out, data.Host, registries...)
}

// Names of the services which have an image to build, in alphabetical order.
func servicesToBuild(project *types.Project) []string {
	var names []string

	for _, name := range project.ServiceNames() {
		if project.Services[name].Build != nil {
			names = append(names, name)
		}
	}

	return names
}

func (d *docker) configureTargetSSH(id domain.TargetID, config Data) error {
	host, isRemote := config.Host.TryGet()

	if !isRemote {
		return nil
	}

	var key monad.Maybe[ssh.ConnectionKey]

	if privKey, hasKey := config.PrivateKey.TryGet(); hasKey {
		key.Set(ssh.ConnectionKey{
			Name: string(id),
			Key:  privKey,
		})
	}

	return d.sshConfig.Upsert(ssh.Connection{
		Identifier: string(id),
		Host:       host,
		User:       config.User,
		Port:       config.Port,
		PrivateKey: key,
	})
}

// add some labels to a given target.
func appendLabels(target types.Labels, labelsToAdd types.Labels) types.Labels {
	if target == nil {
		target = types.Labels{}
	}

	for k, v := range labelsToAdd {
		target[k] = v
	}

	return target
}

// Apply common compose labels as per https://github.com/docker/compose/blob/126cb988c6f0c00a2a9887b8a39dc0907daec289/cmd/compose/compose.go#L200
func getProjectCustomLabels(project, service, workingDir string, composeFiles ...string) map[string]string {
	labels := map[string]string{
		api.ProjectLabel:     project,
		api.ServiceLabel:     service,
		api.VersionLabel:     api.ComposeVersion,
		api.ConfigFilesLabel: strings.Join(composeFiles, ","),
		api.OneoffLabel:      "False",
	}

	if workingDir != "" {
		labels[api.WorkingDirLabel] = workingDir
	}

	return labels
}
