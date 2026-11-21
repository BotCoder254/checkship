package domaincheck

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app/check_app_domains"
	"github.com/checkship-deploy/checkship/pkg/assert"
	"golang.org/x/net/dns/dnsmessage"
)

type fakeResolver struct {
	ips     map[string][]string
	cnames  map[string]string
	ns      map[string][]string
	lookups int
}

func (r *fakeResolver) LookupNS(_ context.Context, name string) ([]*net.NS, error) {
	hosts, found := r.ns[name]
	if !found {
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
	out := make([]*net.NS, len(hosts))
	for i, h := range hosts {
		out[i] = &net.NS{Host: h}
	}
	return out, nil
}

func (r *fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	r.lookups++
	values, found := r.ips[host]

	if !found {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}

	addrs := make([]net.IPAddr, len(values))
	for i, v := range values {
		addrs[i] = net.IPAddr{IP: net.ParseIP(v)}
	}

	return addrs, nil
}

func (r *fakeResolver) LookupCNAME(_ context.Context, host string) (string, error) {
	if cname, found := r.cnames[host]; found {
		return cname, nil
	}

	return host + ".", nil
}

func noCAA(context.Context, string) ([]caaRecord, error) { return nil, nil }

func Test_Checker(t *testing.T) {
	expected := []net.IP{net.ParseIP("127.0.0.1")}

	// A TLS server whose certificate is valid for example.com (httptest's), reached only
	// through the dial hook, which also records where the checker tried to connect.
	newChecker := func(t *testing.T, r *fakeResolver) (*Checker, *[]string) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
		t.Cleanup(server.Close)
		roots := x509.NewCertPool()
		roots.AddCert(server.Certificate())
		dialed := &[]string{}

		return &Checker{
			resolver: r,
			caa:      noCAA,
			roots:    roots,
			now:      time.Now,
			cache:    make(map[string]cached),
			dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				*dialed = append(*dialed, address)
				var d net.Dialer
				return d.DialContext(ctx, network, server.Listener.Addr().String())
			},
		}, dialed
	}

	t.Run("should report a domain pointing here with a valid certificate", func(t *testing.T) {
		c, dialed := newChecker(t, &fakeResolver{ips: map[string][]string{"example.com": {"127.0.0.1"}}})

		status := c.Check(context.Background(), "example.com", "shop.apps.example.net", expected, true)

		assert.Equal(t, check_app_domains.DnsPointsHere, status.Dns)
		assert.Equal(t, check_app_domains.CertificateOk, status.Certificate)
		assert.NotNil(t, status.ExpiresAt)
		assert.DeepEqual(t, []string{"127.0.0.1:443"}, *dialed)
	})

	t.Run("should report the proxy's default certificate when the domain's isn't issued", func(t *testing.T) {
		c, _ := newChecker(t, &fakeResolver{ips: map[string][]string{"shop.example.org": {"127.0.0.1"}}})

		status := c.Check(context.Background(), "shop.example.org", "shop.apps.example.net", expected, true)

		assert.Equal(t, check_app_domains.DnsPointsHere, status.Dns)
		assert.Equal(t, check_app_domains.CertificateDefault, status.Certificate)
	})

	t.Run("should never connect to an address outside the target's", func(t *testing.T) {
		for _, address := range []string{"169.254.169.254", "127.0.0.2", "10.0.0.1", "203.0.113.99"} {
			c, dialed := newChecker(t, &fakeResolver{ips: map[string][]string{"shop.example.com": {address}}})

			status := c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, true)

			assert.Equal(t, check_app_domains.DnsPointsElsewhere, status.Dns, address)
			assert.Equal(t, "", status.Certificate, address)
			assert.HasLength(t, 0, *dialed, address)
		}
	})

	t.Run("should not connect for a domain only reached through a CNAME", func(t *testing.T) {
		c, dialed := newChecker(t, &fakeResolver{
			ips:    map[string][]string{"shop.example.com": {"198.51.100.7"}},
			cnames: map[string]string{"shop.example.com": "shop.apps.example.net."},
		})

		status := c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, true)

		assert.Equal(t, check_app_domains.DnsPointsHere, status.Dns)
		assert.HasLength(t, 0, *dialed)
	})

	t.Run("should recognize Cloudflare's proxy", func(t *testing.T) {
		c, dialed := newChecker(t, &fakeResolver{ips: map[string][]string{"shop.example.com": {"104.16.132.229", "2606:4700::6810:84e5"}}})

		status := c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, true)

		assert.Equal(t, check_app_domains.DnsBehindCloudflare, status.Dns)
		assert.HasLength(t, 0, *dialed)
	})

	t.Run("should tell when the domain's DNS is at Cloudflare", func(t *testing.T) {
		c, _ := newChecker(t, &fakeResolver{ns: map[string][]string{"gitdesk.app": {"ada.ns.cloudflare.com.", "bob.ns.cloudflare.com."}}})

		assert.True(t, c.Check(context.Background(), "www.gitdesk.app", "x.apps.example.net", expected, false).CloudflareDns)

		c, _ = newChecker(t, &fakeResolver{ns: map[string][]string{"example.com": {"ns1.example-dns.net."}}})
		assert.False(t, c.Check(context.Background(), "shop.example.com", "x.apps.example.net", expected, false).CloudflareDns)
	})

	t.Run("should report a missing record", func(t *testing.T) {
		c, _ := newChecker(t, &fakeResolver{})

		status := c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, true)

		assert.Equal(t, check_app_domains.DnsNoRecord, status.Dns)
	})

	t.Run("should not check certificates on http targets", func(t *testing.T) {
		c, dialed := newChecker(t, &fakeResolver{ips: map[string][]string{"example.com": {"127.0.0.1"}}})

		status := c.Check(context.Background(), "example.com", "shop.apps.example.net", expected, false)

		assert.Equal(t, "", status.Certificate)
		assert.HasLength(t, 0, *dialed)
	})

	t.Run("should cache results for a while", func(t *testing.T) {
		r := &fakeResolver{ips: map[string][]string{"shop.example.com": {"198.51.100.7"}}}
		c, _ := newChecker(t, r)

		c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, false)
		lookups := r.lookups
		c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, false)

		assert.Equal(t, lookups, r.lookups)
	})

	t.Run("should report CAA records", func(t *testing.T) {
		r := &fakeResolver{ips: map[string][]string{"shop.example.com": {"127.0.0.1"}}}
		c, _ := newChecker(t, r)
		c.caa = func(context.Context, string) ([]caaRecord, error) {
			return []caaRecord{{tag: "issue", value: "digicert.com"}, {tag: "iodef", value: "mailto:x@example.com"}}, nil
		}

		status := c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, false)
		assert.Equal(t, check_app_domains.CaaBlocks, status.Caa)
		assert.DeepEqual(t, []string{"digicert.com"}, status.CaaIssuers)

		c, _ = newChecker(t, r)
		c.caa = func(context.Context, string) ([]caaRecord, error) { return nil, errors.New("timeout") }
		status = c.Check(context.Background(), "shop.example.com", "shop.apps.example.net", expected, false)
		assert.Equal(t, check_app_domains.CaaUnknown, status.Caa)
	})
}

