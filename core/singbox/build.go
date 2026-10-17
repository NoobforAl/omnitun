// Package singbox converts profiles into a sing-box configuration.
package singbox

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/NoobforAl/omnitun/core/profile"
)

// ErrOtherEngine marks profiles that another engine of the core runs (SSTP,
// IKEv2, AmneziaWG): they have no sing-box form.
var ErrOtherEngine = errors.New("not run by sing-box")

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
	// DNS is the remote resolver used through the tunnel, as a URL:
	// https://host/path (DoH), tls://host (DoT), tcp://host, udp://host or a bare IP (UDP).
	// Default https://1.1.1.1/dns-query.
	DNS string
	// LocalDNS is the resolver used outside the tunnel (the proxy servers' own
	// names, and bypassed sites), as a resolver URL. Empty uses the operating
	// system's resolver, which phones need the engine to name (see engine.systemDNS).
	LocalDNS string
	// ProxyPrivate also sends local addresses (LocalPrefixes) through the
	// tunnel; by default they bypass it and aren't routed into the VPN.
	ProxyPrivate bool
	// SystemProxy points the OS proxy settings at the local port while
	// running (proxy mode only); sing-box restores them on stop.
	SystemProxy bool

	// Failover and FailoverInterval are used by the engine, not the config:
	// when a pinned profile stops responding, switch to the next working one.
	// IncludePackages, when set, sends only these Android apps through the
	// VPN; otherwise ExcludePackages bypass it. TUN mode only.
	IncludePackages []string
	ExcludePackages []string

	Failover         bool
	FailoverInterval time.Duration

	// TorSocks is the running Tor client's SOCKS5 address (host:port); Tor
	// profiles are skipped without it.
	TorSocks string

	// Bypass lists destinations that go around the VPN. With BypassOnly it is
	// the other way round: only they use the VPN and everything else goes direct.
	Bypass     *BypassList
	BypassOnly bool
}

