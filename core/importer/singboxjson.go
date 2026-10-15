package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/NoobforAl/omnitun/core/profile"
)

// looksLikeJSON reports whether text is a JSON object.
func looksLikeJSON(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "{")
}

type sbTLS struct {
	Enabled    bool     `json:"enabled"`
	ServerName string   `json:"server_name"`
	Insecure   bool     `json:"insecure"`
	ALPN       []string `json:"alpn"`
	UTLS       *struct {
		Fingerprint string `json:"fingerprint"`
	} `json:"utls"`
	Reality *struct {
		PublicKey string `json:"public_key"`
		ShortID   string `json:"short_id"`
	} `json:"reality"`
}

type sbTransport struct {
	Type        string            `json:"type"`
	Path        string            `json:"path"`
	Headers     map[string]any    `json:"headers"`
	Host        any               `json:"host"`
	ServiceName string            `json:"service_name"`
	Extra       map[string]string `json:"-"`
}

type sbPeer struct {
	Address    string   `json:"address"`
	Port       int      `json:"port"`
	PublicKey  string   `json:"public_key"`
	PreShared  string   `json:"pre_shared_key"`
	AllowedIPs []string `json:"allowed_ips"`
	Reserved   []int    `json:"reserved"`
}

type sbOutbound struct {
	Type       string       `json:"type"`
	Tag        string       `json:"tag"`
	Server     string       `json:"server"`
	ServerPort int          `json:"server_port"`
	UUID       string       `json:"uuid"`
	Password   string       `json:"password"`
	Method     string       `json:"method"`
	Security   string       `json:"security"`
	AlterID    int          `json:"alter_id"`
	Flow       string       `json:"flow"`
	Plugin     string       `json:"plugin"`
	PluginOpts string       `json:"plugin_opts"`
	TLS        *sbTLS       `json:"tls"`
	Transport  *sbTransport `json:"transport"`
	Obfs       *struct {
		Type     string `json:"type"`
		Password string `json:"password"`
	} `json:"obfs"`
	CongestionControl string `json:"congestion_control"`
	UDPRelayMode      string `json:"udp_relay_mode"`
	// WireGuard endpoint
	Address    []string `json:"address"`
	PrivateKey string   `json:"private_key"`
	MTU        int      `json:"mtu"`
	Peers      []sbPeer `json:"peers"`
}

// ImportSingBoxJSON converts the proxy outbounds and WireGuard endpoints of a
// sing-box config. Groups, direct/block/dns outbounds are skipped.
func ImportSingBoxJSON(text string) (*Result, error) {
	var doc struct {
		Outbounds []sbOutbound `json:"outbounds"`
		Endpoints []sbOutbound `json:"endpoints"`
	}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf("sing-box JSON: %w", err)
	}
	res := &Result{}
	for _, o := range append(doc.Outbounds, doc.Endpoints...) {
		switch o.Type {
		case "direct", "block", "dns", "selector", "urltest", "":
			continue
		}
		p, err := sbToProfile(o)
		if err == nil {
			err = p.Validate()
		}
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", o.Tag, err))
			continue
		}
		p.EnsureID()
		res.Profiles = append(res.Profiles, p)
	}
	if len(res.Profiles) == 0 {
		if len(res.Errors) > 0 {
			return res, fmt.Errorf("no usable outbounds: %s", res.Errors[0])
		}
		return nil, fmt.Errorf("sing-box config has no proxy outbounds")
	}
	return res, nil
}

func sbToProfile(o sbOutbound) (*profile.Profile, error) {
	p := &profile.Profile{
		Name:   o.Tag,
		Server: o.Server,
		Port:   o.ServerPort,
		UUID:   o.UUID, Password: o.Password, Flow: o.Flow, AlterID: o.AlterID,
	}
	p.Source = fmt.Sprintf("sing-box:%s:%s:%d:%s", o.Type, o.Server, o.ServerPort, o.Tag)
	if t := o.TLS; t != nil && t.Enabled {
		p.TLS = &profile.TLS{Enabled: true, ServerName: t.ServerName, Insecure: t.Insecure, ALPN: t.ALPN}
		if t.UTLS != nil {
			p.TLS.Fingerprint = t.UTLS.Fingerprint
		}
		if t.Reality != nil && t.Reality.PublicKey != "" {
			p.TLS.Reality = &profile.Reality{PublicKey: t.Reality.PublicKey, ShortID: t.Reality.ShortID}
		}
	}
	if tr := o.Transport; tr != nil {
		pt := &profile.Transport{Type: tr.Type, Path: tr.Path, ServiceName: tr.ServiceName}
		switch h := tr.Host.(type) {
		case string:
			pt.Host = h
		case []any:
			var hosts []string
			for _, x := range h {
				hosts = append(hosts, fmt.Sprint(x))
			}
			pt.Host = strings.Join(hosts, ",")
		}
		if hv, ok := tr.Headers["Host"]; ok && pt.Host == "" {
			pt.Host = fmt.Sprint(hv)
		}
		p.Transport = pt
	}
	switch o.Type {
	case "vless":
		p.Protocol = profile.VLESS
	case "vmess":
		p.Protocol = profile.VMess
		p.Method = o.Security
	case "trojan":
		p.Protocol = profile.Trojan
	case "shadowsocks":
		p.Protocol = profile.Shadowsocks
		p.Method, p.Plugin, p.PluginOpts = o.Method, o.Plugin, o.PluginOpts
	case "hysteria2":
		p.Protocol = profile.Hysteria2
		if o.Obfs != nil && o.Obfs.Type != "" {
			p.Hysteria2 = &profile.Hy2{Obfs: o.Obfs.Type, ObfsPassword: o.Obfs.Password}
		}
	case "tuic":
		p.Protocol = profile.TUIC
		p.TUIC = &profile.TUICOpts{CongestionControl: o.CongestionControl, UDPRelayMode: o.UDPRelayMode}
	case "wireguard":
		if len(o.Peers) == 0 {
			return nil, fmt.Errorf("wireguard without peers")
		}
		peer := o.Peers[0]
		p.Protocol = profile.WireGuard
		p.Server, p.Port = peer.Address, peer.Port
		p.WireGuard = &profile.WG{
			PrivateKey: o.PrivateKey, PeerPublicKey: peer.PublicKey, PreSharedKey: peer.PreShared,
			Address: o.Address, AllowedIPs: peer.AllowedIPs, Reserved: peer.Reserved, MTU: o.MTU,
		}
	default:
		return nil, fmt.Errorf("unsupported outbound type %q", o.Type)
	}
	return p, nil
}
