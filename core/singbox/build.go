// Package singbox converts profiles into a sing-box configuration.
package singbox

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// Tags of the built-in outbounds.
const (
	TagSelect = "select"
	TagAuto   = "auto"
	TagDirect = "direct"
)

// Mode selects how traffic enters the engine.
type Mode string

const (
	// ModeProxy exposes a local SOCKS5/HTTP port only; no VPN permission needed.
	ModeProxy Mode = "proxy"
	// ModeTUN captures all traffic through a TUN device.
	ModeTUN Mode = "tun"
)

// Options controls config generation.
type Options struct {
	Mode       Mode
	ListenAddr string // proxy listen address, default 127.0.0.1
	ListenPort int    // proxy port, default 2080
	// Selected is the profile ID to use; empty or TagAuto picks the lowest-latency profile.
	Selected string
	LogLevel string
	// TestURL is probed by the auto group, default https://www.gstatic.com/generate_204.
	TestURL string
	// DNS is the remote DoH server, default 1.1.1.1.
	DNS string
	// ProxyPrivate also sends LAN/private addresses through the tunnel
	// (by default they bypass it).
	ProxyPrivate bool
	// SystemProxy points the OS proxy settings at the local port while
	// running (proxy mode only); sing-box restores them on stop.
	SystemProxy bool
}

func (o *Options) defaults() {
	if o.Mode == "" {
		o.Mode = ModeProxy
	}
	if o.ListenAddr == "" {
		o.ListenAddr = "127.0.0.1"
	}
	if o.ListenPort == 0 {
		o.ListenPort = 2080
	}
	if o.LogLevel == "" {
		o.LogLevel = "info"
	}
	if o.TestURL == "" {
		o.TestURL = "https://www.gstatic.com/generate_204"
	}
	if o.DNS == "" {
		o.DNS = "1.1.1.1"
	}
	if o.Selected == "" {
		o.Selected = TagAuto
	}
}

type obj = map[string]any

// Build returns a sing-box config (as a JSON-ready map) that routes traffic
// through the given profiles. Profiles the engine cannot run are skipped and
// reported in the returned skipped list.
func Build(profiles []*profile.Profile, opts Options) (obj, []error, error) {
	opts.defaults()
	var (
		outbounds []any
		endpoints []any
		tags      []string
		skipped   []error
	)
	for _, p := range profiles {
		p.EnsureID()
		out, isEndpoint, err := convert(p)
		if err != nil {
			skipped = append(skipped, fmt.Errorf("%s: %w", p.DisplayName(), err))
			continue
		}
		out["tag"] = p.ID
		if isEndpoint {
			endpoints = append(endpoints, out)
		} else {
			outbounds = append(outbounds, out)
		}
		tags = append(tags, p.ID)
	}
	if len(tags) == 0 {
		return nil, skipped, fmt.Errorf("no usable profiles")
	}
	if opts.Selected != TagAuto && !slices.Contains(tags, opts.Selected) {
		return nil, skipped, fmt.Errorf("selected profile %q is not available", opts.Selected)
	}

	outbounds = append(outbounds,
		obj{
			"type":      "selector",
			"tag":       TagSelect,
			"outbounds": append([]string{TagAuto}, tags...),
			"default":   opts.Selected,
		},
		obj{
			"type":      "urltest",
			"tag":       TagAuto,
			"outbounds": tags,
			"url":       opts.TestURL,
			"interval":  "3m",
			"tolerance": 50,
		},
		obj{"type": "direct", "tag": TagDirect},
	)

	var inbounds []any
	switch opts.Mode {
	case ModeProxy:
		mixed := obj{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      opts.ListenAddr,
			"listen_port": opts.ListenPort,
		}
		if opts.SystemProxy {
			mixed["set_system_proxy"] = true
		}
		inbounds = append(inbounds, mixed)
	case ModeTUN:
		inbounds = append(inbounds, obj{
			"type":         "tun",
			"tag":          "tun-in",
			"address":      []string{"172.19.0.1/30", "fdfe:dcba:9876::1/126"},
			"mtu":          9000,
			"auto_route":   true,
			"strict_route": true,
			"stack":        "mixed",
		})
	default:
		return nil, skipped, fmt.Errorf("unknown mode %q", opts.Mode)
	}

	rules := []any{
		obj{"action": "sniff"},
		obj{"protocol": "dns", "action": "hijack-dns"},
	}
	if !opts.ProxyPrivate {
		rules = append(rules, obj{"ip_is_private": true, "outbound": TagDirect})
	}

	cfg := obj{
		"log": obj{"level": opts.LogLevel, "timestamp": true},
		"dns": obj{
			"servers": []any{
				obj{"type": "https", "tag": "remote", "server": opts.DNS, "detour": TagSelect},
				obj{"type": "local", "tag": "local"},
			},
			"final":    "remote",
			"strategy": "prefer_ipv4",
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": obj{
			"rules":                   rules,
			"final":                   TagSelect,
			"auto_detect_interface":   true,
			"default_domain_resolver": "local",
		},
	}
	if len(endpoints) > 0 {
		cfg["endpoints"] = endpoints
	}
	return cfg, skipped, nil
}

