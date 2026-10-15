// Package profile defines the protocol-neutral connection profile used across
// the client, the engine and the server CLI.
package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Protocol identifiers.
const (
	VLESS       = "vless"
	VMess       = "vmess"
	Trojan      = "trojan"
	Shadowsocks = "shadowsocks"
	Hysteria2   = "hysteria2"
	TUIC        = "tuic"
	WireGuard   = "wireguard"
	AmneziaWG   = "amneziawg"
	Naive       = "naive"
	Tor         = "tor"
)

// Profile is one connectable server configuration.
type Profile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Server   string `json:"server"`
	Port     int    `json:"port"`

	// Credentials. Which fields are used depends on Protocol.
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
	Username string `json:"username,omitempty"`
	Method   string `json:"method,omitempty"` // shadowsocks cipher, vmess security
	Flow     string `json:"flow,omitempty"`   // vless flow, e.g. xtls-rprx-vision
	AlterID  int    `json:"alter_id,omitempty"`

	TLS        *TLS       `json:"tls,omitempty"`
	Transport  *Transport `json:"transport,omitempty"`
	WireGuard  *WG        `json:"wireguard,omitempty"`
	Hysteria2  *Hy2       `json:"hysteria2,omitempty"`
	TUIC       *TUICOpts  `json:"tuic,omitempty"`
	Plugin     string     `json:"plugin,omitempty"` // shadowsocks SIP003 plugin
	PluginOpts string     `json:"plugin_opts,omitempty"`

	// Source is the original link or file text the profile came from.
	Source string `json:"source,omitempty"`
}

// TLS settings shared by TLS-based protocols.
type TLS struct {
	Enabled     bool     `json:"enabled"`
	ServerName  string   `json:"server_name,omitempty"`
	ALPN        []string `json:"alpn,omitempty"`
	Insecure    bool     `json:"insecure,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"` // uTLS fingerprint
	Reality     *Reality `json:"reality,omitempty"`
}

// Reality settings for VLESS+Reality.
type Reality struct {
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id,omitempty"`
}

// Transport is the V2Ray-style transport layer (ws, grpc, http, httpupgrade).
type Transport struct {
	Type        string `json:"type"`
	Path        string `json:"path,omitempty"`
	Host        string `json:"host,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
}

// WG holds WireGuard and AmneziaWG settings.
type WG struct {
	PrivateKey    string   `json:"private_key"`
	PeerPublicKey string   `json:"peer_public_key"`
	PreSharedKey  string   `json:"pre_shared_key,omitempty"`
	Address       []string `json:"address"`
	AllowedIPs    []string `json:"allowed_ips,omitempty"`
	DNS           []string `json:"dns,omitempty"`
	MTU           int      `json:"mtu,omitempty"`
	Reserved      []int    `json:"reserved,omitempty"`
	Keepalive     int      `json:"keepalive,omitempty"`
	// Amnezia holds AmneziaWG obfuscation parameters (Jc, Jmin, Jmax, S1, S2, H1-H4).
	Amnezia map[string]string `json:"amnezia,omitempty"`
}

// Hy2 holds Hysteria2 options.
type Hy2 struct {
	Obfs         string `json:"obfs,omitempty"`
	ObfsPassword string `json:"obfs_password,omitempty"`
	UpMbps       int    `json:"up_mbps,omitempty"`
	DownMbps     int    `json:"down_mbps,omitempty"`
}

// TUICOpts holds TUIC v5 options.
type TUICOpts struct {
	CongestionControl string `json:"congestion_control,omitempty"`
	UDPRelayMode      string `json:"udp_relay_mode,omitempty"`
}

// Validate checks the fields every protocol needs.
func (p *Profile) Validate() error {
	if p.Protocol == "" {
		return fmt.Errorf("profile %q: missing protocol", p.Name)
	}
	if p.Server == "" {
		return fmt.Errorf("profile %q: missing server", p.Name)
	}
	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("profile %q: invalid port %d", p.Name, p.Port)
	}
	switch p.Protocol {
	case VLESS, VMess:
		if p.UUID == "" {
			return fmt.Errorf("profile %q: missing uuid", p.Name)
		}
	case Trojan, Hysteria2:
		if p.Password == "" {
			return fmt.Errorf("profile %q: missing password", p.Name)
		}
	case Shadowsocks:
		if p.Method == "" || p.Password == "" {
			return fmt.Errorf("profile %q: missing method or password", p.Name)
		}
	case TUIC:
		if p.UUID == "" {
			return fmt.Errorf("profile %q: missing uuid", p.Name)
		}
	case WireGuard, AmneziaWG:
		if p.WireGuard == nil || p.WireGuard.PrivateKey == "" || p.WireGuard.PeerPublicKey == "" {
			return fmt.Errorf("profile %q: missing wireguard keys", p.Name)
		}
		if len(p.WireGuard.Address) == 0 {
			return fmt.Errorf("profile %q: missing wireguard address", p.Name)
		}
	}
	return nil
}

// EnsureID assigns a stable ID derived from the connection parameters when none is set.
func (p *Profile) EnsureID() {
	if p.ID != "" {
		return
	}
	key := p.Source
	if key == "" {
		key = fmt.Sprintf("%s|%s|%d|%s|%s", p.Protocol, p.Server, p.Port, p.UUID, p.Password)
	}
	sum := sha256.Sum256([]byte(key))
	p.ID = hex.EncodeToString(sum[:6])
}

// AsSlice wraps p in a one-element slice.
func (p *Profile) AsSlice() []*Profile { return []*Profile{p} }

// DisplayName returns Name or a fallback built from server and port.
func (p *Profile) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("%s:%d", p.Server, p.Port)
}
