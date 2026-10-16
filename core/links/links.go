// Package links turns a profile back into a share link (the reverse of
// core/importer). The server uses it to hand out links, and the app to share
// a profile as a link or QR code. Export followed by importer.ParseLink gives
// the same profile back.
package links

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// ErrNoLink means the protocol has no share-link format (OpenVPN travels as
// a config file).
var ErrNoLink = errors.New("this protocol has no share link; export it as a config file")

// Export returns the share link for p.
func Export(p *profile.Profile) (string, error) {
	switch p.Protocol {
	case profile.VLESS:
		return vless(p), nil
	case profile.VMess:
		return vmess(p)
	case profile.Trojan:
		return trojan(p), nil
	case profile.Shadowsocks:
		return shadowsocks(p), nil
	case profile.Hysteria2:
		return hysteria2(p), nil
	case profile.Hysteria:
		return hysteria(p), nil
	case profile.TUIC:
		return tuic(p), nil
	case profile.AnyTLS:
		return anytls(p), nil
	case profile.SSH:
		return ssh(p), nil
	case profile.Naive:
		return naive(p), nil
	case profile.IKEv2:
		return ikev2(p), nil
	case profile.SSTP:
		return sstp(p), nil
	case profile.Tor:
		return tor(p), nil
	case profile.WireGuard, profile.AmneziaWG:
		return wireguard(p)
	}
	return "", fmt.Errorf("%w (%s)", ErrNoLink, p.Protocol)
}

func hostPort(p *profile.Profile) string {
	return net.JoinHostPort(p.Server, strconv.Itoa(p.Port))
}

// query is an ordered query string: parameters keep the order they were
// added in, which keeps links stable and readable.
type query struct{ keys, vals []string }

func (q *query) add(k, v string) {
	if v != "" {
		q.keys, q.vals = append(q.keys, k), append(q.vals, v)
	}
}

func (q *query) addFlag(k string, on bool) {
	if on {
		q.add(k, "1")
	}
}

func (q *query) String() string {
	var b strings.Builder
	for i, k := range q.keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(k) + "=" + url.QueryEscape(q.vals[i]))
	}
	return b.String()
}

// build assembles scheme://userinfo@host:port/path?query#name.
func build(scheme, userinfo, host, path string, q *query, name string) string {
	var b strings.Builder
	b.WriteString(scheme + "://")
	if userinfo != "" {
		b.WriteString(userinfo + "@")
	}
	b.WriteString(host + path)
	if s := q.String(); s != "" {
		b.WriteString("?" + s)
	}
	if name != "" {
		b.WriteString("#" + url.PathEscape(name))
	}
	return b.String()
}

// insecure adds the flag under the names different clients read.
func insecure(q *query, t *profile.TLS, names ...string) {
	if t != nil && t.Insecure {
		for _, n := range names {
			q.add(n, "1")
		}
	}
}

// tlsQuery writes the v2rayN-style TLS/Reality parameters.
func tlsQuery(q *query, t *profile.TLS, flow string) {
	switch {
	case t == nil || !t.Enabled:
		q.add("security", "none")
		return
	case t.Reality != nil:
		q.add("security", "reality")
	default:
		q.add("security", "tls")
	}
	q.add("flow", flow)
	q.add("sni", t.ServerName)
	q.add("fp", t.Fingerprint)
	q.add("alpn", strings.Join(t.ALPN, ","))
	if t.Reality != nil {
		q.add("pbk", t.Reality.PublicKey)
		q.add("sid", t.Reality.ShortID)
	}
	insecure(q, t, "insecure", "allowInsecure", "allow_insecure")
}

func transportQuery(q *query, tr *profile.Transport) {
	if tr == nil {
		q.add("type", "tcp")
		return
	}
	q.add("type", tr.Type)
	q.add("host", tr.Host)
	q.add("path", tr.Path)
	q.add("serviceName", tr.ServiceName)
}

func vless(p *profile.Profile) string {
	q := &query{}
	transportQuery(q, p.Transport)
	tlsQuery(q, p.TLS, p.Flow)
	if p.TLS == nil || !p.TLS.Enabled {
		q.add("flow", p.Flow)
	}
	return build("vless", url.PathEscape(p.UUID), hostPort(p), "", q, p.Name)
}

