package singbox

import (
	"testing"

	"github.com/NoobforAl/omnitun/core/profile"
)

func TestTorProfileRoutesIntoSocks(t *testing.T) {
	p := &profile.Profile{Name: "Tor", Protocol: profile.Tor, Tor: &profile.TorOpts{}}
	p.EnsureID()
	if _, skipped, err := Build([]*profile.Profile{p}, Options{}); err == nil || len(skipped) != 1 {
		t.Fatalf("without a Tor client the profile should be skipped: err=%v skipped=%v", err, skipped)
	}
	cfg, skipped, err := Build([]*profile.Profile{p}, Options{TorSocks: "127.0.0.1:9150", Selected: p.ID})
	if err != nil || len(skipped) != 0 {
		t.Fatalf("err=%v skipped=%v", err, skipped)
	}
	var found obj
	for _, o := range cfg["outbounds"].([]any) {
		if m, ok := o.(obj); ok && m["tag"] == p.ID {
			found = m
		}
	}
	if found["type"] != "socks" || found["server"] != "127.0.0.1" || found["server_port"] != 9150 {
		t.Fatalf("outbound = %v", found)
	}
}

func TestTorForcesHTTPSDNS(t *testing.T) {
	o := Options{TorSocks: "127.0.0.1:1", DNS: "tls://1.1.1.1"}
	o.defaults()
	if o.DNS != "https://1.1.1.1/dns-query" {
		t.Fatalf("DNS = %q", o.DNS)
	}
	o = Options{DNS: "tls://1.1.1.1"}
	o.defaults()
	if o.DNS != "tls://1.1.1.1" {
		t.Fatalf("DNS changed without Tor: %q", o.DNS)
	}
}

func TestTorTunUsesGvisorStack(t *testing.T) {
	p := &profile.Profile{Name: "Tor", Protocol: profile.Tor, Tor: &profile.TorOpts{}}
	p.EnsureID()
	cfg, _, err := Build([]*profile.Profile{p}, Options{Mode: ModeTUN, TorSocks: "127.0.0.1:9150", Selected: p.ID})
	if err != nil {
		t.Fatal(err)
	}
	tun := cfg["inbounds"].([]any)[0].(obj)
	if tun["stack"] != "gvisor" {
		t.Fatalf("stack = %v", tun["stack"])
	}
}
