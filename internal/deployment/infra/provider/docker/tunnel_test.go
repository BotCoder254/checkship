package docker_test

import (
	"strings"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/provider/docker"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
)

func Test_TunnelConnector(t *testing.T) {
	target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
	assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true)))
	lower := strings.ToLower(string(target.ID()))

	t.Run("should run cloudflared on the proxy's network with the token in its environment", func(t *testing.T) {
		project := docker.TunnelProject(target, "secret-tunnel-token")
		service := project.Services["cloudflared"]

		assert.Equal(t, "checkship-tunnel-"+lower, project.Name)
		assert.Equal(t, docker.CloudflaredImage, service.Image)
		assert.DeepEqual(t, []string{"tunnel", "--no-autoupdate", "run"}, []string(service.Command))
		assert.Equal(t, "secret-tunnel-token", *service.Environment["TUNNEL_TOKEN"])
		assert.False(t, strings.Contains(strings.Join(service.Command, " "), "secret"))
		assert.Equal(t, "checkship-gateway-"+lower, project.Networks["gateway"].Name)
		assert.True(t, project.Networks["gateway"].External)
		assert.Equal(t, string(target.ID()), service.Labels[docker.TargetLabel])
	})

	t.Run("should send visitors to the proxy container", func(t *testing.T) {
		d := docker.New(must.Panic(log.NewLogger()))
		assert.Equal(t, "http://checkship-internal-"+lower+"-proxy-1:80", d.ProxyService(target))

		secure := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
		assert.Nil(t, secure.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("https://apps.example.net")), true)))
		assert.Equal(t, "https://checkship-internal-"+strings.ToLower(string(secure.ID()))+"-proxy-1:443", d.ProxyService(secure))
	})
}
