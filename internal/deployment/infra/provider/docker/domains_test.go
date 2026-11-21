package docker_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkship-deploy/checkship/cmd/config"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/artifact"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/provider/docker"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/source/raw"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/checkship-deploy/checkship/pkg/must"
)

func Test_CustomDomainRouters(t *testing.T) {
	logger := must.Panic(log.NewLogger())

	deploy := func(t *testing.T, domains domain.ServicesDomains) (*dockerMockService, domain.Services, string) {
		target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
		assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("https://apps.example.com")), true)))
		production := domain.NewEnvironmentConfig(target.ID())
		production.HasDomains(domains)
		app := fixture.App(fixture.WithAppName("shop"), fixture.WithEnvironmentConfig(production, domain.NewEnvironmentConfig(target.ID())))
		deployment := fixture.Deployment(
			fixture.FromApp(app),
			fixture.ForEnvironment(domain.Production),
			fixture.WithSourceData(raw.Data(`services:
  web:
    image: traefik/whoami
    ports:
      - "8080:80"
      - "8081:81"
  worker:
    image: traefik/whoami
`)),
		)

		opts := config.Default(config.WithTestDefaults())
		t.Cleanup(func() { os.RemoveAll(opts.DataDir()) })
		artifactManager := artifact.NewLocal(opts, logger)
		deploymentContext, err := artifactManager.PrepareBuild(context.Background(), deployment)
		assert.Nil(t, err)
		assert.Nil(t, raw.New().Fetch(context.Background(), deploymentContext, deployment))
		mock := newMockService()
		provider := docker.New(logger, docker.WithTestConfig(mock, mock, filepath.Join(opts.DataDir(), "config")))

		services, err := provider.Deploy(context.Background(), deploymentContext, deployment, target, nil)
		assert.Nil(t, err)
		deploymentContext.Logger().Close()

		log, _ := os.ReadFile(artifactManager.LogPath(context.Background(), deployment))

		return mock, services, string(log)
	}

	t.Run("should add one router per custom domain on the main http entrypoint", func(t *testing.T) {
		mock, services, log := deploy(t, domain.ServicesDomains{"web": {"shop.example.com", "www.shop.example.com"}})

		labels := mock.ups[0].project.Services["web"].Labels
		main := string(services.Entrypoints()[0].Name())

		// The default router keeps the proxy's default rule (the default address)
		_, hasRule := labels["traefik.http.routers."+main+".rule"]
		assert.False(t, hasRule)

		var rules []string
		for key, value := range labels {
			if strings.HasPrefix(key, "traefik.http.routers."+main+"-d") && strings.HasSuffix(key, ".rule") {
				rules = append(rules, value)
				router := strings.TrimSuffix(key, ".rule")
				assert.Equal(t, "http", labels[router+".entrypoints"])
				assert.Equal(t, main, labels[router+".service"])
			}
		}

		assert.HasLength(t, 2, rules)
		assert.True(t, strings.Contains(strings.Join(rules, " "), "Host(`shop.example.com`)"))
		assert.True(t, strings.Contains(strings.Join(rules, " "), "Host(`www.shop.example.com`)"))

		// Recorded on the service, for its URLs; the second http port is a custom entrypoint
		assert.DeepEqual(t, []domain.Hostname{"shop.example.com", "www.shop.example.com"}, services[0].Domains())
		assert.HasLength(t, 1, services[1:])
		assert.HasLength(t, 0, services[1].Domains())
		assert.True(t, strings.Contains(log, "[INFO] serving web on shop.example.com, www.shop.example.com"))
	})

	t.Run("should keep router names stable across deployments", func(t *testing.T) {
		first, _, _ := deploy(t, domain.ServicesDomains{"web": {"shop.example.com"}})
		second, _, _ := deploy(t, domain.ServicesDomains{"web": {"shop.example.com"}})

		keys := func(m *dockerMockService) []string {
			var out []string
			for key := range m.ups[0].project.Services["web"].Labels {
				if strings.Contains(key, "-d") && strings.HasSuffix(key, ".rule") {
					out = append(out, key[strings.Index(key, "-d"):])
				}
			}
			return out
		}

		assert.DeepEqual(t, keys(first), keys(second))
	})

	t.Run("should say in the log why domains could not be applied", func(t *testing.T) {
		_, _, log := deploy(t, domain.ServicesDomains{
			"worker": {"worker.example.com"},
			"ghost":  {"ghost.example.com"},
		})

		assert.True(t, strings.Contains(log, "[WARN] custom domains of service worker ignored: it exposes no http port"))
		assert.True(t, strings.Contains(log, "[WARN] custom domains of service ghost ignored: there is no such service in the compose file"))
	})
}
