package protocols

import (
	"fmt"
	"strings"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
)

type obj = map[string]any

// ssMethod is the Shadowsocks-2022 cipher used by the server.
const ssMethod = "2022-blake3-aes-128-gcm"

func init() {
	// The order is the display and failover order: Reality first (needs no
	// certificate and looks like a real site), UDP protocols last.
	for _, p := range []Protocol{
		vlessReality{}, shadowsocks{}, trojan{}, hysteria2{}, tuic{}, anyTLS{}, shadowTLS{}, naive{}, sstp{}, ikev2{}, amneziawg{}, wireguard{},
		openvpn{udp: true}, openvpn{},
	} {
		register(p)
	}
}

// base is the inbound every protocol starts from.
func base(key string, port int) obj {
	return obj{"tag": key, "listen": "::", "listen_port": port}
}

// ---------------------------------------------------------------- VLESS+Reality

type vlessReality struct{}

func (vlessReality) Spec() Spec {
	return Spec{
		Key: "vless-reality", Name: "VLESS + Reality", Label: "VLESS-Reality", Profile: profile.VLESS,
		Description: "Looks like a visit to a real website; needs no certificate. The best default.",
		Transport:   "tcp", DefaultPort: 443, ShareLink: true,
		Options: []Option{{Key: "reality-site", Label: "Site to imitate", Kind: "string", Default: "www.microsoft.com:443",
			Help: "A real HTTPS site; unauthenticated visitors see its certificate."}},
	}
}

func (vlessReality) Inbounds(env Env, port int, accts []Account) Config {
	host, hsPort := SplitHostPort(env.Reality.Handshake, 443)
	in := base("vless-reality", port)
	in["type"] = "vless"
	in["users"] = users(accts, func(a Account) obj {
		return obj{"name": a.Name, "uuid": a.UUID, "flow": "xtls-rprx-vision"}
	})
	in["tls"] = obj{
		"enabled":     true,
		"server_name": host,
		"reality": obj{
			"enabled":     true,
			"handshake":   obj{"server": host, "server_port": hsPort},
			"private_key": env.Reality.PrivateKey,
			"short_id":    []string{env.Reality.ShortID},
		},
	}
	return Config{Inbounds: []obj{in}}
}

func (v vlessReality) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	sni, _ := SplitHostPort(env.Reality.Handshake, 443)
	return &profile.Profile{
		Name: profileName(env, v.Spec()), Protocol: profile.VLESS, Server: env.Host, Port: port,
		UUID: u.UUID, Flow: "xtls-rprx-vision",
		TLS: &profile.TLS{Enabled: true, ServerName: sni, Fingerprint: "chrome",
			Reality: &profile.Reality{PublicKey: env.Reality.PublicKey, ShortID: env.Reality.ShortID}},
	}, nil
}

// ------------------------------------------------------------ Shadowsocks 2022

type shadowsocks struct{}

func (shadowsocks) Spec() Spec {
	return Spec{
		Key: "shadowsocks", Name: "Shadowsocks 2022", Label: "Shadowsocks", Profile: profile.Shadowsocks,
		Description: "Simple and fast; works on TCP and UDP. No certificate needed.",
		Transport:   "tcp+udp", DefaultPort: 8388, ShareLink: true,
	}
}

func ssInbound(env Env, tag string, port int, accts []Account) obj {
	in := base(tag, port)
	in["type"] = "shadowsocks"
	in["method"] = ssMethod
	in["password"] = env.SSServerKey
	in["users"] = users(accts, func(a Account) obj { return obj{"name": a.Name, "password": a.SSKey} })
	return in
}

func (shadowsocks) Inbounds(env Env, port int, accts []Account) Config {
	return Config{Inbounds: []obj{ssInbound(env, "shadowsocks", port, accts)}}
}

func (s shadowsocks) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	return &profile.Profile{
		Name: profileName(env, s.Spec()), Protocol: profile.Shadowsocks, Server: env.Host, Port: port,
		Method: ssMethod, Password: env.SSServerKey + ":" + u.SSKey,
	}, nil
}

// ------------------------------------------------------------------ Trojan

type trojan struct{}

func (trojan) Spec() Spec {
	return Spec{
		Key: "trojan", Name: "Trojan", Label: "Trojan", Profile: profile.Trojan,
		Description: "TLS with the server's certificate; blends with HTTPS.",
		Transport:   "tcp", DefaultPort: 8443, NeedsCertificate: true, ShareLink: true,
	}
}

func (trojan) Inbounds(env Env, port int, accts []Account) Config {
	in := base("trojan", port)
	in["type"] = "trojan"
	in["users"] = users(accts, func(a Account) obj { return obj{"name": a.Name, "password": a.Password} })
	in["tls"] = tlsBlock(env, nil)
	return Config{Inbounds: []obj{in}}
}

func (t trojan) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	return &profile.Profile{
		Name: profileName(env, t.Spec()), Protocol: profile.Trojan, Server: env.Host, Port: port,
		Password: u.Password, TLS: serverTLS(env),
	}, nil
}

// --------------------------------------------------------------- Hysteria2

type hysteria2 struct{}

