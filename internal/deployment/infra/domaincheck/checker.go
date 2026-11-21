// Package domaincheck looks custom domains up (seelf#5): where their DNS points, whether a CAA
// record forbids Let's Encrypt, and which certificate the target serves for them.
//
// Safety: the only connections it makes besides DNS queries are TLS handshakes to port 443 of
// the target's own addresses (the ones its default address resolves to), pinned by IP with the
// domain as SNI. A domain that resolves anywhere else is never connected to, so the check can't
// be aimed at other hosts (server-side request forgery, DNS rebinding). No HTTP request is sent.
package domaincheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/checkship-deploy/checkship/internal/deployment/app/check_app_domains"
	"github.com/checkship-deploy/checkship/internal/deployment/domain"
)

const (
	lookupTimeout    = 3 * time.Second
	handshakeTimeout = 5 * time.Second
	cacheFor         = 30 * time.Second
)

type (
	resolver interface {
		LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
		LookupCNAME(ctx context.Context, host string) (string, error)
		LookupNS(ctx context.Context, name string) ([]*net.NS, error)
	}

	Checker struct {
		resolver resolver
		caa      caaLookup
		dial     func(ctx context.Context, network, address string) (net.Conn, error)
		roots    *x509.CertPool // nil: the system roots
		now      func() time.Time

		mu    sync.Mutex
		cache map[string]cached
	}

	cached struct {
		status check_app_domains.DomainStatus
		until  time.Time
	}

	Option func(*Checker)
)

// Builds a checker using the system resolver, the nameservers of /etc/resolv.conf for CAA and
// the system's trusted roots.
func New(options ...Option) *Checker {
	dialer := &net.Dialer{Timeout: handshakeTimeout}
	c := &Checker{
		resolver: net.DefaultResolver,
		caa:      systemCAALookup,
		dial:     dialer.DialContext,
		now:      time.Now,
		cache:    make(map[string]cached),
	}

	for _, option := range options {
		option(c)
	}

	return c
}

func (c *Checker) Resolve(ctx context.Context, host string) []net.IP {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()

	addrs, err := c.resolver.LookupIPAddr(ctx, host)

	if err != nil {
		return nil
	}

	return ips(addrs)
}

func (c *Checker) Check(
	ctx context.Context,
	hostname domain.Hostname,
	expectedHost string,
	expected []net.IP,
	https bool,
) check_app_domains.DomainStatus {
	key := string(hostname) + "|" + expectedHost + "|" + addressesKey(expected)

	if https {
		key += "|https"
	}

	c.mu.Lock()
	if entry, found := c.cache[key]; found && c.now().Before(entry.until) {
		c.mu.Unlock()
		return entry.status
	}
	c.mu.Unlock()

	status := c.check(ctx, hostname, expectedHost, expected, https)

	c.mu.Lock()
	c.cache[key] = cached{status: status, until: c.now().Add(cacheFor)}
	c.mu.Unlock()

	return status
}

func (c *Checker) check(
	ctx context.Context,
	hostname domain.Hostname,
	expectedHost string,
	expected []net.IP,
	https bool,
) check_app_domains.DomainStatus {
	status := check_app_domains.DomainStatus{
		Hostname:  string(hostname),
		Unicode:   hostname.Unicode(),
		Addresses: []string{},
	}

	// DNS
	lookupCtx, cancel := context.WithTimeout(ctx, lookupTimeout)
	addrs, err := c.resolver.LookupIPAddr(lookupCtx, string(hostname))
	cancel()

	found := ips(addrs)

	for _, ip := range found {
		status.Addresses = append(status.Addresses, ip.String())
	}

	pinned := intersection(found, expected)

	switch {
	case err != nil && isNotFound(err):
		status.Dns = check_app_domains.DnsNoRecord
	case err != nil:
		status.Dns = check_app_domains.DnsUnknown
	case len(pinned) > 0 || c.cnameLeadsTo(ctx, hostname, expectedHost):
		status.Dns = check_app_domains.DnsPointsHere
	case allCloudflare(found):
		status.Dns = check_app_domains.DnsBehindCloudflare
	default:
		status.Dns = check_app_domains.DnsPointsElsewhere
	}

	status.CloudflareDns = c.atCloudflare(ctx, hostname)

	// CAA
	caaCtx, cancel := context.WithTimeout(ctx, lookupTimeout)
	records, err := c.caa(caaCtx, string(hostname))
	cancel()

	if err != nil {
		status.Caa = check_app_domains.CaaUnknown
	} else if allowed, issuers := allowsLetsEncrypt(records); allowed {
		status.Caa = check_app_domains.CaaOk
	} else {
		status.Caa = check_app_domains.CaaBlocks
		status.CaaIssuers = issuers
	}

	// Certificate: only through the target's own addresses
	if https && len(pinned) > 0 {
		c.checkCertificate(ctx, hostname, pinned[0], &status)
	}

	return status
}

