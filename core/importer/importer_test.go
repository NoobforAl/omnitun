package importer

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/profile"
)

const (
	testUUID = "b831381d-6324-4d53-ad4f-8cda48b30811"
	testKey  = "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk="
	testPub  = "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
)

func TestParseVLESSReality(t *testing.T) {
	link := "vless://" + testUUID + "@example.com:443?security=reality&sni=www.microsoft.com&fp=chrome&pbk=abc&sid=12ab&flow=xtls-rprx-vision&type=tcp#My%20Server"
	p, err := ParseLink(link)
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != profile.VLESS || p.Server != "example.com" || p.Port != 443 || p.UUID != testUUID {
		t.Fatalf("unexpected profile: %+v", p)
	}
	if p.Name != "My Server" || p.Flow != "xtls-rprx-vision" {
		t.Fatalf("name/flow: %q %q", p.Name, p.Flow)
	}
	if p.TLS == nil || p.TLS.Reality == nil || p.TLS.Reality.PublicKey != "abc" || p.TLS.Reality.ShortID != "12ab" {
		t.Fatalf("reality not parsed: %+v", p.TLS)
	}
	if p.Transport != nil {
		t.Fatalf("tcp should have no transport, got %+v", p.Transport)
	}
	if p.ID == "" {
		t.Fatal("ID not set")
	}
}

func TestParseVLESSWebSocket(t *testing.T) {
	p, err := ParseLink("vless://" + testUUID + "@1.2.3.4:8443?security=tls&type=ws&path=%2Fws&host=cdn.example.com&sni=cdn.example.com#ws")
	if err != nil {
		t.Fatal(err)
	}
	if p.Transport == nil || p.Transport.Type != "ws" || p.Transport.Path != "/ws" || p.Transport.Host != "cdn.example.com" {
		t.Fatalf("transport: %+v", p.Transport)
	}
	if p.TLS == nil || p.TLS.Reality != nil || p.TLS.ServerName != "cdn.example.com" {
		t.Fatalf("tls: %+v", p.TLS)
	}
}

func TestParseVMess(t *testing.T) {
	body := `{"v":"2","ps":"vm","add":"vm.example.com","port":"443","id":"` + testUUID + `","aid":"0","scy":"auto","net":"ws","type":"none","host":"vm.example.com","path":"/v","tls":"tls","sni":""}`
	p, err := ParseLink("vmess://" + base64.StdEncoding.EncodeToString([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != profile.VMess || p.Port != 443 || p.Name != "vm" {
		t.Fatalf("unexpected: %+v", p)
	}
	if p.TLS == nil || p.TLS.ServerName != "vm.example.com" {
		t.Fatalf("tls: %+v", p.TLS)
	}
	if p.Transport == nil || p.Transport.Type != "ws" || p.Transport.Path != "/v" {
		t.Fatalf("transport: %+v", p.Transport)
	}
}

func TestParseTrojanDefaultsToTLS(t *testing.T) {
	p, err := ParseLink("trojan://secret@tr.example.com:443?sni=tr.example.com#tr")
	if err != nil {
		t.Fatal(err)
	}
	if p.Password != "secret" || p.TLS == nil || !p.TLS.Enabled {
		t.Fatalf("unexpected: %+v", p)
	}
}

func TestParseShadowsocks(t *testing.T) {
	userinfo := base64.RawURLEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:pass"))
	cases := map[string]string{
		"sip002":       "ss://" + userinfo + "@ss.example.com:8388#SS%201",
		"plain-sip002": "ss://2022-blake3-aes-128-gcm:pass@ss.example.com:8388#SS%201",
		"legacy":       "ss://" + base64.StdEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:pass@ss.example.com:8388")) + "#SS%201",
	}
	for name, link := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := ParseLink(link)
			if err != nil {
				t.Fatal(err)
			}
			if p.Server != "ss.example.com" || p.Port != 8388 || p.Password != "pass" || p.Name != "SS 1" {
				t.Fatalf("unexpected: %+v", p)
			}
			if p.Method == "" {
				t.Fatal("method missing")
			}
		})
	}
}