func trojan(p *profile.Profile) string {
	q := &query{}
	if p.Transport != nil {
		transportQuery(q, p.Transport)
	}
	tlsQuery(q, p.TLS, "")
	return build("trojan", url.PathEscape(p.Password), hostPort(p), "", q, p.Name)
}

func vmess(p *profile.Profile) (string, error) {
	v := map[string]any{
		"v": "2", "ps": p.Name, "add": p.Server, "port": strconv.Itoa(p.Port), "id": p.UUID,
		"aid": strconv.Itoa(p.AlterID), "scy": orDefault(p.Method, "auto"), "net": "tcp", "type": "none",
	}
	if tr := p.Transport; tr != nil {
		v["net"] = tr.Type
		switch tr.Type {
		case "grpc":
			v["path"] = tr.ServiceName
		default:
			v["path"], v["host"] = tr.Path, tr.Host
		}
	}
	if t := p.TLS; t != nil && t.Enabled {
		v["tls"], v["sni"], v["fp"], v["alpn"] = "tls", t.ServerName, t.Fingerprint, strings.Join(t.ALPN, ",")
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(raw), nil
}

func shadowsocks(p *profile.Profile) string {
	userinfo := base64.RawURLEncoding.EncodeToString([]byte(p.Method + ":" + p.Password))
	q := &query{}
	if p.Plugin != "" {
		plugin := p.Plugin
		if p.PluginOpts != "" {
			plugin += ";" + p.PluginOpts
		}
		q.add("plugin", plugin)
	}
	return build("ss", userinfo, hostPort(p), "", q, p.Name)
}

func hysteria2(p *profile.Profile) string {
	q := &query{}
	if p.TLS != nil {
		q.add("sni", p.TLS.ServerName)
		q.add("alpn", strings.Join(p.TLS.ALPN, ","))
		insecure(q, p.TLS, "insecure", "allowInsecure", "allow_insecure")
	}
	if h := p.Hysteria2; h != nil {
		q.add("obfs", h.Obfs)
		q.add("obfs-password", h.ObfsPassword)
	}
	return build("hysteria2", userinfoPassword(p.Password), hostPort(p), "", q, p.Name)
}

// userinfoPassword escapes a password that may contain "user:pass".
func userinfoPassword(pw string) string {
	user, pass, ok := strings.Cut(pw, ":")
	if !ok {
		return url.PathEscape(pw)
	}
	return url.PathEscape(user) + ":" + url.PathEscape(pass)
}

func hysteria(p *profile.Profile) string {
	q := &query{}
	q.add("protocol", "udp")
	q.add("auth", p.Password)
	if p.TLS != nil {
		q.add("peer", p.TLS.ServerName)
		q.add("alpn", strings.Join(p.TLS.ALPN, ","))
		insecure(q, p.TLS, "insecure")
	}
	if h := p.Hysteria2; h != nil {
		if h.UpMbps > 0 {
			q.add("upmbps", strconv.Itoa(h.UpMbps))
		}
		if h.DownMbps > 0 {
			q.add("downmbps", strconv.Itoa(h.DownMbps))
		}
		q.add("obfsParam", h.Obfs)
	}
	return build("hysteria", "", hostPort(p), "", q, p.Name)
}

func tuic(p *profile.Profile) string {
	q := &query{}
	if t := p.TUIC; t != nil {
		q.add("congestion_control", t.CongestionControl)
		q.add("udp_relay_mode", t.UDPRelayMode)
	}
	if p.TLS != nil {
		q.add("alpn", strings.Join(p.TLS.ALPN, ","))
		q.add("sni", p.TLS.ServerName)
		insecure(q, p.TLS, "insecure", "allowInsecure", "allow_insecure")
	}
	return build("tuic", url.PathEscape(p.UUID)+":"+url.PathEscape(p.Password), hostPort(p), "", q, p.Name)
}

func anytls(p *profile.Profile) string {
	q := &query{}
	if p.TLS != nil {
		q.add("sni", p.TLS.ServerName)
		q.add("fp", p.TLS.Fingerprint)
		q.add("alpn", strings.Join(p.TLS.ALPN, ","))
		insecure(q, p.TLS, "insecure", "allowInsecure", "allow_insecure")
	}
	return build("anytls", url.PathEscape(p.Password), hostPort(p), "/", q, p.Name)
}

func ssh(p *profile.Profile) string {
	q := &query{}
	if o := p.SSH; o != nil {
		q.add("pk", o.PrivateKey)
		q.add("pkp", o.PrivateKeyPassphrase)
		q.add("hk", strings.Join(o.HostKey, ","))
		q.add("client_version", o.ClientVersion)
	}
	userinfo := url.PathEscape(p.Username)
	if p.Password != "" {
		userinfo += ":" + url.PathEscape(p.Password)
	}
	return build("ssh", userinfo, hostPort(p), "", q, p.Name)
}

func naive(p *profile.Profile) string {
	scheme, q := "naive+https", &query{}
	if p.TLS != nil {
		q.add("sni", p.TLS.ServerName)
	}
	if o := p.Naive; o != nil {
		if o.QUIC {
			scheme = "naive+quic"
		}
		if len(o.ExtraHeaders) > 0 {
			keys := make([]string, 0, len(o.ExtraHeaders))
			for k := range o.ExtraHeaders {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			lines := make([]string, len(keys))
			for i, k := range keys {
				lines[i] = k + ": " + o.ExtraHeaders[k]
			}
			q.add("extra-headers", strings.Join(lines, "\r\n"))
		}
	}
	userinfo := url.PathEscape(p.Username) + ":" + url.PathEscape(p.Password)
	return build(scheme, userinfo, hostPort(p), "/", q, p.Name)
}

func ikev2(p *profile.Profile) string {
	q := &query{}
	o := p.IKEv2
	if o == nil {
		o = &profile.IKEv2Opts{Auth: "eap"}
	}
	q.add("auth", o.Auth)
	q.add("psk", o.PSK)
	q.add("remote_id", o.RemoteID)
	q.add("local_id", o.LocalID)
	q.add("ca", o.CACert)
	if o.MTU > 0 {
		q.add("mtu", strconv.Itoa(o.MTU))
	}
	q.addFlag("tcp", o.TCP)
	q.addFlag("system", o.System)
	userinfo := ""
	if p.Username != "" {
		userinfo = url.PathEscape(p.Username)
		if p.Password != "" {
			userinfo += ":" + url.PathEscape(p.Password)
		}
	}
	return build("ikev2", userinfo, hostPort(p), "", q, p.Name)
}

func sstp(p *profile.Profile) string {
	q := &query{}
	if p.TLS != nil && !p.TLS.Insecure {
		q.add("insecure", "0") // the parser's default is insecure=1
	}
	userinfo := url.PathEscape(p.Username) + ":" + url.PathEscape(p.Password)
	return build("sstp", userinfo, hostPort(p), "", q, p.Name)
}

func tor(p *profile.Profile) string {
	if p.Tor != nil && p.Tor.Builtin != "" {
		return "tor://" + p.Tor.Builtin
	}
	if p.Tor != nil && len(p.Tor.Bridges) > 0 {
		return strings.Join(p.Tor.Bridges, "\n")
	}
	return "tor://"
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func wireguard(p *profile.Profile) (string, error) {
	g := p.WireGuard
	if g == nil || g.PrivateKey == "" {
		return "", errors.New("the WireGuard profile has no private key")
	}
	q := &query{}
	q.add("publickey", g.PeerPublicKey)
	q.add("presharedkey", g.PreSharedKey)
	q.add("address", strings.Join(g.Address, ","))
	q.add("allowedips", strings.Join(g.AllowedIPs, ","))
	q.add("dns", strings.Join(g.DNS, ","))
	if g.MTU > 0 {
		q.add("mtu", strconv.Itoa(g.MTU))
	}
	if g.Keepalive > 0 {
		q.add("keepalive", strconv.Itoa(g.Keepalive))
	}
	if len(g.Reserved) > 0 {
		parts := make([]string, len(g.Reserved))
		for i, r := range g.Reserved {
			parts[i] = strconv.Itoa(r)
		}
		q.add("reserved", strings.Join(parts, ","))
	}
	scheme := "wireguard"
	if p.Protocol == profile.AmneziaWG {
		scheme = "awg"
		keys := make([]string, 0, len(g.Amnezia))
		for k := range g.Amnezia {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			q.add(k, g.Amnezia[k])
		}
	}
	return build(scheme, url.PathEscape(g.PrivateKey), hostPort(p), "", q, p.Name), nil
}
