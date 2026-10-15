package importer

import (
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/profile"
)

const clashConfig = `
mixed-port: 7890
proxies:
  - name: "reality"
    type: vless
    server: r.example.com
    port: 443
    uuid: b831381d-6324-4d53-ad4f-8cda48b30811
    network: tcp
    tls: true
    flow: xtls-rprx-vision
    servername: www.microsoft.com
    client-fingerprint: chrome
    reality-opts:
      public-key: jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0
      short-id: 12ab
  - name: vmess-ws
    type: vmess
    server: v.example.com
    port: 443
    uuid: b831381d-6324-4d53-ad4f-8cda48b30811
    alterId: 0
    cipher: auto
    tls: true
    network: ws
    ws-opts:
      path: /ray
      headers:
        Host: cdn.example.com
  - {name: trojan-grpc, type: trojan, server: t.example.com, port: 443, password: pw, sni: t.example.com, network: grpc, grpc-opts: {grpc-service-name: gun}}
  - name: ss-obfs
    type: ss
    server: s.example.com
    port: 8388
    cipher: chacha20-ietf-poly1305
    password: pw
    plugin: obfs
    plugin-opts: {mode: http, host: bing.com}
  - {name: hy2, type: hysteria2, server: h.example.com, port: 443, password: pw, sni: h.example.com, skip-cert-verify: true, obfs: salamander, obfs-password: ob}
  - {name: tuic, type: tuic, server: u.example.com, port: 443, uuid: b831381d-6324-4d53-ad4f-8cda48b30811, password: pw, congestion-controller: bbr, alpn: [h3]}
  - name: wg
    type: wireguard
    server: w.example.com
    port: 51820
    ip: 10.0.0.2
    private-key: yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
    public-key: xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
    reserved: [1, 2, 3]
  - {name: snell, type: snell, server: x.example.com, port: 1, psk: x}
proxy-groups:
  - {name: auto, type: url-test, proxies: [reality, hy2]}
`

func TestImportClash(t *testing.T) {
	res, err := Import(clashConfig)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profiles) != 7 || len(res.Errors) != 1 {
		t.Fatalf("profiles=%d errors=%v", len(res.Profiles), res.Errors)
	}
	by := map[string]*profile.Profile{}
	for _, p := range res.Profiles {
		by[p.Name] = p
	}
	if r := by["reality"]; r.TLS == nil || r.TLS.Reality == nil || r.TLS.Reality.ShortID != "12ab" || r.Flow != "xtls-rprx-vision" {
		t.Fatalf("reality: %+v %+v", r, r.TLS)
	}
	if v := by["vmess-ws"]; v.Transport == nil || v.Transport.Path != "/ray" || v.Transport.Host != "cdn.example.com" {
		t.Fatalf("vmess ws: %+v", v.Transport)
	}
	if tr := by["trojan-grpc"]; tr.Transport == nil || tr.Transport.ServiceName != "gun" || !tr.TLS.Enabled {
		t.Fatalf("trojan grpc: %+v", tr)
	}
	if ss := by["ss-obfs"]; ss.Plugin != "obfs-local" || ss.PluginOpts != "obfs=http;obfs-host=bing.com" {
		t.Fatalf("ss plugin: %q %q", ss.Plugin, ss.PluginOpts)
	}
	if hy := by["hy2"]; hy.Hysteria2 == nil || hy.Hysteria2.ObfsPassword != "ob" || !hy.TLS.Insecure {
		t.Fatalf("hy2: %+v", hy)
	}
	if wg := by["wg"]; wg.WireGuard == nil || len(wg.WireGuard.Reserved) != 3 || wg.WireGuard.Address[0] != "10.0.0.2" {
		t.Fatalf("wg: %+v", wg.WireGuard)
	}
	// Re-importing gives the same IDs (so subscription refreshes keep the selection).
	again, _ := Import(clashConfig)
	if again.Profiles[0].ID != res.Profiles[0].ID {
		t.Fatal("IDs are not stable across imports")
	}
}

const singBoxConfig = `{
  "outbounds": [
    {"type": "selector", "tag": "proxy", "outbounds": ["a", "b"]},
    {"type": "vless", "tag": "a", "server": "r.example.com", "server_port": 443,
     "uuid": "b831381d-6324-4d53-ad4f-8cda48b30811", "flow": "xtls-rprx-vision",
     "tls": {"enabled": true, "server_name": "www.microsoft.com", "utls": {"enabled": true, "fingerprint": "chrome"},
             "reality": {"enabled": true, "public_key": "jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0", "short_id": "12ab"}}},
    {"type": "shadowsocks", "tag": "b", "server": "s.example.com", "server_port": 8388, "method": "2022-blake3-aes-128-gcm", "password": "a:b"},
    {"type": "trojan", "tag": "c", "server": "t.example.com", "server_port": 443, "password": "pw",
     "tls": {"enabled": true}, "transport": {"type": "ws", "path": "/ws", "headers": {"Host": "cdn.example.com"}}},
    {"type": "direct", "tag": "direct"}
  ],
  "endpoints": [
    {"type": "wireguard", "tag": "wg", "address": ["10.0.0.2/32"], "private_key": "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=",
     "peers": [{"address": "w.example.com", "port": 51820, "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=", "allowed_ips": ["0.0.0.0/0"]}]}
  ]
}`

