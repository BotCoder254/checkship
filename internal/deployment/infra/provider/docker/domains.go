package docker

import (
	"crypto/sha1"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/compose-spec/compose-go/v2/types"
)

// Adds one Traefik router per custom domain (seelf#5) to a service's main http entrypoint.
//
// Each domain gets its own router rather than one rule with several Host() matchers: Traefik
// orders one certificate per router, with every host of its rule in it, so a single domain
// whose DNS isn't ready yet would make the whole certificate fail, the default address's
// included. The routers share the entrypoint's service (same container port), and the
// entrypoint's certificate resolver gives each one its own certificate on https targets.
// The default router is left alone: the default address keeps working.
//
// Hostnames are validated by domain.HostnameFrom ([a-z0-9.-] only), so they can't break out
// of the rule's backticks.
func addDomainRouters(labels types.Labels, entrypoint string, hostnames []domain.Hostname) {
	for _, hostname := range hostnames {
		router := domainRouterName(entrypoint, hostname)

		labels["traefik.http.routers."+router+".rule"] = "Host(`" + string(hostname) + "`)"
		labels["traefik.http.routers."+router+".entrypoints"] = httpMainEntryPoint
		labels["traefik.http.routers."+router+".service"] = entrypoint
	}
}

// Stable router name for a custom domain: the same hostname always gives the same router, so
// a redeploy doesn't churn the proxy configuration.
func domainRouterName(entrypoint string, hostname domain.Hostname) string {
	sum := sha1.Sum([]byte(hostname))
	return entrypoint + "-d" + hex.EncodeToString(sum[:4])
}

func joinHostnames(hostnames []domain.Hostname) string {
	values := make([]string, len(hostnames))

	for i, hostname := range hostnames {
		values[i] = string(hostname)
	}

	return strings.Join(values, ", ")
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))

	for key := range m {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}
