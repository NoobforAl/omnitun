package importer

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// parseURL parses a link and returns the URL with its host and port split out.
func parseURL(link string) (*url.URL, string, int, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, "", 0, fmt.Errorf("invalid link: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return nil, "", 0, fmt.Errorf("missing server address")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, "", 0, fmt.Errorf("invalid port %q", u.Port())
	}
	return u, host, port, nil
}

func fragmentName(u *url.URL) string {
	return strings.TrimSpace(u.Fragment)
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for v := range strings.SplitSeq(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func truthy(s string) bool {
	switch strings.ToLower(s) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// tlsFromQuery reads the common TLS/Reality query parameters used by
// v2rayN-style vless and trojan links.
func tlsFromQuery(q url.Values, defaultOn bool) *profile.TLS {
	security := q.Get("security")
	if security == "" && defaultOn {
		security = "tls"
	}
	if security == "" || security == "none" {
		return nil
	}
	t := &profile.TLS{
		Enabled:     true,
		ServerName:  firstNonEmpty(q.Get("sni"), q.Get("peer")),
		ALPN:        splitList(q.Get("alpn")),
		Insecure:    truthy(q.Get("allowInsecure")) || truthy(q.Get("insecure")),
		Fingerprint: q.Get("fp"),
	}
	if security == "reality" {
		t.Reality = &profile.Reality{PublicKey: q.Get("pbk"), ShortID: q.Get("sid")}
		if t.Fingerprint == "" {
			t.Fingerprint = "chrome"
		}
	}
	return t
}

func transportFromQuery(q url.Values) *profile.Transport {
	typ := q.Get("type")
	switch typ {
	case "", "tcp", "raw":
		if q.Get("headerType") == "http" {
			return &profile.Transport{Type: "http", Path: q.Get("path"), Host: q.Get("host")}
		}
		return nil
	case "ws", "httpupgrade":
		return &profile.Transport{Type: typ, Path: q.Get("path"), Host: q.Get("host")}
	case "grpc":
		return &profile.Transport{Type: "grpc", ServiceName: q.Get("serviceName")}
	case "http", "h2":
		return &profile.Transport{Type: "http", Path: q.Get("path"), Host: q.Get("host")}
	default:
		return &profile.Transport{Type: typ, Path: q.Get("path"), Host: q.Get("host")}
	}
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// vless://uuid@host:port?security=reality&sni=..&pbk=..&sid=..&flow=..&type=ws&path=..#name
func parseVLESS(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	return &profile.Profile{
		Name:      fragmentName(u),
		Protocol:  profile.VLESS,
		Server:    host,
		Port:      port,
		UUID:      u.User.Username(),
		Flow:      q.Get("flow"),
		TLS:       tlsFromQuery(q, false),
		Transport: transportFromQuery(q),
	}, nil
}

// trojan://password@host:port?sni=..&type=ws#name — TLS is on by default.
func parseTrojan(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	return &profile.Profile{
		Name:      fragmentName(u),
		Protocol:  profile.Trojan,
		Server:    host,
		Port:      port,
		Password:  u.User.Username(),
		TLS:       tlsFromQuery(q, true),
		Transport: transportFromQuery(q),
	}, nil
}

// vmess://base64(json) in the v2rayN format.
func parseVMess(link string) (*profile.Profile, error) {
	body := strings.TrimPrefix(link[strings.Index(link, "://")+3:], "/")
	decoded, ok := decodeBase64(body)
	if !ok {
		return nil, fmt.Errorf("vmess: body is not base64")
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(decoded), &v); err != nil {
		return nil, fmt.Errorf("vmess: %w", err)
	}
	str := func(k string) string {
		switch x := v[k].(type) {
		case string:
			return x
		case float64:
			return strconv.Itoa(int(x))
		}
		return ""
	}
	port, err := strconv.Atoi(str("port"))
	if err != nil {
		return nil, fmt.Errorf("vmess: invalid port %q", str("port"))
	}
	aid, _ := strconv.Atoi(str("aid"))
	p := &profile.Profile{
		Name:     str("ps"),
		Protocol: profile.VMess,
		Server:   str("add"),
		Port:     port,
		UUID:     str("id"),
		AlterID:  aid,
		Method:   firstNonEmpty(str("scy"), "auto"),
	}
	if str("tls") == "tls" {
		p.TLS = &profile.TLS{
			Enabled:     true,
			ServerName:  firstNonEmpty(str("sni"), str("host")),
			ALPN:        splitList(str("alpn")),
			Fingerprint: str("fp"),
		}
	}
	q := url.Values{}
	q.Set("type", str("net"))
	q.Set("path", str("path"))
	q.Set("host", str("host"))
	q.Set("headerType", str("type"))
	q.Set("serviceName", str("path"))
	p.Transport = transportFromQuery(q)
	return p, nil
}

// ss://base64(method:password)@host:port#name (SIP002) or
// ss://base64(method:password@host:port)#name (legacy).
func parseShadowsocks(link string) (*profile.Profile, error) {
	rest := link[len("ss://"):]
	name := ""
	if i := strings.Index(rest, "#"); i >= 0 {
		name, _ = url.PathUnescape(rest[i+1:])
		rest = rest[:i]
	}
	if !strings.Contains(rest, "@") {
		decoded, ok := decodeBase64(rest)
		if !ok {
			return nil, fmt.Errorf("ss: invalid legacy link")
		}
		rest = decoded
	}
	u, err := url.Parse("ss://" + rest)
	if err != nil {
		return nil, fmt.Errorf("ss: %w", err)
	}
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil || host == "" {
		return nil, fmt.Errorf("ss: invalid server address")
	}
	method, password := "", ""
	if pw, ok := u.User.Password(); ok {
		method, password = u.User.Username(), pw
	} else if decoded, ok := decodeBase64(u.User.Username()); ok {
		method, password, _ = strings.Cut(decoded, ":")
	} else {
		return nil, fmt.Errorf("ss: invalid user info")
	}
	p := &profile.Profile{
		Name:     name,
		Protocol: profile.Shadowsocks,
		Server:   host,
		Port:     port,
		Method:   method,
		Password: password,
	}
	if plugin := u.Query().Get("plugin"); plugin != "" {
		p.Plugin, p.PluginOpts, _ = strings.Cut(plugin, ";")
	}
	return p, nil
}

// hysteria2://auth@host:port?sni=..&obfs=salamander&obfs-password=..&insecure=1#name
func parseHysteria2(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	password := u.User.Username()
	if pw, ok := u.User.Password(); ok {
		password += ":" + pw
	}
	p := &profile.Profile{
		Name:     fragmentName(u),
		Protocol: profile.Hysteria2,
		Server:   host,
		Port:     port,
		Password: password,
		TLS: &profile.TLS{
			Enabled:    true,
			ServerName: q.Get("sni"),
			Insecure:   truthy(q.Get("insecure")),
			ALPN:       splitList(q.Get("alpn")),
		},
	}
	if obfs := q.Get("obfs"); obfs != "" {
		p.Hysteria2 = &profile.Hy2{Obfs: obfs, ObfsPassword: q.Get("obfs-password")}
	}
	return p, nil
}

// tuic://uuid:password@host:port?congestion_control=bbr&alpn=h3&sni=..#name
func parseTUIC(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	pw, _ := u.User.Password()
	return &profile.Profile{
		Name:     fragmentName(u),
		Protocol: profile.TUIC,
		Server:   host,
		Port:     port,
		UUID:     u.User.Username(),
		Password: pw,
		TLS: &profile.TLS{
			Enabled:    true,
			ServerName: q.Get("sni"),
			ALPN:       splitList(firstNonEmpty(q.Get("alpn"), "h3")),
			Insecure:   truthy(q.Get("allow_insecure")) || truthy(q.Get("insecure")),
		},
		TUIC: &profile.TUICOpts{
			CongestionControl: q.Get("congestion_control"),
			UDPRelayMode:      q.Get("udp_relay_mode"),
		},
	}, nil
}

// wireguard://privatekey@host:port?publickey=..&address=..&mtu=..&reserved=1,2,3#name
func parseWireGuardLink(link string) (*profile.Profile, error) {
	u, host, port, err := parseURL(link)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	priv, _ := url.PathUnescape(u.User.Username())
	wg := &profile.WG{
		PrivateKey:    priv,
		PeerPublicKey: firstNonEmpty(q.Get("publickey"), q.Get("peer_public_key")),
		PreSharedKey:  firstNonEmpty(q.Get("presharedkey"), q.Get("pre_shared_key")),
		Address:       splitList(firstNonEmpty(q.Get("address"), q.Get("ip"))),
	}
	wg.MTU, _ = strconv.Atoi(q.Get("mtu"))
	wg.AllowedIPs = splitList(firstNonEmpty(q.Get("allowedips"), q.Get("allowed_ips")))
	wg.DNS = splitList(q.Get("dns"))
	wg.Keepalive, _ = strconv.Atoi(firstNonEmpty(q.Get("keepalive"), q.Get("persistent_keepalive")))
	for _, r := range splitList(q.Get("reserved")) {
		n, err := strconv.Atoi(r)
		if err != nil {
			return nil, fmt.Errorf("wireguard: invalid reserved %q", r)
		}
		wg.Reserved = append(wg.Reserved, n)
	}
	proto := profile.WireGuard
	// awg:// is the same link with AmneziaWG's obfuscation parameters.
	if strings.EqualFold(u.Scheme, "awg") {
		proto = profile.AmneziaWG
		for k := range amneziaKeys {
			if v := q.Get(k); v != "" {
				if wg.Amnezia == nil {
					wg.Amnezia = map[string]string{}
				}
				wg.Amnezia[k] = v
			}
		}
	}
	return &profile.Profile{
		Name:      fragmentName(u),
		Protocol:  proto,
		Server:    host,
		Port:      port,
		WireGuard: wg,
	}, nil
}
