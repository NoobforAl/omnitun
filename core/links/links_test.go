package links

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
)

// same compares two profiles by what they connect with (not Source or ID).
func same(t *testing.T, a, b *profile.Profile) {
	t.Helper()
	norm := func(p *profile.Profile) string {
		c := *p
		c.Source, c.ID = "", ""
		raw, _ := json.Marshal(c)
		return string(raw)
	}
	if norm(a) != norm(b) {
		t.Errorf("profiles differ:\n  want %s\n  got  %s", norm(a), norm(b))
	}
}

// Every link format the importer reads must survive parse -> export -> parse.
func TestExportRoundTrip(t *testing.T) {
	const (
		uuid = "b831381d-6324-4d53-ad4f-8cda48b30811"
	)
	for name, link := range map[string]string{
		"vless reality":  "vless://" + uuid + "@203.0.113.7:443?type=tcp&security=reality&flow=xtls-rprx-vision&sni=www.microsoft.com&fp=chrome&pbk=PUBKEY&sid=ab12#Lab%20VLESS",
		"vless ws tls":   "vless://" + uuid + "@example.com:443?type=ws&security=tls&sni=example.com&host=example.com&path=%2Fws&alpn=h2%2Chttp%2F1.1#ws",
		"vless grpc":     "vless://" + uuid + "@example.com:443?type=grpc&security=tls&sni=example.com&serviceName=svc#grpc",
		"vless insecure": "vless://" + uuid + "@203.0.113.7:443?type=tcp&security=tls&sni=203.0.113.7&allowInsecure=1#x",
		"vless plain":    "vless://" + uuid + "@203.0.113.7:80?type=tcp&security=none#plain",
		"trojan":         "trojan://pass%40word@203.0.113.7:8443?security=tls&sni=203.0.113.7&insecure=1&allowInsecure=1#Lab%20Trojan",
		"trojan ws":      "trojan://secret@example.com:443?type=ws&security=tls&sni=example.com&path=%2Ft#t",
		"hysteria2":      "hysteria2://pw@203.0.113.7:8443?sni=203.0.113.7&insecure=1&obfs=salamander&obfs-password=op#hy2",
		"hysteria2 user": "hysteria2://user:pw@203.0.113.7:8443?sni=x#hy2u",
		"hysteria v1":    "hysteria://203.0.113.7:8443?protocol=udp&auth=secret&peer=x&insecure=1&upmbps=10&downmbps=50&alpn=hysteria&obfsParam=o#hy1",
		"tuic":           "tuic://" + uuid + ":pw@203.0.113.7:8444?congestion_control=bbr&alpn=h3&sni=203.0.113.7&allow_insecure=1#tuic",
		"anytls":         "anytls://pw@203.0.113.7:8445/?sni=203.0.113.7&insecure=1&fp=chrome#anytls",
		"ss 2022":        "ss://MjAyMi1ibGFrZTMtYWVzLTEyOC1nY206c2VydmVya2V5OnVzZXJrZXk@203.0.113.7:8388#ss",
		"ss shadow-tls":  "ss://MjAyMi1ibGFrZTMtYWVzLTEyOC1nY206YTpi@203.0.113.7:8446?plugin=shadow-tls%3Bhost%3Dwww.microsoft.com%3Bpassword%3Dpw%3Bversion%3D3#stls",
		"ssh":            "ssh://root:pw@203.0.113.7:22?hk=ssh-ed25519+AAAA#ssh",
		"naive":          "naive+https://user:pw@203.0.113.7:8447/?sni=203.0.113.7#naive",
		"naive quic":     "naive+quic://user:pw@203.0.113.7:443/#nq",
		"sstp":           "sstp://alice:pw@203.0.113.7:443#sstp",
		"sstp verified":  "sstp://alice:pw@203.0.113.7:443?insecure=0#sstp2",
		"ikev2 psk":      "ikev2://203.0.113.7:4500?auth=psk&psk=secret&remote_id=server.test&tcp=1#ikev2",
		"ikev2 eap":      "ikev2://bob:eap-secret@203.0.113.7:500?auth=eap#eap",
		"wireguard":      "wireguard://UFJJVkFURUtFWQ%3D%3D@203.0.113.7:51820?publickey=UEVFUlBVQg%3D%3D&address=10.30.0.2%2F32&allowedips=0.0.0.0%2F0%2C%3A%3A%2F0&dns=1.1.1.1&mtu=1380&keepalive=25&reserved=1%2C2%2C3#wg",
		"tor":            "tor://",
		"tor snowflake":  "tor://snowflake",
		"vmess":          "",
	} {
		t.Run(name, func(t *testing.T) {
			var p *profile.Profile
			var err error
			if name == "vmess" {
				p = &profile.Profile{Name: "vm", Protocol: profile.VMess, Server: "example.com", Port: 443, UUID: uuid, Method: "auto",
					TLS: &profile.TLS{Enabled: true, ServerName: "example.com"}, Transport: &profile.Transport{Type: "ws", Path: "/p", Host: "example.com"}}
			} else if p, err = importer.ParseLink(link); err != nil {
				t.Fatalf("parse %q: %v", link, err)
			}
			out, err := Export(p)
			if err != nil {
				t.Fatal(err)
			}
			back, err := importer.ParseLink(out)
			if err != nil {
				t.Fatalf("exported link does not parse: %v\n%s", err, out)
			}
			same(t, p, back)
		})
	}
}

func TestExportRejectsConfigFileProtocols(t *testing.T) {
	for _, proto := range []string{profile.AmneziaWG, profile.OpenVPN} {
		if _, err := Export(&profile.Profile{Protocol: proto}); !errors.Is(err, ErrNoLink) {
			t.Errorf("%s: err = %v", proto, err)
		}
	}
}
