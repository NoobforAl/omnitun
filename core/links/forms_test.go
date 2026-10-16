package links

import (
	"errors"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

const (
	formUUID = "b831381d-6324-4d53-ad4f-8cda48b30811"
	wgPriv   = "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk="
	wgPub    = "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
)

// formSamples fill each protocol's editor the way a user would.
var formSamples = map[string]map[string]string{
	profile.VLESS: {"name": "v", "server": "vpn.example.com", "uuid": formUUID, "security": "reality", "flow": "xtls-rprx-vision",
		"tls.server_name": "www.microsoft.com", "tls.fingerprint": "chrome", "reality.public_key": wgPub, "reality.short_id": "ab12cd34"},
	profile.VMess:  {"server": "vpn.example.com", "uuid": formUUID, "security": "tls", "transport.type": "ws", "transport.path": "/ws", "transport.host": "cdn.example.com"},
	profile.Trojan: {"server": "vpn.example.com", "password": "pw", "tls.server_name": "vpn.example.com", "transport.type": "grpc", "transport.service_name": "svc"},
	profile.Shadowsocks: {"server": "vpn.example.com", "port": "8388", "method": "2022-blake3-aes-128-gcm",
		"password": "c2VydmVyLWtleS0xMjM0NQ=="},
	profile.Hysteria2: {"server": "vpn.example.com", "password": "pw", "hysteria.obfs": "salamander", "hysteria.obfs_password": "ob", "tls.server_name": "vpn.example.com"},
	profile.TUIC:      {"server": "vpn.example.com", "uuid": formUUID, "password": "pw", "tuic.congestion_control": "bbr", "tuic.udp_relay_mode": "native"},
	profile.AnyTLS:    {"server": "vpn.example.com", "password": "pw", "tls.server_name": "vpn.example.com"},
	profile.Naive:     {"server": "vpn.example.com", "username": "u", "password": "p"},
	profile.SSH:       {"server": "vpn.example.com", "username": "u", "password": "p"},
	profile.WireGuard: {"server": "203.0.113.5", "private_key": wgPriv, "peer_public_key": wgPub, "address": "10.8.0.2/32",
		"dns": "1.1.1.1, 8.8.8.8", "mtu": "1380", "keepalive": "25"},
	profile.AmneziaWG: {"server": "203.0.113.5", "port": "51821", "private_key": wgPriv, "peer_public_key": wgPub, "address": "10.31.0.2/32",
		"amnezia.jc": "4", "amnezia.jmin": "40", "amnezia.jmax": "70", "amnezia.s1": "15", "amnezia.s2": "20",
		"amnezia.h1": "11", "amnezia.h2": "12", "amnezia.h3": "13", "amnezia.h4": "14", "amnezia.i1": "<b 0xc0ff><r 16>"},
	profile.SSTP:  {"server": "vpn.example.com", "username": "alice", "password": "pw", "tls.insecure": "true"},
	profile.IKEv2: {"server": "vpn.example.com", "username": "alice", "password": "pw", "ikev2.tcp": "true", "ikev2.remote_id": "vpn.example.com"},
	profile.Tor:   {"tor.builtin": "snowflake"},
	profile.OpenVPN: {"server": "vpn.example.com", "username": "u", "password": "p", "openvpn.network": "tcp",
		"openvpn.ca": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"},
	profile.Hysteria: {"server": "vpn.example.com", "password": "auth", "hysteria.obfs": "obfspw", "hysteria.up_mbps": "20", "hysteria.down_mbps": "100"},
}

// Every protocol with an editor has a sample, and every sample builds a valid
// profile whose editor values read back as they were typed.
func TestFormsBuildValidProfiles(t *testing.T) {
	for _, f := range profile.Forms() {
		sample, ok := formSamples[f.Protocol]
		if !ok {
			t.Errorf("%s has an editor but no test sample", f.Protocol)
			continue
		}
		t.Run(f.Protocol, func(t *testing.T) {
			p, err := profile.FromValues(f.Protocol, sample)
			if err != nil {
				t.Fatal(err)
			}
			if p.ID == "" || p.Name == "" {
				t.Fatalf("no id or name: %+v", p)
			}
			got := p.Values()
			for k, want := range sample {
				if got[k] != want {
					t.Errorf("%s: typed %q, read back %q", k, want, got[k])
				}
			}
			// Reading values must not change the profile.
			before, after := *p, *p
			_ = after.Values()
			if before.TLS != after.TLS || before.WireGuard != after.WireGuard || before.Hysteria2 != after.Hysteria2 {
				t.Error("Values changed the profile")
			}
		})
	}
}

// A profile made in the editor is a real profile: it exports as a share link
// (where the protocol has one) that imports back to the same profile, and the
// sing-box protocols produce an outbound.
func TestFormProfilesExportImportAndBuild(t *testing.T) {
	for protocol, sample := range formSamples {
		t.Run(protocol, func(t *testing.T) {
			p, err := profile.FromValues(protocol, sample)
			if err != nil {
				t.Fatal(err)
			}
			if link, err := Export(p); err == nil {
				res, err := importer.Import(link)
				if err != nil || len(res.Profiles) != 1 {
					t.Fatalf("link %q did not import: %v", link, err)
				}
				// Compare through the editor: same values, whatever the link drops.
				want, got := p.Values(), res.Profiles[0].Values()
				for k, v := range want {
					if k == "name" || v == "" || v == "false" || got[k] == v { // links add their own defaults (ALPN, SNI)
						continue
					}
					t.Errorf("%s: %q became %q through %s", k, v, got[k], link)
				}
			} else if !errors.Is(err, ErrNoLink) {
				t.Fatalf("link: %v", err)
			}
			switch protocol {
			case profile.AmneziaWG, profile.SSTP, profile.IKEv2, profile.Tor:
				return // not sing-box outbounds
			}
			if _, _, _, err := singbox.Outbound(p, "t"); err != nil {
				t.Fatalf("not a usable outbound: %v", err)
			}
		})
	}
}

func TestFormErrorsAreClear(t *testing.T) {
	cases := []struct {
		protocol string
		values   map[string]string
		want     string
	}{
		{profile.VLESS, map[string]string{"server": "x"}, "UUID is required"},
		{profile.VLESS, map[string]string{"server": "x", "uuid": formUUID, "security": "reality"}, "reality"},
		{profile.Trojan, map[string]string{"password": "p"}, "Server is required"},
		{profile.Trojan, map[string]string{"server": "x", "password": "p", "colour": "red"}, `no setting "colour"`},
		{profile.Trojan, map[string]string{"server": "x", "password": "p", "port": "many"}, "whole number"},
		{profile.Trojan, map[string]string{"server": "x", "password": "p", "tls.fingerprint": "netscape"}, "not one of"},
		{profile.Trojan, map[string]string{"server": "x", "password": "p", "port": "70000"}, "invalid port"},
		{profile.WireGuard, map[string]string{"server": "x", "private_key": wgPriv, "peer_public_key": wgPub}, "Your tunnel address"},
		{profile.VMess, map[string]string{"server": "x", "uuid": formUUID, "security": "reality"}, "not one of"},
	}
	for _, c := range cases {
		_, err := profile.FromValues(c.protocol, c.values)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %v: error %v, want it to mention %q", c.protocol, c.values, err, c.want)
		}
	}
	if _, err := profile.New("carrier-pigeon"); err == nil {
		t.Error("an editor for an unknown protocol")
	}
}

// Editing an imported profile keeps what the form does not show.
func TestApplyEditsAnImportedProfile(t *testing.T) {
	res, err := importer.Import("trojan://secret@vpn.example.com:443?sni=a.example.com#old")
	if err != nil {
		t.Fatal(err)
	}
	p := res.Profiles[0]
	if err := p.Apply(map[string]string{"name": "new", "tls.server_name": "b.example.com", "port": "8443"}); err != nil {
		t.Fatal(err)
	}
	if p.Name != "new" || p.Port != 8443 || p.TLS.ServerName != "b.example.com" || p.Password != "secret" {
		t.Fatalf("edited profile: %+v", p)
	}
	if err := p.Apply(map[string]string{"tls.server_name": ""}); err != nil || p.TLS.ServerName != "" {
		t.Fatalf("clearing a field: %v %+v", err, p.TLS)
	}
}
