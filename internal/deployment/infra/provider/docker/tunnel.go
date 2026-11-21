package docker

import (
	"context"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/api"
)

// Connect Cloudflare's tunnel connector, pinned like the proxy's image.
const cloudflaredImage = "cloudflare/cloudflared:2024.12.2"

// Runs cloudflared on the target, on the proxy's network, as its own compose project. It opens
// an outgoing connection to Cloudflare and forwards the tunnel's visitors to the proxy, so the
// target needs no open port. The tunnel token is given through the environment (TUNNEL_TOKEN),
// not the command line, so it doesn't show in process listings.
func (d *docker) RunTunnelConnector(ctx context.Context, target domain.Target, tunnelToken string) error {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	return client.compose.Up(ctx, tunnelProject(target, tunnelToken), api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			QuietPull:     true,
			Recreate:      api.RecreateDiverged,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	})
}

func (d *docker) RemoveTunnelConnector(ctx context.Context, target domain.Target) error {
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	return client.compose.Down(ctx, tunnelProjectName(target.ID()), api.DownOptions{
		RemoveOrphans: true,
	})
}

// Where the connector reaches the proxy: its container on the shared network, on the port of
// the target's main http entrypoint (443 for https targets, where it speaks TLS).
func (d *docker) ProxyService(target domain.Target) string {
	container := targetProjectName(target.ID()) + "-proxy-1"

	if url, isSet := target.Url().TryGet(); isSet && url.UseSSL() {
		return "https://" + container + ":443"
	}

	return "http://" + container + ":80"
}

func tunnelProject(target domain.Target, tunnelToken string) *types.Project {
	name := tunnelProjectName(target.ID())
	network := targetPublicNetworkName(target.ID())
	labels := types.Labels{TargetLabel: string(target.ID())}

	return &types.Project{
		Name: name,
		Networks: types.Networks{
			"gateway": types.NetworkConfig{Name: network, External: true},
		},
		Services: types.Services{
			"cloudflared": types.ServiceConfig{
				Name:         "cloudflared",
				Image:        cloudflaredImage,
				Restart:      types.RestartPolicyUnlessStopped,
				Command:      types.ShellCommand{"tunnel", "--no-autoupdate", "run"},
				Environment:  types.NewMappingWithEquals([]string{"TUNNEL_TOKEN=" + tunnelToken}),
				Labels:       labels,
				CustomLabels: getProjectCustomLabels(name, "cloudflared", ""),
				Networks:     map[string]*types.ServiceNetworkConfig{"gateway": nil},
			},
		},
	}
}

func tunnelProjectName(id domain.TargetID) string {
	return "checkship-tunnel-" + strings.ToLower(string(id))
}