func (hysteria2) Spec() Spec {
	return Spec{
		Key: "hysteria2", Name: "Hysteria2", Label: "Hysteria2", Profile: profile.Hysteria2,
		Description: "QUIC based; fast on lossy links. UDP only.",
		Transport:   "udp", DefaultPort: 8443, NeedsCertificate: true, ShareLink: true,
	}
}

func (hysteria2) Inbounds(env Env, port int, accts []Account) Config {
	in := base("hysteria2", port)
	in["type"] = "hysteria2"
	in["users"] = users(accts, func(a Account) obj { return obj{"name": a.Name, "password": a.Password} })
	in["tls"] = tlsBlock(env, obj{"alpn": []string{"h3"}})
	return Config{Inbounds: []obj{in}}
}

func (h hysteria2) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	return &profile.Profile{
		Name: profileName(env, h.Spec()), Protocol: profile.Hysteria2, Server: env.Host, Port: port,
		Password: u.Password, TLS: serverTLS(env),
	}, nil
}

// -------------------------------------------------------------------- TUIC

type tuic struct{}

func (tuic) Spec() Spec {
	return Spec{
		Key: "tuic", Name: "TUIC v5", Label: "TUIC", Profile: profile.TUIC,
		Description: "QUIC based, low latency. UDP only.",
		Transport:   "udp", DefaultPort: 8444, NeedsCertificate: true, ShareLink: true,
	}
}

func (tuic) Inbounds(env Env, port int, accts []Account) Config {
	in := base("tuic", port)
	in["type"] = "tuic"
	in["congestion_control"] = "bbr"
	in["users"] = users(accts, func(a Account) obj {
		return obj{"name": a.Name, "uuid": a.UUID, "password": a.Password}
	})
	in["tls"] = tlsBlock(env, obj{"alpn": []string{"h3"}})
	return Config{Inbounds: []obj{in}}
}

func (t tuic) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	return &profile.Profile{
		Name: profileName(env, t.Spec()), Protocol: profile.TUIC, Server: env.Host, Port: port,
		UUID: u.UUID, Password: u.Password, TLS: serverTLS(env, "h3"),
		TUIC: &profile.TUICOpts{CongestionControl: "bbr"},
	}, nil
}

// ------------------------------------------------------------------ AnyTLS

type anyTLS struct{}

func (anyTLS) Spec() Spec {
	return Spec{
		Key: "anytls", Name: "AnyTLS", Label: "AnyTLS", Profile: profile.AnyTLS,
		Description: "TLS with padding that hides traffic patterns.",
		Transport:   "tcp", DefaultPort: 8445, NeedsCertificate: true, ShareLink: true,
	}
}

func (anyTLS) Inbounds(env Env, port int, accts []Account) Config {
	in := base("anytls", port)
	in["type"] = "anytls"
	in["users"] = users(accts, func(a Account) obj { return obj{"name": a.Name, "password": a.Password} })
	in["tls"] = tlsBlock(env, nil)
	return Config{Inbounds: []obj{in}}
}

func (a anyTLS) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	return &profile.Profile{
		Name: profileName(env, a.Spec()), Protocol: profile.AnyTLS, Server: env.Host, Port: port,
		Password: u.Password, TLS: serverTLS(env),
	}, nil
}

// --------------------------------------------------------------- ShadowTLS

type shadowTLS struct{}

func (shadowTLS) Spec() Spec {
	return Spec{
		Key: "shadowtls", Name: "ShadowTLS v3 + Shadowsocks", Label: "ShadowTLS", Profile: profile.Shadowsocks,
		Description: "Borrows a real site's TLS handshake, then carries Shadowsocks. Needs no certificate.",
		Transport:   "tcp", DefaultPort: 8446, ShareLink: true,
	}
}

func (shadowTLS) Inbounds(env Env, port int, accts []Account) Config {
	// ShadowTLS v3 borrows the handshake of the Reality site, then hands the
	// stream to an internal (not listening) Shadowsocks inbound.
	host, hsPort := SplitHostPort(env.Reality.Handshake, 443)
	inner := ssInbound(env, "shadowtls-ss", 0, accts)
	delete(inner, "listen")
	delete(inner, "listen_port")
	in := base("shadowtls", port)
	in["type"] = "shadowtls"
	in["version"] = 3
	in["users"] = users(accts, func(a Account) obj { return obj{"name": a.Name, "password": a.Password} })
	in["handshake"] = obj{"server": host, "server_port": hsPort}
	in["detour"] = "shadowtls-ss"
	return Config{Inbounds: []obj{inner, in}}
}

func (s shadowTLS) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	if err := safeField("password", u.Password, unsafeInOpts); err != nil {
		return nil, err
	}
	sni, _ := SplitHostPort(env.Reality.Handshake, 443)
	opts := fmt.Sprintf("host=%s;password=%s;version=3", sni, u.Password)
	if env.SelfSigned {
		opts += ";insecure=1"
	}
	return &profile.Profile{
		Name: profileName(env, s.Spec()), Protocol: profile.Shadowsocks, Server: env.Host, Port: port,
		Method: ssMethod, Password: env.SSServerKey + ":" + u.SSKey,
		Plugin: "shadow-tls", PluginOpts: opts,
	}, nil
}