// SelfPackage in ExcludePackages stands for this app: the platform replaces it
// with the app's own package so its sockets (Tor, bridge transports) stay off
// the VPN.
const SelfPackage = "@self"

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
		o.DNS = "https://1.1.1.1/dns-query"
	}
	// Tor exits don't allow DNS over TLS (port 853) or plain DNS, but they do
	// allow HTTPS: resolve through DoH while Tor is the way out.
	if o.TorSocks != "" && !strings.HasPrefix(o.DNS, "https://") {
		o.DNS = "https://1.1.1.1/dns-query"
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
		var (
			out        obj
			isEndpoint bool
			extras     []obj
			err        error
		)
		if p.Protocol == profile.Tor {
			out, err = torOutbound(opts.TorSocks, p.ID)
		} else {
			out, isEndpoint, extras, err = convert(p, p.ID)
		}
		if err != nil {
			skipped = append(skipped, fmt.Errorf("%s: %w", p.DisplayName(), err))
			continue
		}
		if isEndpoint {
			endpoints = append(endpoints, out)
		} else {
			outbounds = append(outbounds, out)
		}
		for _, x := range extras {
			outbounds = append(outbounds, x)
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
		tun := obj{
			"type":         "tun",
			"tag":          "tun-in",
			"address":      tunAddresses,
			"mtu":          9000,
			"auto_route":   true,
			"strict_route": true,
			"stack":        "mixed",
		}
		// Tor runs as this app, which is kept out of the VPN; the system TCP
		// stack answers through a listener in this app, and those replies
		// would then bypass the tunnel. gVisor handles everything in userspace.
		if opts.TorSocks != "" {
			tun["stack"] = "gvisor"
		}
		// Local networks don't enter the VPN at all (unless ProxyPrivate), nor do
		// the address ranges of the user's bypass list.
		var local []netip.Prefix
		if !opts.ProxyPrivate {
			local = BypassPrefixes(profileKeep(profiles))
		}
		switch {
		case opts.BypassOnly && opts.Bypass.OnlyRanges():
			// Only addresses are listed: nothing else has to enter the tunnel at
			// all, so the rest of the device's traffic costs the VPN nothing. The
			// tunnel's own subnet stays routed, because its DNS server lives there.
			tun["route_address"] = prefixStrings(append(ParsePrefixes(tunAddresses), MergePrefixes(opts.Bypass.CIDRs)...))
			if len(local) > 0 {
				tun["route_exclude_address"] = prefixStrings(local)
			}
		case !opts.BypassOnly && !opts.Bypass.Empty():
			// Listed ranges stay out of the VPN's routes, as many as fit the
			// route budget; the rest is caught by the rules below.
			tun["route_exclude_address"] = prefixStrings(FitExclusions(local, opts.Bypass.CIDRs, RouteBudget))
		case len(local) > 0:
			tun["route_exclude_address"] = prefixStrings(local)
		}
		// Per-app split tunneling (Android): include wins over exclude.
		if len(opts.IncludePackages) > 0 {
			tun["include_package"] = opts.IncludePackages
		} else if len(opts.ExcludePackages) > 0 {
			tun["exclude_package"] = opts.ExcludePackages
		}
		inbounds = append(inbounds, tun)
	default:
		return nil, skipped, fmt.Errorf("unknown mode %q", opts.Mode)
	}

	remoteDNS, err := dnsServer(opts.DNS)
	if err != nil {
		return nil, skipped, err
	}
	remoteDNS["tag"] = "remote"
	remoteDNS["detour"] = TagSelect

	// Rules run in order, cheapest first, so most connections are decided
	// before anything costly happens: DNS by port, then address matches (no
	// packet inspection), and only then sniffing for names — and only when a
	// listed name needs it.
	rules := []any{obj{"port": 53, "action": "hijack-dns"}}
	if !opts.ProxyPrivate {
		// Local addresses go direct (proxy mode, and anything that still
		// reaches the TUN); ranges a profile routes on purpose stay proxied.
		rules = append(rules, obj{"ip_cidr": prefixStrings(BypassPrefixes(profileKeep(profiles))), "outbound": TagDirect})
	}
	// The user's list: these go direct, or (BypassOnly) only these use the VPN.
	routeFinal, dnsFinal, dnsRules := TagSelect, "remote", []any(nil)
	if bypass := opts.Bypass.Optimized(); !bypass.Empty() {
		listed, listedDNS := TagDirect, "local"
		if opts.BypassOnly {
			listed, listedDNS = TagSelect, "remote"
			routeFinal, dnsFinal = TagDirect, "local"
		}
		byAddress, byName := bypass.addressRules(), bypass.domainRules()
		for _, r := range byAddress {
			r["outbound"] = listed
			rules = append(rules, obj(r))
		}
		if len(byName) > 0 {
			// A connection arrives as an address; its name comes from the first
			// packet. TLS, HTTP and QUIC cover the web; nothing else is read.
			rules = append(rules,
				obj{"action": "sniff", "sniffer": []string{"tls", "http", "quic"}},
				obj{"protocol": "dns", "action": "hijack-dns"},
			)
			for _, r := range byName {
				r["outbound"] = listed
				rules = append(rules, obj(r))
			}
			// Names are looked up the same way as they are reached.
			for _, r := range byName {
				d := obj{}
				for k, v := range r {
					if k != "outbound" {
						d[k] = v
					}
				}
				d["server"] = listedDNS
				dnsRules = append(dnsRules, d)
			}
		}
	}

	localDNS := obj{"type": "local"}
	if opts.LocalDNS != "" {
		if localDNS, err = dnsServer(opts.LocalDNS); err != nil {
			return nil, skipped, err
		}
	}
	localDNS["tag"] = "local"

	cfg := obj{
		"log": obj{"level": opts.LogLevel, "timestamp": true},
		"dns": obj{
			"servers": []any{
				remoteDNS,
				localDNS,
			},
			"final":    dnsFinal,
			"strategy": "prefer_ipv4",
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": obj{
			"rules":                   rules,
			"final":                   routeFinal,
			"default_domain_resolver": "local",
		},
	}
	// Binding outbound sockets to the physical interface is only needed to keep
	// them out of our own TUN; in proxy mode it just adds failure modes (some
	// Windows security software rejects interface-bound sockets).
	if opts.Mode == ModeTUN {
		cfg["route"].(obj)["auto_detect_interface"] = true
	}
	if len(dnsRules) > 0 {
		cfg["dns"].(obj)["rules"] = dnsRules
	}
	if len(endpoints) > 0 {
		cfg["endpoints"] = endpoints
	}
	return cfg, skipped, nil
}

// convert maps one profile to a sing-box outbound tagged tag (or an endpoint
// for WireGuard and OpenVPN). extras are helper outbounds it depends on
// (ShadowTLS), tagged after tag.
func convert(p *profile.Profile, tag string) (out obj, isEndpoint bool, extras []obj, err error) {
	out, isEndpoint, extras, err = convertProfile(p, tag)
	if out != nil {
		out["tag"] = tag
	}
	return
}