func TestImportSingBoxJSON(t *testing.T) {
	res, err := Import(singBoxConfig)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profiles) != 4 {
		t.Fatalf("profiles=%d errors=%v", len(res.Profiles), res.Errors)
	}
	a := res.Profiles[0]
	if a.Protocol != profile.VLESS || a.TLS.Reality == nil || a.TLS.Fingerprint != "chrome" {
		t.Fatalf("vless: %+v %+v", a, a.TLS)
	}
	if c := res.Profiles[2]; c.Transport == nil || c.Transport.Host != "cdn.example.com" {
		t.Fatalf("trojan ws: %+v", c.Transport)
	}
	if wg := res.Profiles[3]; wg.Protocol != profile.WireGuard || wg.Server != "w.example.com" || wg.Port != 51820 {
		t.Fatalf("wg: %+v", wg)
	}
}

func TestParseOVPN(t *testing.T) {
	base := "client\ndev tun\nproto tcp-client\nremote vpn1.example.com 443\nremote vpn2.example.com 1194 udp\nremote-random\n" +
		"cipher AES-256-CBC\ndata-ciphers AES-256-GCM:CHACHA20-POLY1305\nauth SHA256\ncomp-lzo\nverify-x509-name server name\n" +
		"remote-cert-tls server\nresolv-retry infinite\nauth-user-pass\n<ca>\n-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n</ca>\n" +
		"key-direction 1\n<tls-auth>\n-----BEGIN OpenVPN Static key V1-----\nab\n-----END OpenVPN Static key V1-----\n</tls-auth>\n"
	res, err := Import(base)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Profiles[0]
	o := p.OpenVPN
	if p.Protocol != profile.OpenVPN || p.Server != "vpn1.example.com" || p.Port != 443 || o.Network != "tcp" {
		t.Fatalf("first remote: %+v %+v", p, o)
	}
	if len(o.Remotes) != 1 || o.Remotes[0].Network != "udp" || o.Remotes[0].Port != 1194 || !o.RemoteRandom {
		t.Fatalf("remotes: %+v", o.Remotes)
	}
	if !o.NeedsAuth || o.ControlWrap != "tls-auth" || o.KeyDirection != "1" || o.CompressionLZO != "adaptive" ||
		o.VerifyX509Type != "name" || len(o.DataCiphers) != 2 || o.Cipher != "AES-256-CBC" {
		t.Fatalf("options: %+v", o)
	}

	for name, bad := range map[string]string{
		"file reference": strings.Replace(base, "<ca>", "ca ca.crt\n<x>", 1),
		"tap":            strings.Replace(base, "dev tun", "dev tap", 1),
		"no ca":          strings.Replace(strings.Replace(base, "<ca>", "<cax>", 1), "</ca>", "</cax>", 1),
		"auth file":      strings.Replace(base, "auth-user-pass\n", "auth-user-pass creds.txt\n", 1),
	} {
		if _, err := Import(bad); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	// Inline credentials.
	withCreds := base + "<auth-user-pass>\nalice\nsecret\n</auth-user-pass>\n"
	res, err = Import(withCreds)
	if err != nil {
		t.Fatal(err)
	}
	if p := res.Profiles[0]; p.Username != "alice" || p.Password != "secret" {
		t.Fatalf("inline creds: %q %q", p.Username, p.Password)
	}
}

func TestImportClashNewTypes(t *testing.T) {
	res, err := Import(`proxies:
  - {name: hy1, type: hysteria, server: h.example.com, port: 443, auth-str: pw, obfs: ob, up: "20 Mbps", down: 80, sni: h.example.com, skip-cert-verify: true}
  - {name: any, type: anytls, server: a.example.com, port: 443, password: pw, sni: a.example.com}
  - {name: ssh, type: ssh, server: s.example.com, port: 22, username: u, password: pw, host-key: ["ssh-ed25519 AAAA"]}
  - {name: stls, type: ss, server: t.example.com, port: 443, cipher: 2022-blake3-aes-128-gcm, password: "AAAAAAAAAAAAAAAAAAAAAA==", plugin: shadow-tls, plugin-opts: {host: www.microsoft.com, password: sp, version: 3}}
`)
	if err != nil || len(res.Profiles) != 4 {
		t.Fatalf("err=%v profiles=%d errors=%v", err, len(res.Profiles), res.Errors)
	}
	hy, any, ssh, stls := res.Profiles[0], res.Profiles[1], res.Profiles[2], res.Profiles[3]
	if hy.Protocol != profile.Hysteria || hy.Password != "pw" || hy.Hysteria2.UpMbps != 20 || hy.Hysteria2.DownMbps != 80 || hy.Hysteria2.Obfs != "ob" || !hy.TLS.Insecure {
		t.Fatalf("hysteria: %+v %+v", hy, hy.Hysteria2)
	}
	if any.Protocol != profile.AnyTLS || any.TLS.ServerName != "a.example.com" {
		t.Fatalf("anytls: %+v", any)
	}
	if ssh.Protocol != profile.SSH || ssh.Username != "u" || ssh.SSH == nil || len(ssh.SSH.HostKey) != 1 {
		t.Fatalf("ssh: %+v", ssh)
	}
	if stls.Plugin != "shadow-tls" || stls.PluginOpts != "host=www.microsoft.com;password=sp;version=3" {
		t.Fatalf("shadow-tls: %q %q", stls.Plugin, stls.PluginOpts)
	}
}