// convert maps one profile to a sing-box outbound (or endpoint for WireGuard).
func convert(p *profile.Profile) (out obj, isEndpoint bool, err error) {
	base := obj{"server": p.Server, "server_port": p.Port}
	switch p.Protocol {
	case profile.VLESS:
		base["type"] = "vless"
		base["uuid"] = p.UUID
		if p.Flow != "" {
			base["flow"] = p.Flow
		}
		addTLS(base, p.TLS)
		addTransport(base, p.Transport)
	case profile.VMess:
		base["type"] = "vmess"
		base["uuid"] = p.UUID
		base["security"] = orDefault(p.Method, "auto")
		base["alter_id"] = p.AlterID
		addTLS(base, p.TLS)
		addTransport(base, p.Transport)
	case profile.Trojan:
		base["type"] = "trojan"
		base["password"] = p.Password
		addTLS(base, p.TLS)
		addTransport(base, p.Transport)
	case profile.Shadowsocks:
		base["type"] = "shadowsocks"
		base["method"] = p.Method
		base["password"] = p.Password
		if p.Plugin != "" {
			base["plugin"] = p.Plugin
			base["plugin_opts"] = p.PluginOpts
		}
	case profile.Hysteria2:
		base["type"] = "hysteria2"
		base["password"] = p.Password
		if h := p.Hysteria2; h != nil {
			if h.Obfs != "" {
				base["obfs"] = obj{"type": h.Obfs, "password": h.ObfsPassword}
			}
			if h.UpMbps > 0 {
				base["up_mbps"] = h.UpMbps
			}
			if h.DownMbps > 0 {
				base["down_mbps"] = h.DownMbps
			}
		}
		addTLS(base, tlsOrDefault(p.TLS))
	case profile.TUIC:
		base["type"] = "tuic"
		base["uuid"] = p.UUID
		base["password"] = p.Password
		if t := p.TUIC; t != nil {
			if t.CongestionControl != "" {
				base["congestion_control"] = t.CongestionControl
			}
			if t.UDPRelayMode != "" {
				base["udp_relay_mode"] = t.UDPRelayMode
			}
		}
		addTLS(base, tlsOrDefault(p.TLS))
	case profile.WireGuard:
		return wireguardEndpoint(p), true, nil
	case profile.AmneziaWG:
		return nil, false, fmt.Errorf("amneziawg runs on the dedicated AWG engine (not sing-box)")
	default:
		return nil, false, fmt.Errorf("protocol %q is not supported by the sing-box engine", p.Protocol)
	}
	return base, false, nil
}

func wireguardEndpoint(p *profile.Profile) obj {
	wg := p.WireGuard
	allowed := wg.AllowedIPs
	if len(allowed) == 0 {
		allowed = []string{"0.0.0.0/0", "::/0"}
	}
	peer := obj{
		"address":     p.Server,
		"port":        p.Port,
		"public_key":  wg.PeerPublicKey,
		"allowed_ips": allowed,
	}
	if wg.PreSharedKey != "" {
		peer["pre_shared_key"] = wg.PreSharedKey
	}
	if wg.Keepalive > 0 {
		peer["persistent_keepalive_interval"] = wg.Keepalive
	}
	if len(wg.Reserved) > 0 {
		peer["reserved"] = wg.Reserved
	}
	ep := obj{
		"type":        "wireguard",
		"address":     normalizePrefixes(wg.Address),
		"private_key": wg.PrivateKey,
		"peers":       []any{peer},
	}
	if wg.MTU > 0 {
		ep["mtu"] = wg.MTU
	}
	return ep
}

// normalizePrefixes adds a host prefix length to bare addresses.
func normalizePrefixes(addrs []string) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if !strings.Contains(a, "/") {
			if strings.Contains(a, ":") {
				a += "/128"
			} else {
				a += "/32"
			}
		}
		out = append(out, a)
	}
	return out
}

func addTLS(out obj, t *profile.TLS) {
	if t == nil || !t.Enabled {
		return
	}
	tls := obj{"enabled": true}
	if t.ServerName != "" {
		tls["server_name"] = t.ServerName
	}
	if t.Insecure {
		tls["insecure"] = true
	}
	if len(t.ALPN) > 0 {
		tls["alpn"] = t.ALPN
	}
	if t.Fingerprint != "" {
		tls["utls"] = obj{"enabled": true, "fingerprint": t.Fingerprint}
	}
	if r := t.Reality; r != nil {
		tls["reality"] = obj{"enabled": true, "public_key": r.PublicKey, "short_id": r.ShortID}
	}
	out["tls"] = tls
}

func addTransport(out obj, t *profile.Transport) {
	if t == nil {
		return
	}
	switch t.Type {
	case "ws":
		tr := obj{"type": "ws", "path": orDefault(t.Path, "/")}
		if t.Host != "" {
			tr["headers"] = obj{"Host": t.Host}
		}
		// v2ray links carry early data as ?ed=2048 in the path.
		if path, ed, ok := strings.Cut(tr["path"].(string), "?ed="); ok {
			if n, err := strconv.Atoi(ed); err == nil {
				tr["path"] = path
				tr["max_early_data"] = n
				tr["early_data_header_name"] = "Sec-WebSocket-Protocol"
			}
		}
		out["transport"] = tr
	case "httpupgrade":
		tr := obj{"type": "httpupgrade", "path": orDefault(t.Path, "/")}
		if t.Host != "" {
			tr["host"] = t.Host
		}
		out["transport"] = tr
	case "grpc":
		out["transport"] = obj{"type": "grpc", "service_name": t.ServiceName}
	case "http":
		tr := obj{"type": "http"}
		if t.Path != "" {
			tr["path"] = t.Path
		}
		if t.Host != "" {
			tr["host"] = splitComma(t.Host)
		}
		out["transport"] = tr
	}
}

func tlsOrDefault(t *profile.TLS) *profile.TLS {
	if t == nil {
		return &profile.TLS{Enabled: true}
	}
	t.Enabled = true
	return t
}

func splitComma(s string) []string {
	var out []string
	for v := range strings.SplitSeq(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