func convertProfile(p *profile.Profile, tag string) (out obj, isEndpoint bool, extras []obj, err error) {
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
		switch p.Plugin {
		case "":
		case "shadow-tls":
			stls, err := shadowTLSOutbound(p, tag+"-shadowtls")
			if err != nil {
				return nil, false, nil, err
			}
			base["detour"] = stls["tag"]
			extras = append(extras, stls)
		default:
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
	case profile.Hysteria:
		base["type"] = "hysteria"
		if p.Password != "" {
			base["auth_str"] = p.Password
		}
		// Hysteria v1 needs bandwidth hints; these defaults suit mobile links.
		up, down := 10, 50
		if h := p.Hysteria2; h != nil {
			if h.Obfs != "" {
				base["obfs"] = h.Obfs
			}
			if h.UpMbps > 0 {
				up = h.UpMbps
			}
			if h.DownMbps > 0 {
				down = h.DownMbps
			}
		}
		base["up_mbps"], base["down_mbps"] = up, down
		t := tlsOrDefault(p.TLS)
		if len(t.ALPN) == 0 {
			c := *t
			c.ALPN = []string{"hysteria"}
			t = &c
		}
		addTLS(base, t)
	case profile.AnyTLS:
		base["type"] = "anytls"
		base["password"] = p.Password
		addTLS(base, tlsOrDefault(p.TLS))
	case profile.SSH:
		base["type"] = "ssh"
		base["user"] = p.Username
		if p.Password != "" {
			base["password"] = p.Password
		}
		if o := p.SSH; o != nil {
			if o.PrivateKey != "" {
				base["private_key"] = o.PrivateKey
			}
			if o.PrivateKeyPassphrase != "" {
				base["private_key_passphrase"] = o.PrivateKeyPassphrase
			}
			if len(o.HostKey) > 0 {
				base["host_key"] = o.HostKey
			}
			if o.ClientVersion != "" {
				base["client_version"] = o.ClientVersion
			}
		}
	case profile.Naive:
		// Naive runs on Chromium's network stack (cronet): it takes only the
		// server name and an optional pinned certificate, no other TLS options.
		base["type"] = "naive"
		base["username"], base["password"] = p.Username, p.Password
		tls := obj{"enabled": true}
		if p.TLS != nil {
			if p.TLS.ServerName != "" {
				tls["server_name"] = p.TLS.ServerName
			}
			if p.TLS.Certificate != "" {
				tls["certificate"] = p.TLS.Certificate
			}
			if p.TLS.Insecure && p.TLS.Certificate == "" {
				return nil, false, nil, fmt.Errorf("naive can't skip certificate checks; pin the server certificate instead")
			}
		}
		base["tls"] = tls
		if o := p.Naive; o != nil {
			if o.QUIC {
				base["quic"] = true
			}
			if len(o.ExtraHeaders) > 0 {
				base["extra_headers"] = o.ExtraHeaders
			}
		}
	case profile.WireGuard:
		return wireguardEndpoint(p), true, nil, nil
	case profile.OpenVPN:
		ep, err := openvpnEndpoint(p)
		return ep, true, nil, err
	case profile.AmneziaWG:
		return nil, false, nil, fmt.Errorf("%w: amneziawg runs on the dedicated AWG engine (not sing-box)", ErrOtherEngine)
	case profile.IKEv2, profile.SSTP:
		return nil, false, nil, fmt.Errorf("%w: %s runs on the built-in classic VPN engine (not sing-box)", ErrOtherEngine, p.Protocol)
	default:
		return nil, false, nil, fmt.Errorf("protocol %q is not supported by the sing-box engine", p.Protocol)
	}
	return base, false, extras, nil
}

// shadowTLSOutbound is the ShadowTLS layer under a Shadowsocks profile whose
// plugin is "shadow-tls" (options "host=<sni>;password=<pw>;version=<1-3>").
func shadowTLSOutbound(p *profile.Profile, tag string) (obj, error) {
	opts := map[string]string{}
	for kv := range strings.SplitSeq(p.PluginOpts, ";") {
		k, v, _ := strings.Cut(strings.TrimSpace(kv), "=")
		if k != "" {
			opts[k] = v
		}
	}
	version := 3
	if v := opts["version"]; v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 3 {
			return nil, fmt.Errorf("shadow-tls: invalid version %q", v)
		}
		version = n
	}
	if opts["host"] == "" {
		return nil, fmt.Errorf("shadow-tls: missing host")
	}
	tls := obj{"enabled": true, "server_name": opts["host"], "utls": obj{"enabled": true, "fingerprint": "chrome"}}
	if opts["insecure"] == "1" || opts["insecure"] == "true" {
		tls["insecure"] = true
	}
	out := obj{"type": "shadowtls", "tag": tag, "server": p.Server, "server_port": p.Port, "version": version, "tls": tls}
	if version > 1 {
		if opts["password"] == "" {
			return nil, fmt.Errorf("shadow-tls: missing password")
		}
		out["password"] = opts["password"]
	}
	return out, nil
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

