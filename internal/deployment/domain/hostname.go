package domain

import (
	"database/sql/driver"
	"net"
	"regexp"
	"slices"
	"strings"

	"github.com/checkship-deploy/checkship/pkg/apperr"
	"github.com/checkship-deploy/checkship/pkg/storage"
	"golang.org/x/net/idna"
)

var (
	ErrInvalidHostname                 = apperr.New("invalid_hostname")
	ErrDomainAlreadyUsed               = apperr.New("domain_already_used")
	ErrDomainReservedByTarget          = apperr.New("domain_reserved_by_target")
	ErrDomainReserved                  = apperr.New("domain_reserved")
	ErrDomainDuplicated                = apperr.New("domain_duplicated")
	ErrTooManyDomains                  = apperr.New("too_many_domains")
	ErrCustomDomainsNeedAutomaticProxy = apperr.New("custom_domains_need_automatic_proxy")
)

const (
	MaxDomainsPerService     = 20
	MaxDomainsPerEnvironment = 50
	maxHostnameLength        = 253
)

// One DNS label: letters, digits and inner hyphens, 1 to 63 characters (RFC 1123). Only these
// characters ever reach a Traefik rule, so a hostname can't break out of its backticks.
var hostnameLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type (
	// A fully qualified domain name an application environment is served at (seelf#5), stored
	// in its ASCII form: internationalized names are converted to punycode.
	Hostname string

	// Custom domains of an application environment, per compose service.
	ServicesDomains map[string][]Hostname
)

// Parses a hostname typed by someone: trimmed, lower cased, converted to ASCII. Refuses
// schemes, ports, paths, wildcards, IP addresses and single labels.
func HostnameFrom(raw string) (Hostname, error) {
	value := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")

	if value == "" || strings.ContainsAny(value, "/:*@?#[] \t`'\"\\") {
		return "", ErrInvalidHostname
	}

	if net.ParseIP(value) != nil {
		return "", ErrInvalidHostname
	}

	ascii, err := idna.Lookup.ToASCII(value)

	if err != nil || len(ascii) > maxHostnameLength {
		return "", ErrInvalidHostname
	}

	labels := strings.Split(ascii, ".")

	if len(labels) < 2 {
		return "", ErrInvalidHostname
	}

	for _, label := range labels {
		if !hostnameLabel.MatchString(label) {
			return "", ErrInvalidHostname
		}
	}

	// The last label is a top-level domain: never all digits (that would be an IP-ish name)
	if strings.Trim(labels[len(labels)-1], "0123456789") == "" {
		return "", ErrInvalidHostname
	}

	return Hostname(ascii), nil
}

func (h Hostname) String() string { return string(h) }

// The internationalized form of this hostname, for display. Equal to the hostname when it
// is plain ASCII.
func (h Hostname) Unicode() string {
	if unicode, err := idna.Display.ToUnicode(string(h)); err == nil {
		return unicode
	}

	return string(h)
}

// Tells whether this hostname is the given host or one of its subdomains.
func (h Hostname) IsUnder(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")

	if host == "" {
		return false
	}

	return string(h) == host || strings.HasSuffix(string(h), "."+host)
}

// Every hostname, sorted.
func (d ServicesDomains) All() []Hostname {
	var all []Hostname

	for _, hostnames := range d {
		all = append(all, hostnames...)
	}

	slices.Sort(all)

	return all
}

// Hostnames of the given service, in the order they were given (the first one is the
// primary of that service).
func (d ServicesDomains) For(service string) []Hostname {
	return d[service]
}

func (d ServicesDomains) Value() (driver.Value, error) { return storage.ValueJSON(d) }
func (d *ServicesDomains) Scan(value any) error        { return storage.ScanJSON(value, d) }