func Test_CAA(t *testing.T) {
	t.Run("should allow Let's Encrypt without issue records or when listed", func(t *testing.T) {
		ok, _ := allowsLetsEncrypt(nil)
		assert.True(t, ok)

		ok, _ = allowsLetsEncrypt([]caaRecord{{tag: "issuewild", value: "digicert.com"}})
		assert.True(t, ok)

		ok, _ = allowsLetsEncrypt([]caaRecord{{tag: "issue", value: "digicert.com"}, {tag: "issue", value: "letsencrypt.org; validationmethods=http-01"}})
		assert.True(t, ok)

		ok, issuers := allowsLetsEncrypt([]caaRecord{{tag: "issue", value: "sectigo.com"}})
		assert.False(t, ok)
		assert.DeepEqual(t, []string{"sectigo.com"}, issuers)
	})

	t.Run("should read CAA records from a DNS answer", func(t *testing.T) {
		name := dnsmessage.MustNewName("example.com.")
		builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: 42, Response: true})
		assert.Nil(t, builder.StartQuestions())
		assert.Nil(t, builder.Question(dnsmessage.Question{Name: name, Type: typeCAA, Class: dnsmessage.ClassINET}))
		assert.Nil(t, builder.StartAnswers())
		rdata := append([]byte{0, 5}, []byte("issueletsencrypt.org")...)
		assert.Nil(t, builder.UnknownResource(
			dnsmessage.ResourceHeader{Name: name, Type: typeCAA, Class: dnsmessage.ClassINET, TTL: 300},
			dnsmessage.UnknownResource{Type: typeCAA, Data: rdata},
		))
		answer, err := builder.Finish()
		assert.Nil(t, err)

		records, err := parseCAA(answer, 42)
		assert.Nil(t, err)
		assert.DeepEqual(t, []caaRecord{{tag: "issue", value: "letsencrypt.org"}}, records)

		_, err = parseCAA(answer, 43)
		assert.NotNil(t, err)
	})
}