// openvpnEndpoint maps an OpenVPN profile to sing-box's openvpn-client endpoint.
func openvpnEndpoint(p *profile.Profile) (obj, error) {
	o := p.OpenVPN
	if o == nil {
		return nil, fmt.Errorf("missing OpenVPN settings")
	}
	if o.NeedsAuth && (p.Username == "" || p.Password == "") {
		return nil, profile.ErrNeedsCredentials
	}
	ep := obj{"type": "openvpn-client"}
	if len(o.Remotes) == 0 {
		ep["server"], ep["server_port"], ep["network"] = p.Server, p.Port, orDefault(o.Network, "udp")
	} else {
		servers := []any{obj{"server": p.Server, "server_port": p.Port, "network": orDefault(o.Network, "udp")}}
		for _, r := range o.Remotes {
			servers = append(servers, obj{"server": r.Server, "server_port": r.Port, "network": orDefault(r.Network, "udp")})
		}
		ep["servers"] = servers
	}
	if o.RemoteRandom {
		ep["remote_random"] = true
	}
	if p.Username != "" {
		ep["username"], ep["password"] = p.Username, p.Password
	}
	tls := obj{"certificate": []string{o.CA}}
	if o.Cert != "" {
		tls["client_certificate"] = []string{o.Cert}
		tls["client_key"] = []string{o.Key}
	}
	if o.VerifyX509Name != "" {
		tls["server_name"] = o.VerifyX509Name
		if o.VerifyX509Type != "" {
			tls["server_name_type"] = o.VerifyX509Type
		} else {
			tls["server_name_type"] = "subject"
		}
	}
	if o.RemoteCertTLS != "" {
		tls["remote_certificate_tls"] = o.RemoteCertTLS
	}
	if o.TLSVersionMin != "" {
		tls["version_min"] = o.TLSVersionMin
	}
	if o.TLSCipher != "" {
		tls["cipher"] = o.TLSCipher
	}
	if o.ControlWrap != "" {
		cw := obj{"type": strings.ReplaceAll(o.ControlWrap, "-", "_"), "key": []string{o.ControlWrapKey}}
		if o.ControlWrap == "tls-auth" {
			switch o.KeyDirection {
			case "1":
				cw["direction"] = "client"
			case "0":
				cw["direction"] = "server"
			}
		}
		tls["control_wrap"] = cw
	}
	ep["tls"] = tls
	if len(o.DataCiphers) > 0 {
		ep["data_ciphers"] = o.DataCiphers
	}
	// In TLS mode a client's `cipher` is the fallback for servers without cipher negotiation.
	if fb := orDefault(o.DataCiphersFallback, o.Cipher); fb != "" {
		ep["data_ciphers_fallback"] = fb
	}
	if o.Auth != "" {
		ep["auth"] = o.Auth
	}
	if o.Compression != "" {
		ep["compression"] = o.Compression
	}
	if o.CompressionLZO != "" {
		ep["compression_lzo"] = o.CompressionLZO
	}
	if o.MTU > 0 {
		ep["mtu"] = o.MTU
	}
	if o.MSSFix > 0 {
		ep["mss_fix"] = o.MSSFix
	}
	return ep, nil
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
	if t.Certificate != "" {
		tls["certificate"] = t.Certificate
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

// dnsServer converts a resolver URL into a sing-box DNS server.
func dnsServer(raw string) (obj, error) {
	if !strings.Contains(raw, "://") {
		raw = "udp://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid DNS server %q", raw)
	}
	server := obj{"server": u.Hostname()}
	if p := u.Port(); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid DNS server port %q", p)
		}
		server["server_port"] = port
	}
	switch u.Scheme {
	case "https":
		server["type"] = "https"
		if u.Path != "" && u.Path != "/dns-query" {
			server["path"] = u.Path
		}
	case "tls", "tcp", "udp":
		server["type"] = u.Scheme
	default:
		return nil, fmt.Errorf("unsupported DNS scheme %q (use https, tls, tcp or udp)", u.Scheme)
	}
	return server, nil
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

// Outbound converts one profile to a sing-box outbound tagged tag, or to an
// endpoint (isEndpoint) for WireGuard and OpenVPN. extras are helper
// outbounds it depends on (ShadowTLS) and must be included too.
func Outbound(p *profile.Profile, tag string) (out map[string]any, isEndpoint bool, extras []map[string]any, err error) {
	return convert(p, tag)
}

// torOutbound routes a Tor profile into the local Tor client's SOCKS5 port.
func torOutbound(socks, tag string) (obj, error) {
	host, port, err := net.SplitHostPort(socks)
	if err != nil || socks == "" {
		return nil, fmt.Errorf("Tor is not running")
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("bad Tor SOCKS address %q", socks)
	}
	return obj{"type": "socks", "tag": tag, "server": host, "server_port": n, "version": "5"}, nil
}
