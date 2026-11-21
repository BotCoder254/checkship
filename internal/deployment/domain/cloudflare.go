package domain

import (
	"context"
	"strings"
	"time"

	"github.com/checkship-deploy/checkship/internal/auth/domain"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/monad"
)

// Connect Cloudflare: with an API token the person creates from a pre-filled link (Zone Read,
// DNS Edit, Cloudflare Tunnel Edit), checkship sets up the DNS records of a target's custom
// domains in their Cloudflare account, and for a target the internet can't reach, a Cloudflare
// Tunnel whose connector runs next to the target's proxy.

var (
	ErrCloudflareTokenInvalid       = apperr.New("cloudflare_token_invalid")        // Not something a Cloudflare token looks like
	ErrCloudflareTokenRefused       = apperr.New("cloudflare_token_refused")        // Cloudflare says it's invalid, expired or not allowed from here
	ErrCloudflarePermissionMissing  = apperr.New("cloudflare_permission_missing")   // Detail: which permission
	ErrCloudflareUnavailable        = apperr.New("cloudflare_unavailable")          // Cloudflare's API couldn't be reached or failed
	ErrCloudflareNotConnected       = apperr.New("cloudflare_not_connected")        // The target has no Cloudflare connection
	ErrCloudflareInvalidMode        = apperr.New("cloudflare_invalid_mode")         // Neither tunnel nor dns
	ErrCloudflareTunnelNotAllowed   = apperr.New("cloudflare_tunnel_not_allowed")   // The token can't edit tunnels
	ErrCloudflareAutomaticProxyOnly = apperr.New("cloudflare_automatic_proxy_only") // A manual proxy target can't be routed
)

const (
	CloudflareModeTunnel CloudflareMode = "tunnel" // Through a Cloudflare Tunnel, for a target the internet can't reach
	CloudflareModeDns    CloudflareMode = "dns"    // A proxied DNS record to the target's public address

	// Permissions, as named on Cloudflare's token page
	CloudflarePermissionZoneRead   = "Zone · Read"
	CloudflarePermissionDnsEdit    = "DNS · Edit"
	CloudflarePermissionTunnelEdit = "Cloudflare Tunnel · Edit"
)

type (
	CloudflareMode string

	// What a target is connected to on Cloudflare. The token is a secret: it is only ever sent
	// to Cloudflare's API and never returned by checkship's.
	CloudflareConnection struct {
		Target        TargetID
		Token         string
		AccountID     string
		AccountName   string
		TunnelAllowed bool // The token can edit tunnels
		Mode          CloudflareMode
		TunnelID      monad.Maybe[string]
		TunnelName    monad.Maybe[string]
		ConnectedAt   time.Time
		ConnectedBy   domain.UserID
	}

	CloudflareConnections interface {
		GetCloudflareConnection(context.Context, TargetID) (CloudflareConnection, error) // apperr.ErrNotFound when none
		SaveCloudflareConnection(context.Context, CloudflareConnection) error
		DeleteCloudflareConnection(context.Context, TargetID) error
	}

	CloudflareZone struct {
		ID          string
		Name        string
		AccountID   string
		AccountName string
	}

	CloudflareRecord struct {
		ID      string
		Type    string
		Name    string
		Content string
		Proxied bool
		Comment string
	}

	CloudflareTunnel struct {
		ID          string
		Name        string
		Status      string // healthy, degraded, inactive, down
		Connections int
	}

	// One public hostname of a tunnel: where Cloudflare sends its visitors.
	CloudflareIngress struct {
		Hostname    string
		Service     string // http://proxy:80
		NoTLSVerify bool   // The proxy's certificate may not be issued yet (https targets)
	}

	// Cloudflare's API, with the person's token on every call.
	Cloudflare interface {
		VerifyToken(ctx context.Context, token string) error
		Zones(ctx context.Context, token string) ([]CloudflareZone, error)
		Records(ctx context.Context, token, zone, name string) ([]CloudflareRecord, error)
		RecordsWithComment(ctx context.Context, token, zone, comment string) ([]CloudflareRecord, error)
		CreateRecord(ctx context.Context, token, zone string, record CloudflareRecord) error
		UpdateRecord(ctx context.Context, token, zone string, record CloudflareRecord) error
		DeleteRecord(ctx context.Context, token, zone, id string) error
		FindTunnel(ctx context.Context, token, account, name string) (monad.Maybe[CloudflareTunnel], error)
		CreateTunnel(ctx context.Context, token, account, name string) (CloudflareTunnel, error)
		Tunnel(ctx context.Context, token, account, id string) (CloudflareTunnel, error)
		TunnelToken(ctx context.Context, token, account, id string) (string, error)
		ConfigureTunnel(ctx context.Context, token, account, id string, ingress []CloudflareIngress) error
		DeleteTunnel(ctx context.Context, token, account, id string) error
	}

	// Runs the tunnel's connector (cloudflared) on a target, next to its proxy.
	TunnelConnector interface {
		RunTunnelConnector(ctx context.Context, target Target, tunnelToken string) error
		RemoveTunnelConnector(ctx context.Context, target Target) error
		// Where the connector reaches the target's proxy, as seen from its network.
		ProxyService(target Target) string
	}

	// A custom domain served by a target: what checkship must route there.
	TargetDomain struct {
		Hostname    Hostname
		App         AppID
		AppName     AppName
		Environment Environment
		Service     string
	}
)

func CloudflareModeFrom(value string) (CloudflareMode, error) {
	switch CloudflareMode(value) {
	case CloudflareModeTunnel, CloudflareModeDns:
		return CloudflareMode(value), nil
	default:
		return "", ErrCloudflareInvalidMode
	}
}

// The comment on every DNS record checkship creates for a target: the only records it may later
// update or delete there.
func CloudflareRecordComment(target TargetID) string {
	return "managed by checkship (target " + string(target) + ")"
}

// Name of the tunnel checkship creates for a target.
func CloudflareTunnelName(target TargetID) string {
	return "checkship-" + strings.ToLower(string(target))
}

// Cloudflare API tokens are 40 characters of [A-Za-z0-9_-] today; a little room is left for
// longer formats. Anything else is refused before being sent anywhere.
func ValidCloudflareToken(token string) bool {
	if len(token) < 32 || len(token) > 128 {
		return false
	}

	for _, r := range token {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return false
		}
	}

	return true
}

// The zone a hostname belongs to, among the given ones: the longest zone name it equals or ends with.
func ZoneOf(hostname Hostname, zones []CloudflareZone) (CloudflareZone, bool) {
	var (
		best  CloudflareZone
		found bool
	)

	for _, zone := range zones {
		if hostname.IsUnder(zone.Name) && len(zone.Name) > len(best.Name) {
			best, found = zone, true
		}
	}

	return best, found
}
