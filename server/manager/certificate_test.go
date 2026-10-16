package manager

import (
	"errors"
	"strings"
	"testing"
)

func TestACMEEnableChangesAddressAndLinks(t *testing.T) {
	m := newManager(t)
	if c, _ := m.Certificate(); c.Mode != "self-signed" || c.Status != "self-signed" || c.Certificate == nil {
		t.Fatalf("default = %+v", c)
	}
	if _, err := m.AddUser(NewUser{Name: "alice"}); err != nil {
		t.Fatal(err)
	}

	domain, mail := "VPN.Example.com", "me@example.com"
	c, err := m.EnableACME(ACMEPatch{Domain: &domain, Email: &mail})
	if err != nil {
		t.Fatal(err)
	}
	if c.Mode != "acme" || c.Status != "waiting" || c.Domain != "vpn.example.com" || c.Provider != "letsencrypt" {
		t.Fatalf("after enable = %+v", c)
	}
	st, _ := m.Store.Load()
	if st.Server.PublicHost != "vpn.example.com" || !st.Server.ACMEActive() {
		t.Fatalf("public host = %q", st.Server.PublicHost)
	}

	set, _ := m.Links("alice")
	if !strings.HasPrefix(set.Subscription, "https://vpn.example.com:") {
		t.Errorf("subscription = %s, want https on the domain", set.Subscription)
	}
	for _, l := range set.Links {
		if strings.Contains(l.URL, "insecure=1") {
			t.Errorf("%s link still asks clients to skip verification: %s", l.Protocol, l.URL)
		}
		if !strings.Contains(l.URL, "vpn.example.com") {
			t.Errorf("%s link does not name the domain: %s", l.Protocol, l.URL)
		}
	}

	// Back to self-signed: verification is skipped again (the fallback
	// certificate was reissued for the domain).
	c, err = m.UseSelfSigned()
	if err != nil || c.Mode != "self-signed" {
		t.Fatalf("self-signed = %+v err %v", c, err)
	}
	set, _ = m.Links("alice")
	if !strings.HasPrefix(set.Subscription, "http://") || !strings.Contains(set.Links[2].URL, "insecure=1") {
		t.Errorf("after disabling ACME: %s / %s", set.Subscription, set.Links[2].URL)
	}
}

func TestACMEValidation(t *testing.T) {
	m := newManager(t)
	str := func(s string) *string { return &s }
	for name, p := range map[string]ACMEPatch{
		"no domain":           {},
		"ip address":          {Domain: str("203.0.113.7")},
		"not a hostname":      {Domain: str("not a host")},
		"single label":        {Domain: str("localhost")},
		"bad email":           {Domain: str("vpn.example.com"), Email: str("nobody")},
		"unknown provider":    {Domain: str("vpn.example.com"), Provider: str("shady-ca")},
		"zerossl no email":    {Domain: str("vpn.example.com"), Provider: str("zerossl")},
		"cloudflare no token": {Domain: str("vpn.example.com"), DNSProvider: str("cloudflare")},
		"alidns half":         {Domain: str("vpn.example.com"), DNSProvider: str("alidns"), DNSToken: str("id")},
		"unknown dns":         {Domain: str("vpn.example.com"), DNSProvider: str("route66"), DNSToken: str("x")},
	} {
		if _, err := m.EnableACME(p); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if st, _ := m.Store.Load(); st.Server.ACME.Enabled || st.Server.PublicHost != "203.0.113.7" {
		t.Fatalf("a rejected request changed the server: %+v host %s", st.Server.ACME, st.Server.PublicHost)
	}
}

func TestACMEDNSSecretsAreNotExposed(t *testing.T) {
	m := newManager(t)
	str := func(s string) *string { return &s }
	c, err := m.EnableACME(ACMEPatch{Domain: str("vpn.example.com"), DNSProvider: str("cloudflare"), DNSToken: str("SECRET-TOKEN")})
	if err != nil || c.DNSProvider != "cloudflare" || !c.DNSTokenSet {
		t.Fatalf("cert = %+v err %v", c, err)
	}
	if raw := mustJSON(t, c); strings.Contains(raw, "SECRET-TOKEN") {
		t.Fatalf("the token leaked into the certificate info: %s", raw)
	}
	st, _ := m.Store.Load()
	if st.Server.ACME.DNSToken != "SECRET-TOKEN" {
		t.Fatal("the token was not stored")
	}
	// Switching the DNS provider off forgets its secrets.
	c, _ = m.EnableACME(ACMEPatch{DNSProvider: str("")})
	if st, _ = m.Store.Load(); st.Server.ACME.DNSToken != "" || c.DNSTokenSet {
		t.Fatalf("secrets kept after switching to HTTP challenges: %+v", st.Server.ACME)
	}
}
