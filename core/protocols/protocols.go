// Package protocols is the one place that knows each VPN protocol Omnitun
// serves and connects to. A protocol is defined once and gives:
//
//   - Spec: name, transport, default port and options, for forms, tables,
//     wizards and docs;
//   - Inbounds: the server side, as sing-box configuration for a set of users;
//   - Profile: the client side, the profile a given user connects with
//     (links.Export turns it into a share link, singbox.Build into config).
//
// The server builds its configuration and its users' links from this, and the
// clients read the same Specs, so creating a server and connecting to it use
// one schema and a new protocol is added in one place.
package protocols

import (
	"fmt"
	"net"
	"strconv"

	"github.com/NoobforAl/omnitun/core/profile"
)

// Option describes a setting of a protocol, for building forms and wizards.
type Option struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Help    string `json:"help,omitempty"`
	Kind    string `json:"kind" enum:"string,int,bool"`
	Default string `json:"default,omitempty"`
}

// Spec describes a protocol.
type Spec struct {
	// Key identifies the protocol in the server's state and API.
	Key string `json:"key"`
	// Name is the display name; Label is the short name used in link and
	// profile names ("<server> <label>").
	Name  string `json:"name"`
	Label string `json:"label"`
	// Profile is the client-side protocol (profile.Protocol) it connects as.
	Profile     string `json:"profile"`
	Description string `json:"description"`
	// Transport is "tcp", "udp" or "tcp+udp" (what a firewall must allow).
	Transport   string `json:"transport" enum:"tcp,udp,tcp+udp"`
	DefaultPort int    `json:"default_port"`
	// NeedsCertificate marks protocols that use the server's TLS certificate.
	NeedsCertificate bool `json:"needs_certificate"`
	// ShareLink is false for protocols that travel as a config file (OpenVPN).
	ShareLink bool     `json:"share_link"`
	Options   []Option `json:"options,omitempty"`
}

// Reality is the VLESS+Reality key material and the site it impersonates.
type Reality struct {
	PrivateKey, PublicKey, ShortID string
	// Handshake is host[:port] of the real TLS site.
	Handshake string
}

// OpenVPN is the server's OpenVPN PKI.
type OpenVPN struct {
	CA, ServerCert, ServerKey, TLSCrypt string
}

// Env is the server a protocol is rendered for.
type Env struct {
	Name string
	// Host is the public IP or domain clients connect to.
	Host string
	// CertPath and KeyPath are the TLS certificate and key files.
	CertPath, KeyPath string
	// CertPEM is the certificate's content (pinned by clients for SelfSigned).
	CertPEM string
	// CertProvider is the tag of a sing-box certificate provider (ACME); when
	// set it replaces CertPath and KeyPath.
	CertProvider string
	// SelfSigned is true only when clients must be told to skip verification.
	SelfSigned bool
	Reality    Reality
	// SSServerKey is the Shadowsocks-2022 server key (base64).
	SSServerKey string
	OpenVPN     OpenVPN
}

// Account is one user's credentials; every protocol uses the ones it needs.
type Account struct {
	Name, UUID, Password, SSKey string
}

// Config is the sing-box configuration a protocol adds to the server.
type Config struct {
	Inbounds  []map[string]any
	Endpoints []map[string]any
}

// Protocol is implemented once per protocol.
type Protocol interface {
	Spec() Spec
	// Inbounds returns the server side for users (at least one) on port.
	Inbounds(env Env, port int, users []Account) Config
	// Profile returns what user u connects with.
	Profile(env Env, port int, u Account) (*profile.Profile, error)
}

// FileProtocol is implemented by protocols that hand users a config file
// instead of (or besides) a share link.
type FileProtocol interface {
	// File returns the user's config file name and content.
	File(env Env, port int, u Account) (name, content string, err error)
}

var (
	registry []Protocol
	byKey    = map[string]Protocol{}
)

func register(p Protocol) {
	registry = append(registry, p)
	byKey[p.Spec().Key] = p
}

// All returns every protocol in display (and failover) order.
func All() []Protocol { return append([]Protocol(nil), registry...) }

// Get returns the protocol with the given key.
func Get(key string) (Protocol, bool) {
	p, ok := byKey[key]
	return p, ok
}

// Keys lists every protocol key in display order.
func Keys() []string {
	keys := make([]string, len(registry))
	for i, p := range registry {
		keys[i] = p.Spec().Key
	}
	return keys
}

// Specs lists every protocol's Spec in display order.
func Specs() []Spec {
	specs := make([]Spec, len(registry))
	for i, p := range registry {
		specs[i] = p.Spec()
	}
	return specs
}

// DefaultPorts maps each protocol key to its default port.
func DefaultPorts() map[string]int {
	m := make(map[string]int, len(registry))
	for _, p := range registry {
		s := p.Spec()
		m[s.Key] = s.DefaultPort
	}
	return m
}

// SplitHostPort splits host[:port], using def when there is no valid port.
func SplitHostPort(hp string, def int) (string, int) {
	host, p, err := net.SplitHostPort(hp)
	if err != nil {
		return hp, def
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return host, def
	}
	return host, port
}

// profileName is the display name of a user's profile for a protocol.
func profileName(env Env, s Spec) string { return fmt.Sprintf("%s %s", env.Name, s.Label) }

// users maps accounts to sing-box user entries.
func users(accts []Account, f func(Account) map[string]any) []any {
	out := make([]any, 0, len(accts))
	for _, a := range accts {
		out = append(out, f(a))
	}
	return out
}

// tlsBlock is the inbound TLS section from the server's certificate.
func tlsBlock(env Env, extra map[string]any) map[string]any {
	t := map[string]any{"enabled": true}
	if env.CertProvider != "" {
		t["certificate_provider"] = env.CertProvider
	} else {
		t["certificate_path"] = env.CertPath
		t["key_path"] = env.KeyPath
	}
	for k, v := range extra {
		t[k] = v
	}
	return t
}

// serverTLS is the client-side TLS block for protocols that verify the
// server's certificate by name (and skip it when self-signed).
func serverTLS(env Env, alpn ...string) *profile.TLS {
	return &profile.TLS{Enabled: true, ServerName: env.Host, Insecure: env.SelfSigned, ALPN: alpn}
}