func TestParseHysteria2(t *testing.T) {
	p, err := ParseLink("hy2://pw@hy.example.com:443?sni=hy.example.com&obfs=salamander&obfs-password=ob&insecure=1#hy")
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != profile.Hysteria2 || p.Password != "pw" || !p.TLS.Insecure {
		t.Fatalf("unexpected: %+v", p)
	}
	if p.Hysteria2 == nil || p.Hysteria2.Obfs != "salamander" || p.Hysteria2.ObfsPassword != "ob" {
		t.Fatalf("obfs: %+v", p.Hysteria2)
	}
}

func TestParseTUIC(t *testing.T) {
	p, err := ParseLink("tuic://" + testUUID + ":pw@tu.example.com:443?congestion_control=bbr&alpn=h3&sni=tu.example.com#tu")
	if err != nil {
		t.Fatal(err)
	}
	if p.UUID != testUUID || p.Password != "pw" || p.TUIC.CongestionControl != "bbr" {
		t.Fatalf("unexpected: %+v", p)
	}
}

func TestParseWireGuardConf(t *testing.T) {
	conf := `[Interface]
PrivateKey = ` + testKey + `
Address = 10.0.0.2/32, fd00::2/128
DNS = 1.1.1.1
MTU = 1280

[Peer]
PublicKey = ` + testPub + `
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = wg.example.com:51820
PersistentKeepalive = 25
`
	res, err := Import(conf)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Profiles[0]
	if p.Protocol != profile.WireGuard || p.Server != "wg.example.com" || p.Port != 51820 {
		t.Fatalf("unexpected: %+v", p)
	}
	if len(p.WireGuard.Address) != 2 || p.WireGuard.MTU != 1280 || p.WireGuard.Keepalive != 25 {
		t.Fatalf("wg: %+v", p.WireGuard)
	}
}

func TestParseAmneziaConf(t *testing.T) {
	conf := `[Interface]
PrivateKey = ` + testKey + `
Address = 10.8.0.2/32
Jc = 4
Jmin = 40
Jmax = 70
S1 = 0
S2 = 0
H1 = 1
H2 = 2
H3 = 3
H4 = 4

[Peer]
PublicKey = ` + testPub + `
Endpoint = 203.0.113.5:51820
`
	p, err := ParseWireGuardConf(conf)
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != profile.AmneziaWG || p.WireGuard.Amnezia["jc"] != "4" {
		t.Fatalf("unexpected: %+v", p.WireGuard)
	}
}

func TestImportBase64Subscription(t *testing.T) {
	lines := strings.Join([]string{
		"trojan://a@t1.example.com:443#one",
		"not a link",
		"hy2://b@h1.example.com:443#two",
	}, "\n")
	res, err := Import(base64.StdEncoding.EncodeToString([]byte(lines)))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profiles) != 2 || len(res.Errors) != 1 {
		t.Fatalf("profiles=%d errors=%v", len(res.Profiles), res.Errors)
	}
}

func TestImportRejectsGarbage(t *testing.T) {
	if _, err := Import("hello world"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseLink("vless://@:443"); err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestImportTorBridges(t *testing.T) {
	res, err := Import("Bridge obfs4 192.0.2.1:443 0123456789ABCDEF0123456789ABCDEF01234567 cert=abc iat-mode=0\n" +
		"webtunnel [2001:db8::1]:443 0123456789ABCDEF0123456789ABCDEF01234567 url=https://example.com/x ver=0.0.3")
	if err != nil || len(res.Profiles) != 1 {
		t.Fatalf("res=%v err=%v", res, err)
	}
	p := res.Profiles[0]
	if p.Protocol != "tor" || len(p.Tor.Bridges) != 2 || p.Tor.Bridges[0][:5] != "obfs4" {
		t.Fatalf("profile = %+v", p)
	}
	for link, kind := range map[string]string{"tor://": "", "tor://snowflake": "snowflake", "tor://obfs4/": "obfs4"} {
		p, err := ParseLink(link)
		if err != nil || p.Protocol != "tor" || p.Tor.Builtin != kind {
			t.Errorf("%s: %+v %v", link, p, err)
		}
	}
	if _, err := ParseLink("tor://nope"); err == nil {
		t.Error("tor://nope should fail")
	}
}
