package cloudflare_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/internal/deployment/infra/cloudflare"
	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

type call struct {
	method, path, query, auth, body string
}

// A fake Cloudflare API answering from a table of "METHOD /path" → (status, JSON envelope).
func fake(t *testing.T, routes map[string]func() (int, string)) (*cloudflare.Client, *[]call) {
	calls := &[]call{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*calls = append(*calls, call{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), string(body)})
		route, found := routes[r.Method+" "+r.URL.Path]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":7003,"message":"not found"}]}`))
			return
		}
		status, payload := route()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(server.Close)
	return cloudflare.New(server.URL), calls
}

func ok(result string) func() (int, string) {
	return func() (int, string) { return 200, `{"success":true,"errors":[],"result":` + result + `}` }
}

func Test_Client(t *testing.T) {
	ctx := context.Background()

	t.Run("should verify an active token and send it as a bearer", func(t *testing.T) {
		c, calls := fake(t, map[string]func() (int, string){"GET /user/tokens/verify": ok(`{"status":"active"}`)})

		assert.Nil(t, c.VerifyToken(ctx, "tok"))
		assert.Equal(t, "Bearer tok", (*calls)[0].auth)
	})

	t.Run("should report a refused token", func(t *testing.T) {
		c, _ := fake(t, map[string]func() (int, string){"GET /user/tokens/verify": func() (int, string) {
			return 400, `{"success":false,"errors":[{"code":1000,"message":"Invalid API Token"}]}`
		}})

		assert.ErrorIs(t, domain.ErrCloudflareTokenRefused, c.VerifyToken(ctx, "tok"))
	})

	t.Run("should verify an account token on its account, not on /user", func(t *testing.T) {
		token := "cfat_" + strings.Repeat("a", 48)
		c, calls := fake(t, map[string]func() (int, string){
			"GET /accounts":                    ok(`[{"id":"acc1","name":"Ada"}]`),
			"GET /accounts/acc1/tokens/verify": ok(`{"status":"active"}`),
			"GET /user/tokens/verify": func() (int, string) {
				return 400, `{"success":false,"errors":[{"code":1000,"message":"Invalid API Token"}]}`
			},
		})

		assert.Nil(t, c.VerifyToken(ctx, token))
		for _, call := range *calls {
			assert.NotEqual(t, "/user/tokens/verify", call.path)
		}
	})

	t.Run("should find an account token's account from its zones", func(t *testing.T) {
		c, _ := fake(t, map[string]func() (int, string){
			"GET /accounts":                    ok(`[]`),
			"GET /zones":                       ok(`[{"id":"z1","name":"gitdesk.app","account":{"id":"acc2","name":"Ada"}}]`),
			"GET /accounts/acc2/tokens/verify": ok(`{"status":"active"}`),
		})

		assert.Nil(t, c.VerifyToken(ctx, "cfat_"+strings.Repeat("b", 48)))
	})

	t.Run("should verify an older token without prefix on its account when /user refuses it", func(t *testing.T) {
		c, _ := fake(t, map[string]func() (int, string){
			"GET /user/tokens/verify": func() (int, string) {
				return 400, `{"success":false,"errors":[{"code":1000,"message":"Invalid API Token"}]}`
			},
			"GET /accounts":                    ok(`[{"id":"acc1"}]`),
			"GET /accounts/acc1/tokens/verify": ok(`{"status":"active"}`),
		})

		assert.Nil(t, c.VerifyToken(ctx, strings.Repeat("c", 40)))
	})

	t.Run("should refuse an expired account token", func(t *testing.T) {
		c, _ := fake(t, map[string]func() (int, string){
			"GET /accounts":                    ok(`[{"id":"acc1"}]`),
			"GET /accounts/acc1/tokens/verify": ok(`{"status":"expired"}`),
		})

		assert.ErrorIs(t, domain.ErrCloudflareTokenRefused, c.VerifyToken(ctx, "cfat_"+strings.Repeat("d", 48)))
	})

	t.Run("should list zones over pages", func(t *testing.T) {
		page := 0
		c, _ := fake(t, map[string]func() (int, string){"GET /zones": func() (int, string) {
			page++
			if page == 1 {
				return 200, `{"success":true,"result":[{"id":"z1","name":"gitdesk.app","account":{"id":"a1","name":"Acme"}}],"result_info":{"page":1,"total_pages":2}}`
			}
			return 200, `{"success":true,"result":[{"id":"z2","name":"acme.dev","account":{"id":"a1","name":"Acme"}}],"result_info":{"page":2,"total_pages":2}}`
		}})

		zones, err := c.Zones(ctx, "tok")

		assert.Nil(t, err)
		assert.DeepEqual(t, []domain.CloudflareZone{
			{ID: "z1", Name: "gitdesk.app", AccountID: "a1", AccountName: "Acme"},
			{ID: "z2", Name: "acme.dev", AccountID: "a1", AccountName: "Acme"},
		}, zones)
	})

	t.Run("should name the missing permission on a 403", func(t *testing.T) {
		c, _ := fake(t, map[string]func() (int, string){"GET /accounts/a1/cfd_tunnel": func() (int, string) {
			return 403, `{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`
		}})

		_, err := c.FindTunnel(ctx, "tok", "a1", "checkship-x")

		assert.ErrorIs(t, domain.ErrCloudflarePermissionMissing, err)
		assert.Equal(t, domain.CloudflarePermissionTunnelEdit, errors.Unwrap(err).Error())
	})

	t.Run("should create proxied records with their comment", func(t *testing.T) {
		c, calls := fake(t, map[string]func() (int, string){"POST /zones/z1/dns_records": ok(`{}`)})

		assert.Nil(t, c.CreateRecord(ctx, "tok", "z1", domain.CloudflareRecord{Type: "CNAME", Name: "gitdesk.app", Content: "t.cfargotunnel.com", Proxied: true, Comment: "managed by checkship (target T)"}))

		var sent map[string]any
		assert.Nil(t, json.Unmarshal([]byte((*calls)[0].body), &sent))
		assert.Equal(t, "CNAME", sent["type"])
		assert.Equal(t, true, sent["proxied"])
		assert.Equal(t, "managed by checkship (target T)", sent["comment"])
	})

	t.Run("should configure a tunnel with a final 404 rule", func(t *testing.T) {
		c, calls := fake(t, map[string]func() (int, string){"PUT /accounts/a1/cfd_tunnel/t1/configurations": ok(`{}`)})

		assert.Nil(t, c.ConfigureTunnel(ctx, "tok", "a1", "t1", []domain.CloudflareIngress{{Hostname: "gitdesk.app", Service: "http://proxy:80"}}))

		body := (*calls)[0].body
		assert.True(t, strings.Contains(body, `"hostname":"gitdesk.app"`))
		assert.True(t, strings.Contains(body, `"service":"http_status:404"`))
	})

	t.Run("should find a tunnel by name and read its token", func(t *testing.T) {
		c, _ := fake(t, map[string]func() (int, string){
			"GET /accounts/a1/cfd_tunnel":          ok(`[{"id":"t1","name":"checkship-x","status":"healthy","connections":[{},{}]}]`),
			"GET /accounts/a1/cfd_tunnel/t1/token": ok(`"eyJh"`),
		})

		found, err := c.FindTunnel(ctx, "tok", "a1", "checkship-x")
		assert.Nil(t, err)
		assert.Equal(t, domain.CloudflareTunnel{ID: "t1", Name: "checkship-x", Status: "healthy", Connections: 2}, found.MustGet())

		token, err := c.TunnelToken(ctx, "tok", "a1", "t1")
		assert.Nil(t, err)
		assert.Equal(t, "eyJh", token)
	})

	t.Run("should treat deleting a missing tunnel as done", func(t *testing.T) {
		c, _ := fake(t, map[string]func() (int, string){})

		assert.Nil(t, c.DeleteTunnel(ctx, "tok", "a1", "gone"))
	})

	t.Run("should report Cloudflare being unreachable", func(t *testing.T) {
		c := cloudflare.New("http://127.0.0.1:1")

		_, err := c.Zones(ctx, "tok")

		assert.ErrorIs(t, domain.ErrCloudflareUnavailable, err)
		_ = apperr.ErrNotFound
	})
}
