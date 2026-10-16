package manager

import (
	"testing"

	"github.com/NoobforAl/omnitun/server/store"
)

func ruleSet(rules []PortRule) map[string]string {
	out := map[string]string{}
	for _, r := range rules {
		out[r.Proto+"/"+itoa(r.Port)] = r.Purpose
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}

func TestFirewallPorts(t *testing.T) {
	m := newManager(t)
	rules, err := m.FirewallPorts()
	if err != nil {
		t.Fatal(err)
	}
	got := ruleSet(rules)
	// Shadowsocks is tcp+udp; trojan (tcp) and hysteria2 (udp) share 8443.
	for _, want := range []string{"tcp/443", "tcp/8388", "udp/8388", "tcp/8443", "udp/8443", "udp/8444", "udp/51820", "udp/1194", "tcp/1194", "tcp/2096"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	if _, ok := got["tcp/2097"]; ok {
		t.Error("the web API on the loopback interface needs no open port")
	}
	if _, ok := got["tcp/80"]; ok {
		t.Error("port 80 opened without ACME")
	}

	// Disabled protocols need nothing; a public API and ACME add ports.
	off := false
	m.SetProtocol(store.TUIC, ProtocolPatch{Enabled: &off})
	on, host, domain := true, "0.0.0.0", "vpn.example.com"
	m.SetAPI(APIPatch{Enabled: &on, Listen: &host})
	if _, err := m.EnableACME(ACMEPatch{Domain: &domain}); err != nil {
		t.Fatal(err)
	}
	rules, _ = m.FirewallPorts()
	got = ruleSet(rules)
	if _, ok := got["udp/8444"]; ok {
		t.Error("a disabled protocol is still listed")
	}
	if got["tcp/80"] != "acme-http-challenge" || got["tcp/2097"] != "web-api" {
		t.Errorf("rules = %v", got)
	}

	// DNS-01 needs no open port; the HTTP challenge port can be moved.
	cf, token := "cloudflare", "t"
	m.EnableACME(ACMEPatch{DNSProvider: &cf, DNSToken: &token})
	if rules, _ = m.FirewallPorts(); ruleSet(rules)["tcp/80"] != "" {
		t.Error("DNS-01 still opens port 80")
	}
	empty, port := "", 8080
	m.EnableACME(ACMEPatch{DNSProvider: &empty, HTTPPort: &port})
	if rules, _ = m.FirewallPorts(); ruleSet(rules)["tcp/8080"] != "acme-http-challenge" {
		t.Errorf("custom challenge port: %v", ruleSet(rules))
	}

	// Two purposes on one port are merged.
	m2 := newManager(t)
	p := 8443
	st, _ := m2.Store.Load()
	_ = st
	m2.SetProtocol(store.AnyTLS, ProtocolPatch{Port: &p}) // conflicts with trojan: refused, ports stay unique
	if rules, _ := m2.FirewallPorts(); len(ruleSet(rules)) != len(rules) {
		t.Fatalf("duplicate rules: %v", rules)
	}
}