// Whether the closest nameservers above the hostname are Cloudflare's (*.ns.cloudflare.com).
func (c *Checker) atCloudflare(ctx context.Context, hostname domain.Hostname) bool {
	labels := strings.Split(string(hostname), ".")

	for i := 0; i < len(labels)-1; i++ {
		lookupCtx, cancel := context.WithTimeout(ctx, lookupTimeout)
		servers, err := c.resolver.LookupNS(lookupCtx, strings.Join(labels[i:], "."))
		cancel()

		if err != nil || len(servers) == 0 {
			continue
		}

		for _, server := range servers {
			if strings.HasSuffix(strings.TrimSuffix(strings.ToLower(server.Host), "."), ".ns.cloudflare.com") {
				return true
			}
		}

		return false
	}

	return false
}

func (c *Checker) cnameLeadsTo(ctx context.Context, hostname domain.Hostname, expectedHost string) bool {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()

	cname, err := c.resolver.LookupCNAME(ctx, string(hostname))

	if err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSuffix(cname, "."), strings.TrimSuffix(expectedHost, "."))
}

// A TLS handshake to the pinned address, never a name: the address is one of the target's.
func (c *Checker) checkCertificate(ctx context.Context, hostname domain.Hostname, ip net.IP, status *check_app_domains.DomainStatus) {
	ctx, cancel := context.WithTimeout(ctx, handshakeTimeout)
	defer cancel()

	conn, err := c.dial(ctx, "tcp", net.JoinHostPort(ip.String(), "443"))

	if err != nil {
		status.Certificate = check_app_domains.CertificateUnreachable
		return
	}

	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client := tls.Client(conn, &tls.Config{
		ServerName: string(hostname),
		RootCAs:    c.roots,
		MinVersion: tls.VersionTLS12,
	})

	err = client.HandshakeContext(ctx)

	var verifyErr *tls.CertificateVerificationError

	switch {
	case err == nil:
		certs := client.ConnectionState().PeerCertificates
		status.Certificate = check_app_domains.CertificateOk

		if len(certs) > 0 {
			expires := certs[0].NotAfter
			status.ExpiresAt = &expires
			status.Issuer = issuerName(certs[0])
		}
	case errors.As(err, &verifyErr):
		// The proxy answered with a certificate that doesn't verify for this name: its own
		// default one, until Let's Encrypt issues the real one.
		status.Certificate = check_app_domains.CertificateDefault
	default:
		status.Certificate = check_app_domains.CertificateUnreachable
	}
}

func issuerName(cert *x509.Certificate) string {
	if len(cert.Issuer.Organization) > 0 {
		return cert.Issuer.Organization[0]
	}

	return cert.Issuer.CommonName
}

func isNotFound(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

func ips(addrs []net.IPAddr) []net.IP {
	out := make([]net.IP, len(addrs))

	for i, addr := range addrs {
		out[i] = addr.IP
	}

	return out
}

// Addresses found for the domain that are also the target's, in the domain's order.
func intersection(found, expected []net.IP) []net.IP {
	var out []net.IP

	for _, ip := range found {
		for _, allowed := range expected {
			if ip.Equal(allowed) {
				out = append(out, ip)
				break
			}
		}
	}

	return out
}

func addressesKey(ips []net.IP) string {
	values := make([]string, len(ips))

	for i, ip := range ips {
		values[i] = ip.String()
	}

	return strings.Join(values, ",")
}
