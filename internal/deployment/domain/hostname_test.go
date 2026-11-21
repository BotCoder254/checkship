package domain_test

import (
	"testing"

	"github.com/checkship-deploy/checkship/internal/deployment/domain"
	"github.com/checkship-deploy/checkship/pkg/assert"
)

func Test_Hostname(t *testing.T) {
	t.Run("should accept and normalize valid hostnames", func(t *testing.T) {
		for raw, expected := range map[string]string{
			"shop.example.com":      "shop.example.com",
			"  Shop.Example.COM.  ": "shop.example.com",
			"a-b.c-d.example.co.uk": "a-b.c-d.example.co.uk",
			"bücher.example":        "xn--bcher-kva.example",
			"shop.docker.localhost": "shop.docker.localhost",
			"x1.example.com":        "x1.example.com",
			"123.example.com":       "123.example.com",
			"xn--bcher-kva.example": "xn--bcher-kva.example",
		} {
			h, err := domain.HostnameFrom(raw)
			assert.Nil(t, err, raw)
			assert.Equal(t, domain.Hostname(expected), h, raw)
		}
	})

	t.Run("should refuse anything that is not a plain hostname", func(t *testing.T) {
		for _, raw := range []string{
			"",
			"localhost",
			"https://shop.example.com",
			"shop.example.com/path",
			"shop.example.com:8080",
			"*.example.com",
			"user@shop.example.com",
			"203.0.113.10",
			"2001:db8::1",
			"[2001:db8::1]",
			"shop..example.com",
			"-shop.example.com",
			"shop-.example.com",
			"sh op.example.com",
			"shop_1.example.com",
			"shop.example.com`) || Host(`evil.com",
			"shop.example.123",
		} {
			_, err := domain.HostnameFrom(raw)
			assert.ErrorIs(t, domain.ErrInvalidHostname, err, raw)
		}
	})

	t.Run("should refuse labels over 63 characters and names over 253", func(t *testing.T) {
		long := ""
		for range 64 {
			long += "a"
		}
		_, err := domain.HostnameFrom(long + ".example.com")
		assert.ErrorIs(t, domain.ErrInvalidHostname, err)

		name := ""
		for range 50 {
			name += "abcd."
		}
		_, err = domain.HostnameFrom(name + "example.com")
		assert.ErrorIs(t, domain.ErrInvalidHostname, err)
	})

	t.Run("should tell whether it is under a host", func(t *testing.T) {
		h := domain.Hostname("shop.apps.example.com")

		assert.True(t, h.IsUnder("apps.example.com"))
		assert.True(t, h.IsUnder("APPS.example.com."))
		assert.True(t, domain.Hostname("apps.example.com").IsUnder("apps.example.com"))
		assert.False(t, h.IsUnder("ps.example.com"))
		assert.False(t, h.IsUnder(""))
	})

	t.Run("should show internationalized names in unicode", func(t *testing.T) {
		assert.Equal(t, "bücher.example", domain.Hostname("xn--bcher-kva.example").Unicode())
		assert.Equal(t, "shop.example.com", domain.Hostname("shop.example.com").Unicode())
	})
}

func Test_EnvironmentConfigDomains(t *testing.T) {
	t.Run("should drop empty services and treat no domain as none", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("target")
		config.HasDomains(domain.ServicesDomains{"web": {}, "api": nil})

		assert.False(t, config.Domains().HasValue())
	})

	t.Run("should compare domains", func(t *testing.T) {
		a := domain.NewEnvironmentConfig("target")
		b := domain.NewEnvironmentConfig("target")
		assert.True(t, a.Equals(b))

		a.HasDomains(domain.ServicesDomains{"web": {"shop.example.com"}})
		assert.False(t, a.Equals(b))

		b.HasDomains(domain.ServicesDomains{"web": {"shop.example.com"}})
		assert.True(t, a.Equals(b))
	})

	t.Run("should list every hostname sorted", func(t *testing.T) {
		d := domain.ServicesDomains{"web": {"www.example.com", "example.com"}, "api": {"api.example.com"}}

		assert.DeepEqual(t, []domain.Hostname{"api.example.com", "example.com", "www.example.com"}, d.All())
		assert.DeepEqual(t, []domain.Hostname{"www.example.com", "example.com"}, d.For("web"))
	})
}
