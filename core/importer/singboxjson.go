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
		Outbounds []json.RawMessage `json:"outbounds"`
		Endpoints []json.RawMessage `json:"endpoints"`
	}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf("sing-box JSON: %w", err)
	}
	res := &Result{}
	all := append(doc.Outbounds, doc.Endpoints...)
	// ShadowTLS outbounds are layers under Shadowsocks ones (via "detour").
	shadowTLS := map[string]sbShadowTLS{}
	for _, raw := range all {
		var st sbShadowTLS
		if json.Unmarshal(raw, &st) == nil && st.Type == "shadowtls" {
			shadowTLS[st.Tag] = st
		}
	}
	for _, raw := range all {
		var head struct {
			Type   string `json:"type"`
			Tag    string `json:"tag"`
			Detour string `json:"detour"`
		}
		if err := json.Unmarshal(raw, &head); err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		switch head.Type {
		case "direct", "block", "dns", "selector", "urltest", "shadowtls", "":
			continue
		}
		var p *profile.Profile
		var err error
		switch head.Type {
		case "openvpn-client":
			p, err = sbOpenVPNToProfile(raw)
		case "hysteria":
			p, err = sbHysteriaToProfile(raw)
		case "ssh":
			p, err = sbSSHToProfile(raw)
		default:
			var o sbOutbound
			if err = json.Unmarshal(raw, &o); err == nil {
				p, err = sbToProfile(o)
			}
		}
		if err == nil && head.Detour != "" {
			if st, ok := shadowTLS[head.Detour]; ok && p.Protocol == profile.Shadowsocks {
				// The ShadowTLS server is the address clients actually dial.
				p.Server, p.Port = st.Server, st.ServerPort
				p.Plugin = "shadow-tls"
				p.PluginOpts = fmt.Sprintf("host=%s;password=%s;version=%d", st.TLS.ServerName, st.Password, max(st.Version, 1))
				if st.TLS.Insecure {
					p.PluginOpts += ";insecure=1"
				}
			} else {
				err = fmt.Errorf("detour %q is not supported", head.Detour)
			}
		}
		if err == nil {
			err = p.Validate()
		}
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", head.Tag, err))
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
	case "anytls":
		p.Protocol = profile.AnyTLS
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

// sbOpenVPN is sing-box's openvpn-client endpoint (the fields Omnitun maps).
type sbOpenVPN struct {
	Tag        string `json:"tag"`
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
	Network    string `json:"network"`
	Servers    []struct {
		Server     string `json:"server"`
		ServerPort int    `json:"server_port"`
		Network    string `json:"network"`
	} `json:"servers"`
	RemoteRandom bool   `json:"remote_random"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	TLS          struct {
		Certificate          stringList `json:"certificate"`
		ClientCertificate    stringList `json:"client_certificate"`
		ClientKey            stringList `json:"client_key"`
		ServerName           string     `json:"server_name"`
		ServerNameType       string     `json:"server_name_type"`
		RemoteCertificateTLS string     `json:"remote_certificate_tls"`
		VersionMin           string     `json:"version_min"`
		Cipher               string     `json:"cipher"`
		ControlWrap          *struct {
			Type      string     `json:"type"`
			Key       stringList `json:"key"`
			Direction string     `json:"direction"`
		} `json:"control_wrap"`
	} `json:"tls"`
	DataCiphers         stringList `json:"data_ciphers"`
	DataCiphersFallback string     `json:"data_ciphers_fallback"`
	Auth                string     `json:"auth"`
	Compression         string     `json:"compression"`
	CompressionLZO      string     `json:"compression_lzo"`
	MTU                 int        `json:"mtu"`
	MSSFix              int        `json:"mss_fix"`
}

// stringList accepts a JSON string or an array of strings (sing-box "Listable").
type stringList []string

func (l *stringList) UnmarshalJSON(b []byte) error {
	var one string
	if json.Unmarshal(b, &one) == nil {
		*l = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*l = many
	return nil
}

func (l stringList) joined() string { return strings.Join(l, "\n") }

func sbOpenVPNToProfile(raw json.RawMessage) (*profile.Profile, error) {
	var v sbOpenVPN
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	if v.TLS.Certificate == nil {
		return nil, fmt.Errorf("only certificate-based (TLS mode) OpenVPN is supported")
	}
	o := &profile.OVPN{
		Network: orDefault(v.Network, "udp"), RemoteRandom: v.RemoteRandom,
		CA: v.TLS.Certificate.joined(), Cert: v.TLS.ClientCertificate.joined(), Key: v.TLS.ClientKey.joined(),
		VerifyX509Name: v.TLS.ServerName, VerifyX509Type: v.TLS.ServerNameType,
		RemoteCertTLS: v.TLS.RemoteCertificateTLS, TLSVersionMin: v.TLS.VersionMin, TLSCipher: v.TLS.Cipher,
		DataCiphers: v.DataCiphers, DataCiphersFallback: v.DataCiphersFallback, Auth: v.Auth,
		Compression: v.Compression, CompressionLZO: v.CompressionLZO, MTU: v.MTU, MSSFix: v.MSSFix,
		NeedsAuth: v.Username != "",
	}
	if cw := v.TLS.ControlWrap; cw != nil && cw.Type != "" {
		o.ControlWrap, o.ControlWrapKey = strings.ReplaceAll(cw.Type, "_", "-"), cw.Key.joined()
		switch cw.Direction {
		case "client":
			o.KeyDirection = "1"
		case "server":
			o.KeyDirection = "0"
		}
	}
	p := &profile.Profile{
		Name: v.Tag, Protocol: profile.OpenVPN, Server: v.Server, Port: v.ServerPort,
		Username: v.Username, Password: v.Password, OpenVPN: o,
	}
	for i, r := range v.Servers {
		if i == 0 && p.Server == "" {
			p.Server, p.Port, o.Network = r.Server, r.ServerPort, orDefault(r.Network, "udp")
			continue
		}
		o.Remotes = append(o.Remotes, profile.OVPNRemote{Server: r.Server, Port: r.ServerPort, Network: orDefault(r.Network, "udp")})
	}
	p.Source = fmt.Sprintf("sing-box:openvpn:%s:%d:%s", p.Server, p.Port, p.Name)
	return p, nil
}

type sbShadowTLS struct {
	Type       string `json:"type"`
	Tag        string `json:"tag"`
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
	Version    int    `json:"version"`
	Password   string `json:"password"`
	TLS        struct {
		ServerName string `json:"server_name"`
		Insecure   bool   `json:"insecure"`
	} `json:"tls"`
}

func sbHysteriaToProfile(raw json.RawMessage) (*profile.Profile, error) {
	var v struct {
		Tag        string `json:"tag"`
		Server     string `json:"server"`
		ServerPort int    `json:"server_port"`
		AuthStr    string `json:"auth_str"`
		Obfs       string `json:"obfs"`
		UpMbps     int    `json:"up_mbps"`
		DownMbps   int    `json:"down_mbps"`
		TLS        *sbTLS `json:"tls"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	p := &profile.Profile{
		Name: v.Tag, Protocol: profile.Hysteria, Server: v.Server, Port: v.ServerPort, Password: v.AuthStr,
		Hysteria2: &profile.Hy2{Obfs: v.Obfs, UpMbps: v.UpMbps, DownMbps: v.DownMbps},
		TLS:       &profile.TLS{Enabled: true},
	}
	if t := v.TLS; t != nil {
		p.TLS.ServerName, p.TLS.Insecure, p.TLS.ALPN = t.ServerName, t.Insecure, t.ALPN
	}
	p.Source = fmt.Sprintf("sing-box:hysteria:%s:%d:%s", p.Server, p.Port, p.Name)
	return p, nil
}

func sbSSHToProfile(raw json.RawMessage) (*profile.Profile, error) {
	var v struct {
		Tag                  string     `json:"tag"`
		Server               string     `json:"server"`
		ServerPort           int        `json:"server_port"`
		User                 string     `json:"user"`
		Password             string     `json:"password"`
		PrivateKey           stringList `json:"private_key"`
		PrivateKeyPassphrase string     `json:"private_key_passphrase"`
		HostKey              stringList `json:"host_key"`
		ClientVersion        string     `json:"client_version"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	port := v.ServerPort
	if port == 0 {
		port = 22
	}
	p := &profile.Profile{Name: v.Tag, Protocol: profile.SSH, Server: v.Server, Port: port, Username: v.User, Password: v.Password}
	if len(v.PrivateKey) > 0 || len(v.HostKey) > 0 || v.ClientVersion != "" {
		p.SSH = &profile.SSHOpts{
			PrivateKey: v.PrivateKey.joined(), PrivateKeyPassphrase: v.PrivateKeyPassphrase,
			HostKey: v.HostKey, ClientVersion: v.ClientVersion,
		}
	}
	p.Source = fmt.Sprintf("sing-box:ssh:%s:%d:%s", p.Server, p.Port, p.Name)
	return p, nil
}
