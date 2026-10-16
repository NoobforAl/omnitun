package serverconf

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/server/store"
)

func acmeState(t *testing.T) (*store.State, *store.Store) {
	st, s := goldenState(t)
	st.Server.PublicHost = "vpn.example.com"
	st.Server.ACME = store.ACME{Enabled: true, Domain: "vpn.example.com", Email: "me@example.com"}
	return st, s
}

func TestBuildUsesACMEProviderInsteadOfCertificateFiles(t *testing.T) {
	st, s := acmeState(t)
	cfg := Build(st, s, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))

	providers, _ := cfg["certificate_providers"].([]any)
	if len(providers) != 1 {
		t.Fatalf("certificate_providers = %v", providers)
	}
	p := providers[0].(obj)
	if p["type"] != "acme" || p["tag"] != ACMETag || p["email"] != "me@example.com" || p["data_directory"] != ACMEDir(s) {
		t.Fatalf("provider = %v", p)
	}
	if d, _ := p["domain"].([]string); len(d) != 1 || d[0] != "vpn.example.com" {
		t.Fatalf("domain = %v", p["domain"])
	}
	if _, set := p["dns01_challenge"]; set {
		t.Fatal("DNS-01 configured without a DNS provider")
	}

	raw, _ := json.Marshal(cfg["inbounds"])
	text := string(raw)
	if strings.Contains(text, "certificate_path") || strings.Contains(text, "tls.crt") {
		t.Fatalf("an inbound still uses the certificate files: %s", text)
	}
	if n := strings.Count(text, `"certificate_provider":"acme"`); n != 5 {
		t.Fatalf("%d inbounds use the provider, want 5 (trojan, hysteria2, tuic, anytls, naive)", n)
	}
}

func TestACMEProviderOptions(t *testing.T) {
	st, s := acmeState(t)
	st.Server.ACME.Provider = "zerossl"
	st.Server.ACME.HTTPPort = 8080
	st.Server.ACME.DNSProvider, st.Server.ACME.DNSToken = "cloudflare", "cf-token"
	p := ACMEProvider(st, s)
	if p["provider"] != "zerossl" || p["alternative_http_port"] != 8080 {
		t.Fatalf("provider = %v", p)
	}
	if dns := p["dns01_challenge"].(obj); dns["provider"] != "cloudflare" || dns["api_token"] != "cf-token" {
		t.Fatalf("dns01 = %v", dns)
	}
	st.Server.ACME.DNSProvider, st.Server.ACME.DNSToken, st.Server.ACME.DNSSecret = "alidns", "id", "secret"
	if dns := ACMEProvider(st, s)["dns01_challenge"].(obj); dns["access_key_id"] != "id" || dns["access_key_secret"] != "secret" {
		t.Fatalf("alidns = %v", dns)
	}
	st.Server.ACME.Provider = "letsencrypt"
	if _, set := ACMEProvider(st, s)["provider"]; set {
		t.Fatal("the default provider should not be written")
	}
}

// With only protocols that need no certificate there is nothing to issue.
func TestNoProviderWithoutCertificateProtocols(t *testing.T) {
	st, s := acmeState(t)
	for key, p := range st.Server.Protocols {
		p.Enabled = key == store.VLESSReality || key == store.Shadowsocks
	}
	if cfg := Build(st, s, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); cfg["certificate_providers"] != nil {
		t.Fatalf("certificate_providers = %v", cfg["certificate_providers"])
	}
}

func TestClientProfilesDoNotPinOrSkipVerificationWithACME(t *testing.T) {
	st, s := acmeState(t)
	profiles, err := Profiles(st, s, st.Users[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles {
		if p.TLS != nil && (p.TLS.Insecure || p.TLS.Certificate != "") {
			t.Errorf("%s: TLS = %+v, want normal verification", p.Name, p.TLS)
		}
	}
}
