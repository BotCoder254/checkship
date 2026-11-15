package docker

import (
	"context"
	"strconv"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/api"
)

const (
	// versitygw, the S3 gateway checkship runs for backups (Apache-2.0). Its posix
	// backend keeps every object as a plain file, so archives stay readable even
	// without the gateway.
	BuiltinStorageImage = "versity/versitygw:v1.11.0"
	StorageAlias        = "checkship-storage" // Name on the target's network
	StorageSubdomain    = "s3"                // Route on the target's proxy: s3.<target host>
	storagePort         = 7070
	storageService      = "storage"
)

func storageProjectName(id domain.TargetID) string {
	return "checkship-internal-storage-" + strings.ToLower(string(id))
}

func (d *docker) SetupStorage(ctx context.Context, target domain.Target, spec domain.BuiltinStorageSpec) ([]domain.StorageEndpoint, error) {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return nil, err
	}

	defer client.Close()

	project, endpoints := storageProject(target, spec)

	if err = client.compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			QuietPull:     true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	}); err != nil {
		return nil, err
	}

	return endpoints, nil
}

func (d *docker) RemoveStorage(ctx context.Context, target domain.Target) error {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	return client.compose.Down(ctx, storageProjectName(target.ID()), api.DownOptions{
		RemoveOrphans: true,
		Volumes:       true,
	})
}

// Compose project of the built-in storage, and the ways to reach it, best first:
//   - its alias on the target's network, when checkship's own container is
//     exposed on that target (the usual local setup);
//   - s3.<host> through the target's proxy, when the target has a URL;
//   - 127.0.0.1, for a local target when checkship runs directly on the host.
func storageProject(target domain.Target, spec domain.BuiltinStorageSpec) (*types.Project, []domain.StorageEndpoint) {
	name := storageProjectName(target.ID())
	idLower := domain.TargetID(strings.ToLower(string(target.ID())))
	labels := types.Labels{TargetLabel: string(target.ID())}
	image := spec.Image

	if image == "" {
		image = BuiltinStorageImage
	}

	service := types.ServiceConfig{
		Name:    storageService,
		Image:   image,
		Restart: types.RestartPolicyUnlessStopped,
		Command: types.ShellCommand{"--port", ":" + strconv.Itoa(storagePort), "posix", "/data"},
		Environment: types.MappingWithEquals{
			"ROOT_ACCESS_KEY": &spec.AccessKey,
			"ROOT_SECRET_KEY": &spec.SecretKey,
		},
		Labels: labels,
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeVolume, Source: "data", Target: "/data"},
		},
		Networks: map[string]*types.ServiceNetworkConfig{
			"default": {Aliases: []string{StorageAlias}},
		},
		CustomLabels: getProjectCustomLabels(name, storageService, ""),
	}

	project := &types.Project{
		Name: name,
		Networks: types.Networks{
			"default": types.NetworkConfig{
				Name:   name + "_network",
				Labels: labels,
			},
		},
		Volumes: types.Volumes{
			"data": types.VolumeConfig{
				Name:   name + "_data",
				Labels: types.Labels{StorageLabel: string(target.ID())},
			},
		},
	}

	endpoints := []domain.StorageEndpoint{
		{Address: StorageAlias + ":" + strconv.Itoa(storagePort)},
	}

	// A target with a URL runs a proxy on its public network, which checkship's
	// own container joins when it is exposed there: the storage joins it too.
	if url, hasUrl := target.Url().TryGet(); hasUrl {
		project.Networks["default"] = types.NetworkConfig{
			Name:     targetPublicNetworkName(idLower),
			External: true,
		}

		// Picked up by the target's proxy, which routes s3.<host> to the storage
		service.Labels = appendLabels(types.Labels{
			SubdomainLabel: StorageSubdomain,
			"traefik.http.services." + name + ".loadbalancer.server.port": strconv.Itoa(storagePort),
		}, labels)
		endpoints = append(endpoints, domain.StorageEndpoint{
			Address: StorageSubdomain + "." + url.Host(),
			UseSSL:  url.UseSSL(),
		})
	}

	if data, ok := target.Provider().(Data); ok && !data.Host.HasValue() {
		service.Ports = []types.ServicePortConfig{
			{Target: storagePort, Published: strconv.Itoa(storagePort), HostIP: "127.0.0.1"},
		}
		endpoints = append(endpoints, domain.StorageEndpoint{Address: "127.0.0.1:" + strconv.Itoa(storagePort)})
	}

	project.Services = types.Services{storageService: service}

	return project, endpoints
}
