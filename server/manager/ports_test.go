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

func TestHostProtocolPorts(t *testing.T) {
	m := newManager(t)
	on := true
	m.SetProtocol(store.IKEv2, ProtocolPatch{Enabled: &on})
	m.SetProtocol(store.SSTP, ProtocolPatch{Enabled: &on})
	rules, _ := m.FirewallPorts()
	got := ruleSet(rules)
	// IKEv2: IKE on UDP 500, NAT-T on 4500 over UDP, and IKE/ESP over TCP 4500.
	for _, want := range []string{"udp/500", "udp/4500", "tcp/4500", "tcp/8448"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	// A protocol on UDP 500 or TCP 4500 would clash with IKEv2.
	port := 4500
	if _, err := m.SetProtocol(store.AnyTLS, ProtocolPatch{Port: &port}); err == nil {
		t.Fatal("AnyTLS on TCP 4500 accepted while IKEv2 uses it")
	}
	// SSTP and IKEv2 are off on a new server.
	fresh := newManager(t)
	protos, _ := fresh.Protocols()
	for _, p := range protos {
		if p.Host && p.Enabled {
			t.Errorf("%s is on by default", p.Key)
		}
		if (p.Key == store.SSTP || p.Key == store.IKEv2) && !p.Host {
			t.Errorf("%s is not marked as a host protocol", p.Key)
		}
	}
}
