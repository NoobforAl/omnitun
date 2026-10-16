package protocols

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/links"
	"github.com/NoobforAl/omnitun/core/singbox"
)

func testEnv() Env {
	return Env{
		Name: "Lab", Host: "203.0.113.7", CertPath: "tls.crt", KeyPath: "tls.key", CertPEM: "CERT\n", SelfSigned: true,
		Reality:     Reality{PrivateKey: "priv", PublicKey: "pub", ShortID: "ab12", Handshake: "www.microsoft.com:443"},
		SSServerKey: "c2VydmVyLWtleS0xMjM0NQ==",
		OpenVPN:     OpenVPN{CA: "CA\n", ServerCert: "CERT", ServerKey: "KEY", TLSCrypt: "CRYPT\n"},
		WireGuard:   WireGuard{PrivateKey: "c2VydmVyLXByaXZhdGU=", PublicKey: "c2VydmVyLXB1YmxpYw==", Address: "10.30.0.1/16"},
	}
}

var alice = Account{
	Name: "alice", UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Password: "pw", SSKey: "YWxpY2Uta2V5LTEyMzQ1Ng==",
	WGPrivateKey: "YWxpY2UtcHJpdmF0ZQ==", WGPublicKey: "YWxpY2UtcHVibGlj", WGAddress: "10.30.0.2/32",
}

func TestRegistry(t *testing.T) {
	keys := Keys()
	if len(keys) == 0 || len(slices.Compact(slices.Sorted(slices.Values(keys)))) != len(keys) {
		t.Fatalf("keys: %v", keys)
	}
	for _, s := range Specs() {
		if s.Name == "" || s.Label == "" || s.Description == "" || s.DefaultPort <= 0 || s.Profile == "" {
			t.Errorf("incomplete spec: %+v", s)
		}
		if !slices.Contains([]string{"tcp", "udp", "tcp+udp"}, s.Transport) {
			t.Errorf("%s: transport %q", s.Key, s.Transport)
		}
		if p, ok := Get(s.Key); !ok || p.Spec().Key != s.Key {
			t.Errorf("Get(%q) failed", s.Key)
		}
	}
	if _, ok := Get("nope"); ok {
		t.Error("Get accepted an unknown key")
	}
	if DefaultPorts()["vless-reality"] != 443 {
		t.Errorf("default ports: %v", DefaultPorts())
	}
}

// Each protocol's server side and client side must fit together: the server
// config is valid JSON with the user in it, the profile builds a sing-box
// outbound, and share-link protocols export a link that parses back to it.
func TestEveryProtocolIsCoherent(t *testing.T) {
	env := testEnv()
	for _, proto := range All() {
		s := proto.Spec()
		t.Run(s.Key, func(t *testing.T) {
			cfg := proto.Inbounds(env, s.DefaultPort, []Account{alice})
			if len(cfg.Inbounds)+len(cfg.Endpoints) == 0 {
				t.Fatal("no inbound")
			}
			raw, err := json.Marshal(cfg)
			// Users appear by name, or (WireGuard peers) by public key.
			if err != nil || !(strings.Contains(string(raw), "alice") || strings.Contains(string(raw), alice.WGPublicKey)) {
				t.Fatalf("config has no user: %v %s", err, raw)
			}

			p, err := proto.Profile(env, s.DefaultPort, alice)
			if err != nil {
				t.Fatal(err)
			}
			if p.Protocol != s.Profile || p.Server != env.Host || !strings.HasPrefix(p.Name, env.Name) {
				t.Fatalf("profile = %+v", p)
			}
			if _, _, _, err := singbox.Outbound(p, "t"); err != nil {
				t.Fatalf("not usable as a sing-box outbound: %v", err)
			}

			link, err := links.Export(p)
			if !s.ShareLink {
				if err == nil {
					t.Fatalf("a protocol without a share link exported %q", link)
				}
				if _, ok := proto.(FileProtocol); !ok {
					t.Fatal("no share link and no config file")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			back, err := importer.ParseLink(link)
			if err != nil {
				t.Fatalf("link %q: %v", link, err)
			}
			if back.Server != p.Server || back.Port != p.Port || back.Protocol != p.Protocol {
				t.Fatalf("link round trip: %+v vs %+v", back, p)
			}
		})
	}
}

func TestShadowTLSInnerInboundIsNotListening(t *testing.T) {
	p, _ := Get("shadowtls")
	cfg := p.Inbounds(testEnv(), 8446, []Account{alice})
	if len(cfg.Inbounds) != 2 {
		t.Fatalf("inbounds = %d", len(cfg.Inbounds))
	}
	inner, outer := cfg.Inbounds[0], cfg.Inbounds[1]
	if _, listens := inner["listen_port"]; listens || outer["detour"] != inner["tag"] {
		t.Fatalf("inner = %v outer = %v", inner, outer)
	}
}

func TestSplitHostPort(t *testing.T) {
	for in, want := range map[string][2]any{
		"example.com:8443": {"example.com", 8443},
		"example.com":      {"example.com", 443},
		"example.com:x":    {"example.com", 443},
	} {
		h, p := SplitHostPort(in, 443)
		if h != want[0] || p != want[1] {
			t.Errorf("%q = %s %d", in, h, p)
		}
	}
}

// A secret with a line break or a semicolon must not reach a text config.
func TestUnsafeSecretsAreRefused(t *testing.T) {
	env := testEnv()
	ovpn, _ := Get("openvpn-udp")
	stls, _ := Get("shadowtls")
	for _, bad := range []string{"pw\nremote evil.example 1194", "pw\r\nup /bin/sh", "name\nx"} {
		acct := Account{Name: "alice", Password: bad, SSKey: alice.SSKey, UUID: alice.UUID}
		if _, err := ovpn.Profile(env, 1194, acct); err == nil {
			t.Errorf("openvpn accepted password %q", bad)
		}
		if f, ok := ovpn.(FileProtocol); ok {
			if _, _, err := f.File(env, 1194, acct); err == nil {
				t.Errorf("the .ovpn writer accepted password %q", bad)
			}
		}
		if _, err := stls.Profile(env, 8446, acct); err == nil {
			t.Errorf("shadowtls accepted password %q", bad)
		}
	}
	if _, err := stls.Profile(env, 8446, Account{Name: "a", Password: "pw;host=evil", SSKey: alice.SSKey}); err == nil {
		t.Error("shadowtls accepted a semicolon in the password")
	}
	if _, err := ovpn.Profile(env, 1194, alice); err != nil {
		t.Errorf("a normal password failed: %v", err)
	}
}
