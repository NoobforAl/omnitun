package importer

import (
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/core/profile"
)

const xrayClient = `{
  "log": {"loglevel": "warning"},
  "inbounds": [{"port": 10808, "protocol": "socks", "settings": {"udp": true}}],
  "outbounds": [
    {"tag": "reality", "protocol": "vless",
     "settings": {"vnext": [{"address": "203.0.113.7", "port": 443,
       "users": [{"id": "b831381d-6324-4d53-ad4f-8cda48b30811", "flow": "xtls-rprx-vision", "encryption": "none"}]}]},
     "streamSettings": {"network": "tcp", "security": "reality",
       "realitySettings": {"serverName": "www.microsoft.com", "fingerprint": "chrome", "publicKey": "PUB", "shortId": "ab12"}}},
    {"tag": "ws", "protocol": "vless",
     "settings": {"address": "example.com", "port": 443, "id": "b831381d-6324-4d53-ad4f-8cda48b30811"},
     "streamSettings": {"network": "ws", "security": "tls",
       "tlsSettings": {"serverName": "example.com", "alpn": ["h2", "http/1.1"], "allowInsecure": true},
       "wsSettings": {"path": "/ws", "headers": {"Host": "cdn.example.com"}}}},
    {"tag": "vm", "protocol": "vmess",
     "settings": {"vnext": [{"address": "vm.example.com", "port": 443,
       "users": [{"id": "b831381d-6324-4d53-ad4f-8cda48b30811", "alterId": 0, "security": "auto"}]}]},
     "streamSettings": {"network": "grpc", "security": "tls", "tlsSettings": {"serverName": "vm.example.com"},
       "grpcSettings": {"serviceName": "svc"}}},
    {"tag": "tj", "protocol": "trojan", "settings": {"servers": [{"address": "tj.example.com", "port": 443, "password": "pw"}]},
     "streamSettings": {"security": "tls", "tlsSettings": {"serverName": "tj.example.com"}}},
    {"tag": "ss", "protocol": "shadowsocks", "settings": {"servers": [{"address": "ss.example.com", "port": 8388, "method": "aes-256-gcm", "password": "secret"}]}},
    {"tag": "wg", "protocol": "wireguard", "settings": {"secretKey": "PRIV", "address": ["10.0.0.2/32"], "mtu": 1280,
       "peers": [{"publicKey": "PEERPUB", "endpoint": "wg.example.com:51820"}]}},
    {"tag": "direct", "protocol": "freedom"},
    {"tag": "block", "protocol": "blackhole"},
    {"tag": "xh", "protocol": "vless", "settings": {"address": "x.example.com", "port": 443, "id": "b831381d-6324-4d53-ad4f-8cda48b30811"},
     "streamSettings": {"network": "xhttp", "security": "tls"}}
  ]
}`

func TestImportXrayClient(t *testing.T) {
	res, err := Import(xrayClient)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]*profile.Profile{}
	for _, p := range res.Profiles {
		by[p.Name] = p
	}
	if len(res.Profiles) != 6 {
		t.Fatalf("got %d profiles, want 6: %v", len(res.Profiles), by)
	}
	if len(res.Errors) != 1 || !strings.Contains(res.Errors[0], "xhttp") {
		t.Fatalf("errors = %v, want the unsupported xhttp transport reported", res.Errors)
	}

	r := by["reality"]
	if r.Protocol != profile.VLESS || r.Server != "203.0.113.7" || r.Flow != "xtls-rprx-vision" ||
		r.TLS == nil || r.TLS.Reality == nil || r.TLS.Reality.PublicKey != "PUB" || r.TLS.ServerName != "www.microsoft.com" {
		t.Errorf("reality = %+v tls=%+v", r, r.TLS)
	}
	ws := by["ws"]
	if ws.Server != "example.com" || ws.Transport == nil || ws.Transport.Type != "ws" || ws.Transport.Host != "cdn.example.com" ||
		ws.TLS == nil || !ws.TLS.Insecure || len(ws.TLS.ALPN) != 2 {
		t.Errorf("ws (flat layout) = %+v transport=%+v", ws, ws.Transport)
	}
	if vm := by["vm"]; vm.Protocol != profile.VMess || vm.Transport == nil || vm.Transport.ServiceName != "svc" {
		t.Errorf("vmess = %+v", vm)
	}
	if tj := by["tj"]; tj.Password != "pw" || tj.TLS == nil {
		t.Errorf("trojan = %+v", tj)
	}
	if ss := by["ss"]; ss.Method != "aes-256-gcm" || ss.Password != "secret" {
		t.Errorf("ss = %+v", ss)
	}
	if wg := by["wg"]; wg.WireGuard == nil || wg.Server != "wg.example.com" || wg.Port != 51820 ||
		wg.WireGuard.PrivateKey != "PRIV" || wg.WireGuard.PeerPublicKey != "PEERPUB" || len(wg.WireGuard.Address) != 1 {
		t.Errorf("wireguard = %+v wg=%+v", wg, wg.WireGuard)
	}
}

func TestImportXrayServerConfigIsRefused(t *testing.T) {
	_, err := Import(`{"inbounds":[{"port":443,"protocol":"vless","settings":{"clients":[{"id":"x"}]}}],"outbounds":[{"protocol":"freedom"}]}`)
	if err == nil || !strings.Contains(err.Error(), "server") {
		t.Fatalf("err = %v, want a hint to import it on the server", err)
	}
}

func TestImportSIP008(t *testing.T) {
	res, err := Import(`{"version":1,"servers":[
	  {"id":"a","remarks":"one","server":"203.0.113.7","server_port":8388,"password":"pw","method":"2022-blake3-aes-128-gcm"},
	  {"id":"b","remarks":"two","server":"203.0.113.8","server_port":8389,"password":"pw2","method":"aes-256-gcm","plugin":"obfs-local","plugin_opts":"obfs=tls"},
	  {"id":"c","remarks":"broken","server":"203.0.113.9","server_port":0,"password":"x","method":"aes-256-gcm"}]}`)
	if err != nil || len(res.Profiles) != 2 || len(res.Errors) != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if p := res.Profiles[1]; p.Name != "two" || p.Plugin != "obfs-local" || p.PluginOpts != "obfs=tls" || p.Port != 8389 {
		t.Errorf("second server = %+v", p)
	}
}

// A sing-box config must still go the sing-box way.
func TestSingBoxJSONIsNotMistakenForXray(t *testing.T) {
	if k := kindOfJSON(`{"outbounds":[{"type":"shadowsocks","tag":"a","server":"1.1.1.1","server_port":1}]}`); k != "sing-box" {
		t.Fatalf("kind = %s", k)
	}
}
