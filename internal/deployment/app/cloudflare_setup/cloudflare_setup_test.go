package cloudflare_setup_test

import (
	"context"
	"strconv"
	"testing"

	authfixture "github.com/checkship-deploy/checkship/internal/auth/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/app/cloudflare_setup"
	"github.com/checkship-deploy/checkship/internal/deployment/app/create_app"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/fixture"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/sqlite"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"github.com/checkship-deploy/checkship/pkg/monad"
	"github.com/checkship-deploy/checkship/pkg/must"
	"github.com/checkship-deploy/checkship/pkg/validate"
)

const goodToken = "abcdefghijklmnopqrstuvwxyz0123456789ABCD"

// An in-memory Cloudflare account.
type fakeCloudflare struct {
	zones          []domain.CloudflareZone
	records        map[string][]domain.CloudflareRecord // per zone
	tunnels        map[string]domain.CloudflareTunnel
	ingress        []domain.CloudflareIngress
	tunnelAllowed  bool
	tunnelReadOnly bool // Can list tunnels (Read) but not create them (Edit)
	dnsReadOnly    bool // Can read DNS records but not write them
	nextID         int
}

func newFake() *fakeCloudflare {
	return &fakeCloudflare{
		zones:         []domain.CloudflareZone{{ID: "z1", Name: "gitdesk.app", AccountID: "a1", AccountName: "Acme"}},
		records:       map[string][]domain.CloudflareRecord{},
		tunnels:       map[string]domain.CloudflareTunnel{},
		tunnelAllowed: true,
	}
}

func (f *fakeCloudflare) id() string { f.nextID++; return "id" + strconv.Itoa(f.nextID) }