// ------------------------------------------------------------------- Naive

type naive struct{}

func (naive) Spec() Spec {
	return Spec{
		Key: "naive", Name: "NaiveProxy", Label: "Naive", Profile: profile.Naive,
		Description: "HTTP/2 CONNECT through Chromium's network stack; looks like Chrome.",
		Transport:   "tcp", DefaultPort: 8447, NeedsCertificate: true, ShareLink: true,
	}
}

func (naive) Inbounds(env Env, port int, accts []Account) Config {
	in := base("naive", port)
	in["type"] = "naive"
	in["network"] = "tcp"
	in["users"] = users(accts, func(a Account) obj { return obj{"username": a.Name, "password": a.Password} })
	in["tls"] = tlsBlock(env, nil)
	return Config{Inbounds: []obj{in}}
}

func (n naive) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	p := &profile.Profile{
		Name: profileName(env, n.Spec()), Protocol: profile.Naive, Server: env.Host, Port: port,
		Username: u.Name, Password: u.Password, TLS: &profile.TLS{Enabled: true},
	}
	// Chromium rejects an untrusted certificate even with "insecure"; pin the
	// self-signed one instead (a share link can't carry it, a profile can).
	if env.SelfSigned && env.CertPEM != "" {
		p.TLS.Certificate = env.CertPEM
	}
	return p, nil
}

// Characters that break out of the text formats a secret is written into: a
// line break in an .ovpn file adds directives, a semicolon (or a line break)
// in plugin options adds options.
var (
	unsafeInLines = string([]rune{13, 10})
	unsafeInOpts  = ";" + unsafeInLines
)

// safeField rejects values that would break out of the text format they are
// written into (a newline in an .ovpn file adds directives, a semicolon in
// plugin options adds options).
func safeField(what, value, bad string) error {
	if strings.ContainsAny(value, bad) {
		return fmt.Errorf("the %s contains characters that cannot be written safely into the client config", what)
	}
	return nil
}

// ----------------------------------------------------------------- OpenVPN

type openvpn struct{ udp bool }

func (o openvpn) key() string {
	if o.udp {
		return "openvpn-udp"
	}
	return "openvpn-tcp"
}

func (o openvpn) network() string {
	if o.udp {
		return "udp"
	}
	return "tcp"
}

func (o openvpn) Spec() Spec {
	return Spec{
		Key: o.key(), Name: "OpenVPN " + strings.ToUpper(o.network()), Label: "OpenVPN " + strings.ToUpper(o.network()),
		Profile:     profile.OpenVPN,
		Description: "Works with any OpenVPN app (.ovpn file); users sign in with name and password.",
		Transport:   o.network(), DefaultPort: 1194, ShareLink: false, FileExt: ".ovpn",
	}
}

func (o openvpn) Inbounds(env Env, port int, accts []Account) Config {
	pool := "10.18.0.1/16"
	if !o.udp {
		pool = "10.19.0.1/16"
	}
	return Config{Endpoints: []obj{{
		"type": "openvpn-server", "tag": o.key(), "listen": "::", "listen_port": port,
		"network": o.network(), "address": []string{pool},
		"users": users(accts, func(a Account) obj { return obj{"username": a.Name, "password": a.Password} }),
		"tls": obj{
			"certificate":               env.OpenVPN.ServerCert,
			"key":                       env.OpenVPN.ServerKey,
			"verify_client_certificate": "none",
			"control_wrap":              obj{"type": "tls_crypt", "key": env.OpenVPN.TLSCrypt},
		},
		"push": obj{"redirect_gateway": true, "dns": []string{"1.1.1.1", "8.8.8.8"}},
	}}}
}

// File is the user's single-file OpenVPN client config with the credentials
// inline (OpenVPN 2.6+ and Omnitun).
func (o openvpn) File(env Env, port int, u Account) (string, string, error) {
	// The credentials are written as two lines of the config file.
	if err := safeField("user name", u.Name, unsafeInLines); err != nil {
		return "", "", err
	}
	if err := safeField("password", u.Password, unsafeInLines); err != nil {
		return "", "", err
	}
	cfg := fmt.Sprintf(`# %s (%s) for %s
client
dev tun
proto %s
remote %s %d
nobind
persist-key
persist-tun
resolv-retry infinite
remote-cert-tls server
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305
verb 3
auth-user-pass
<auth-user-pass>
%s
%s
</auth-user-pass>
<ca>
%s</ca>
<tls-crypt>
%s</tls-crypt>
`, env.Name, o.key(), u.Name, o.network(), env.Host, port, u.Name, u.Password, env.OpenVPN.CA, env.OpenVPN.TLSCrypt)
	return env.Name + "-" + o.key() + ".ovpn", cfg, nil
}

func (o openvpn) Profile(env Env, port int, u Account) (*profile.Profile, error) {
	_, text, _ := o.File(env, port, u)
	p, err := importer.ParseOVPN(text)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.key(), err)
	}
	p.Name = env.Name + " OpenVPN " + strings.ToUpper(o.network())
	return p, nil
}