func (f *fakeCloudflare) VerifyToken(_ context.Context, token string) error {
	if token != goodToken {
		return domain.ErrCloudflareTokenRefused
	}
	return nil
}
func (f *fakeCloudflare) Zones(context.Context, string) ([]domain.CloudflareZone, error) {
	return f.zones, nil
}
func (f *fakeCloudflare) Records(_ context.Context, _, zone, name string) ([]domain.CloudflareRecord, error) {
	var out []domain.CloudflareRecord
	for _, r := range f.records[zone] {
		if r.Name == name {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeCloudflare) RecordsWithComment(_ context.Context, _, zone, comment string) ([]domain.CloudflareRecord, error) {
	var out []domain.CloudflareRecord
	for _, r := range f.records[zone] {
		if r.Comment == comment {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeCloudflare) CreateRecord(_ context.Context, _, zone string, r domain.CloudflareRecord) error {
	if f.dnsReadOnly {
		return apperr.NewWithDetail(domain.ErrCloudflarePermissionMissing.Error(), nil)
	}
	r.ID = f.id()
	f.records[zone] = append(f.records[zone], r)
	return nil
}
func (f *fakeCloudflare) UpdateRecord(_ context.Context, _, zone string, r domain.CloudflareRecord) error {
	if f.dnsReadOnly {
		return apperr.NewWithDetail(domain.ErrCloudflarePermissionMissing.Error(), nil)
	}
	for i, existing := range f.records[zone] {
		if existing.ID == r.ID {
			f.records[zone][i] = r
		}
	}
	return nil
}
func (f *fakeCloudflare) DeleteRecord(_ context.Context, _, zone, id string) error {
	if f.dnsReadOnly {
		return apperr.NewWithDetail(domain.ErrCloudflarePermissionMissing.Error(), nil)
	}
	var kept []domain.CloudflareRecord
	for _, r := range f.records[zone] {
		if r.ID != id {
			kept = append(kept, r)
		}
	}
	f.records[zone] = kept
	return nil
}
func (f *fakeCloudflare) FindTunnel(_ context.Context, _, _, name string) (monad.Maybe[domain.CloudflareTunnel], error) {
	if !f.tunnelAllowed {
		return monad.None[domain.CloudflareTunnel](), apperr.NewWithDetail(domain.ErrCloudflarePermissionMissing.Error(), nil)
	}
	for _, t := range f.tunnels {
		if t.Name == name {
			return monad.Value(t), nil
		}
	}
	return monad.None[domain.CloudflareTunnel](), nil
}
func (f *fakeCloudflare) CreateTunnel(_ context.Context, _, _, name string) (domain.CloudflareTunnel, error) {
	if f.tunnelReadOnly {
		return domain.CloudflareTunnel{}, apperr.NewWithDetail(domain.ErrCloudflarePermissionMissing.Error(), nil)
	}
	t := domain.CloudflareTunnel{ID: f.id(), Name: name, Status: "inactive"}
	f.tunnels[t.ID] = t
	return t, nil
}
func (f *fakeCloudflare) Tunnel(_ context.Context, _, _, id string) (domain.CloudflareTunnel, error) {
	if t, found := f.tunnels[id]; found {
		return t, nil
	}
	return domain.CloudflareTunnel{}, apperr.ErrNotFound
}
func (f *fakeCloudflare) TunnelToken(context.Context, string, string, string) (string, error) {
	return "tunnel-token", nil
}
func (f *fakeCloudflare) ConfigureTunnel(_ context.Context, _, _, _ string, ingress []domain.CloudflareIngress) error {
	f.ingress = ingress
	return nil
}
func (f *fakeCloudflare) DeleteTunnel(_ context.Context, _, _, id string) error {
	delete(f.tunnels, id)
	return nil
}

type fakeConnector struct {
	running bool
	token   string
}

func (c *fakeConnector) RunTunnelConnector(_ context.Context, _ domain.Target, token string) error {
	c.running, c.token = true, token
	return nil
}
func (c *fakeConnector) RemoveTunnelConnector(context.Context, domain.Target) error {
	c.running = false
	return nil
}
func (c *fakeConnector) ProxyService(domain.Target) string { return "http://proxy:80" }

func Test_CloudflareSetup(t *testing.T) {
	type world struct {
		ctx       context.Context
		db        *fixture.Context
		deps      cloudflare_setup.Deps
		cf        *fakeCloudflare
		connector *fakeConnector
		target    domain.Target
		app       string
	}

	arrange := func(t *testing.T, targetUrl string) world {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom(targetUrl)), true)))
		db := fixture.PrepareDatabase(t, fixture.WithUsers(&user), fixture.WithTargets(&target))
		cf, connector := newFake(), &fakeConnector{}

		deps := cloudflare_setup.Deps{
			Connections: sqlite.NewCloudflareConnectionsStore(db.DB),
			Cloudflare:  cf,
			Targets:     db.TargetsStore,
			Apps:        db.AppsStore,
			Connector:   connector,
		}

		create := create_app.Handler(db.AppsStore, db.AppsStore, create_app.DomainsRules{Targets: db.TargetsStore, Apps: db.AppsStore})
		production := create_app.EnvironmentConfig{Target: string(target.ID())}
		production.Domains.Set(map[string][]string{"web": {"gitdesk.app"}})
		app, err := create(db.Context, create_app.Command{Name: "gitdesk", Production: production, Staging: create_app.EnvironmentConfig{Target: string(target.ID())}})
		assert.Nil(t, err)

		return world{ctx: db.Context, db: db, deps: deps, cf: cf, connector: connector, target: target, app: app}
	}

	connect := func(w world, token string) (cloudflare_setup.Status, error) {
		return cloudflare_setup.ConnectHandler(w.deps)(w.ctx, cloudflare_setup.ConnectCommand{TargetID: string(w.target.ID()), Token: token})
	}

	sync := func(w world, mode string, replace ...string) (cloudflare_setup.SyncResult, error) {
		cmd := cloudflare_setup.SyncCommand{TargetID: string(w.target.ID()), Replace: replace}
		if mode != "" {
			cmd.Mode.Set(mode)
		}
		return cloudflare_setup.SyncHandler(w.deps)(w.ctx, cmd)
	}

	t.Run("should refuse malformed and refused tokens", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")

		_, err := connect(w, "not a token")
		assert.ValidationError(t, validate.FieldErrors{"token": domain.ErrCloudflareTokenInvalid}, err)

		_, err = connect(w, "zyxwvutsrqponmlkjihgfedcba9876543210ABCD")
		assert.ValidationError(t, validate.FieldErrors{"token": domain.ErrCloudflareTokenRefused}, err)
	})

	t.Run("should connect and pick the tunnel for a target the internet can't reach", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")

		status, err := connect(w, "  "+goodToken+" ")

		assert.Nil(t, err)
		assert.True(t, status.Connected)
		assert.Equal(t, "Acme", status.AccountName)
		assert.Equal(t, "tunnel", status.Mode)
		assert.DeepEqual(t, []string{"gitdesk.app"}, status.Zones)
	})

	t.Run("should set up a tunnel, its route, the record and the connector", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		_, err := connect(w, goodToken)
		assert.Nil(t, err)

		result, err := sync(w, "")

		assert.Nil(t, err)
		assert.Equal(t, 1, len(w.cf.tunnels))
		assert.DeepEqual(t, []domain.CloudflareIngress{{Hostname: "gitdesk.app", Service: "http://proxy:80"}}, w.cf.ingress)
		records := w.cf.records["z1"]
		assert.HasLength(t, 1, records)
		assert.Equal(t, "CNAME", records[0].Type)
		assert.Equal(t, w.cf.ingressTunnelID()+".cfargotunnel.com", records[0].Content)
		assert.True(t, records[0].Proxied)
		assert.Equal(t, domain.CloudflareRecordComment(w.target.ID()), records[0].Comment)
		assert.True(t, w.connector.running)
		assert.Equal(t, "tunnel-token", w.connector.token)
		assert.HasLength(t, 0, result.Conflicts)

		// Again: nothing new, the same tunnel and record
		_, err = sync(w, "")
		assert.Nil(t, err)
		assert.Equal(t, 1, len(w.cf.tunnels))
		assert.HasLength(t, 1, w.cf.records["z1"])
	})

	t.Run("should leave someone else's record alone unless asked to replace it", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		w.cf.records["z1"] = []domain.CloudflareRecord{{ID: "theirs", Type: "A", Name: "gitdesk.app", Content: "198.51.100.7", Proxied: true}}
		_, err := connect(w, goodToken)
		assert.Nil(t, err)

		result, err := sync(w, "")
		assert.Nil(t, err)
		assert.DeepEqual(t, []cloudflare_setup.Conflict{{
			Hostname: "gitdesk.app",
			Type:     "A",
			Content:  "198.51.100.7",
			Zone:     "gitdesk.app",
			Want:     cloudflare_setup.Wanted{Type: "CNAME", Name: "@", Content: w.cf.ingressTunnelID() + ".cfargotunnel.com", Proxied: true},
			Reason:   cloudflare_setup.ConflictRecord,
		}}, result.Conflicts)
		assert.Equal(t, "theirs", w.cf.records["z1"][0].ID)

		result, err = sync(w, "", "gitdesk.app")
		assert.Nil(t, err)
		assert.HasLength(t, 0, result.Conflicts)
		assert.HasLength(t, 1, w.cf.records["z1"])
		assert.Equal(t, "CNAME", w.cf.records["z1"][0].Type)
	})

	t.Run("should take over a record made by hand as the drawer showed", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		w.cf.records["z1"] = []domain.CloudflareRecord{{ID: "theirs", Type: "AAAA", Name: "gitdesk.app", Content: "100::", Proxied: true}}
		_, err := connect(w, goodToken)
		assert.Nil(t, err)

		result, err := sync(w, "")
		assert.Nil(t, err)
		assert.HasLength(t, 1, result.Conflicts)

		// The person replaces their AAAA with the CNAME shown, by hand, without the comment
		target := result.Conflicts[0].Want.Content
		w.cf.records["z1"] = []domain.CloudflareRecord{{ID: "byhand", Type: "CNAME", Name: "gitdesk.app", Content: target + ".", Proxied: false}}

		result, err = sync(w, "")
		assert.Nil(t, err)
		assert.HasLength(t, 0, result.Conflicts)
		records := w.cf.records["z1"]
		assert.HasLength(t, 1, records)
		assert.Equal(t, "byhand", records[0].ID)
		assert.True(t, records[0].Proxied)
		assert.Equal(t, domain.CloudflareRecordComment(w.target.ID()), records[0].Comment)
		assert.Equal(t, 1, len(result.Status.Records))
	})

	t.Run("should ask for the record by hand when the token can't write DNS, then accept it", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		w.cf.dnsReadOnly = true
		_, err := connect(w, goodToken)
		assert.Nil(t, err)

		result, err := sync(w, "tunnel")
		assert.Nil(t, err)
		assert.HasLength(t, 1, result.Conflicts)
		assert.Equal(t, cloudflare_setup.ConflictDnsEdit, result.Conflicts[0].Reason)
		assert.Equal(t, "", result.Conflicts[0].Type)
		assert.True(t, w.connector.running) // The tunnel side is done all the same

		// Added by hand, proxied: it works, even though checkship can't mark it as its own
		target := result.Conflicts[0].Want.Content
		w.cf.records["z1"] = []domain.CloudflareRecord{{ID: "byhand", Type: "CNAME", Name: "gitdesk.app", Content: target, Proxied: true}}
		result, err = sync(w, "")
		assert.Nil(t, err)
		assert.HasLength(t, 0, result.Conflicts)

		// Not proxied: still to fix
		w.cf.records["z1"][0].Proxied = false
		result, err = sync(w, "")
		assert.Nil(t, err)
		assert.HasLength(t, 1, result.Conflicts)
		assert.Equal(t, "CNAME", result.Conflicts[0].Type)
	})

	t.Run("should switch to a DNS record and drop the tunnel", func(t *testing.T) {
		w := arrange(t, "https://apps.example.net")
		_, err := connect(w, goodToken)
		assert.Nil(t, err)
		_, err = sync(w, "tunnel")
		assert.Nil(t, err)

		_, err = sync(w, "dns")

		assert.Nil(t, err)
		assert.Equal(t, 0, len(w.cf.tunnels))
		assert.False(t, w.connector.running)
		assert.Equal(t, "gitdesk.apps.example.net", w.cf.records["z1"][0].Content)
	})

	t.Run("should not offer tunnels without the permission", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		w.cf.tunnelAllowed = false

		status, err := connect(w, goodToken)
		assert.Nil(t, err)
		assert.False(t, status.TunnelAllowed)
		assert.Equal(t, "dns", status.Mode)

		_, err = sync(w, "tunnel")
		assert.ErrorIs(t, domain.ErrCloudflarePermissionMissing, err)
	})

	t.Run("should fall back to DNS when the token can only read tunnels, and retry once it can edit them", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		w.cf.tunnelReadOnly = true

		status, err := connect(w, goodToken)
		assert.Nil(t, err)
		assert.True(t, status.TunnelAllowed) // Listing worked: Connect can't tell Read from Edit

		_, err = sync(w, "tunnel")
		assert.ErrorIs(t, domain.ErrCloudflarePermissionMissing, err)
		status, _ = cloudflare_setup.StatusHandler(w.deps)(w.ctx, cloudflare_setup.StatusQuery{TargetID: string(w.target.ID())})
		assert.False(t, status.TunnelAllowed)
		assert.Equal(t, "dns", status.Mode)

		// The person adds Tunnel Edit to the same token on Cloudflare, then sets up again
		w.cf.tunnelReadOnly = false
		result, err := sync(w, "tunnel")
		assert.Nil(t, err)
		assert.True(t, result.Status.TunnelAllowed)
		assert.Equal(t, "tunnel", result.Status.Mode)
		assert.True(t, w.connector.running)
	})

	t.Run("should list domains outside the token's account", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		w.cf.zones = []domain.CloudflareZone{{ID: "z9", Name: "other.dev", AccountID: "a1", AccountName: "Acme"}}
		_, err := connect(w, goodToken)
		assert.Nil(t, err)

		result, err := sync(w, "")

		assert.Nil(t, err)
		assert.DeepEqual(t, []string{"gitdesk.app"}, result.Outside)
	})

	t.Run("should remove everything it created on disconnect, and only that", func(t *testing.T) {
		w := arrange(t, "http://docker.localhost")
		w.cf.records["z1"] = []domain.CloudflareRecord{{ID: "mx", Type: "MX", Name: "gitdesk.app", Content: "mail.gitdesk.app"}}
		_, err := connect(w, goodToken)
		assert.Nil(t, err)
		_, err = sync(w, "")
		assert.Nil(t, err)

		_, err = cloudflare_setup.DisconnectHandler(w.deps)(w.ctx, cloudflare_setup.DisconnectCommand{TargetID: string(w.target.ID())})

		assert.Nil(t, err)
		assert.Equal(t, 0, len(w.cf.tunnels))
		assert.False(t, w.connector.running)
		assert.DeepEqual(t, []domain.CloudflareRecord{{ID: "mx", Type: "MX", Name: "gitdesk.app", Content: "mail.gitdesk.app"}}, w.cf.records["z1"])
		_, err = w.deps.Connections.GetCloudflareConnection(w.ctx, w.target.ID())
		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})
}

func (f *fakeCloudflare) ingressTunnelID() string {
	for id := range f.tunnels {
		return id
	}
	return ""
}
