// Package serverconf turns the server state into a sing-box server config and
// into per-user share links and client profiles. The protocols themselves are
// defined in core/protocols; this package only feeds them the server's keys
// and users.
package serverconf

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/NoobforAl/omnitun/core/links"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/protocols"
	"github.com/NoobforAl/omnitun/server/store"
)

type obj = map[string]any

// Env is the server as the protocol registry sees it. s resolves the
// certificate paths against the data directory; the certificate's content is
// read for clients that pin it.
func Env(st *store.State, s *store.Store) protocols.Env {
	srv := st.Server
	env := protocols.Env{
		Name: srv.Name, Host: srv.PublicHost,
		CertPath: s.Path(srv.CertFile), KeyPath: s.Path(srv.KeyFile),
		SelfSigned: srv.SelfSigned,
		Reality: protocols.Reality{
			PrivateKey: srv.Reality.PrivateKey, PublicKey: srv.Reality.PublicKey,
			ShortID: srv.Reality.ShortID, Handshake: srv.Reality.Handshake,
		},
		SSServerKey: srv.SSServerKey,
		OpenVPN: protocols.OpenVPN{
			CA: srv.OpenVPN.CA, ServerCert: srv.OpenVPN.ServerCert,
			ServerKey: srv.OpenVPN.ServerKey, TLSCrypt: srv.OpenVPN.TLSCrypt,
		},
	}
	if data, err := os.ReadFile(env.CertPath); err == nil {
		env.CertPEM = string(data)
	}
	if srv.ACMEActive() {
		// A real certificate: clients verify it normally and nothing is pinned.
		env.CertProvider, env.SelfSigned, env.CertPEM = ACMETag, false, ""
	}
	return env
}

// ACMETag is the tag of the ACME certificate provider in the sing-box config.
const ACMETag = "acme"

// ACMEProvider is the sing-box certificate provider that obtains and renews
// the domain's certificate; its data (account, certificates) lives in the
// data directory's "acme" folder.
func ACMEProvider(st *store.State, s *store.Store) obj {
	a := st.Server.ACME
	p := obj{
		"type": "acme", "tag": ACMETag, "domain": []string{a.Domain}, "default_server_name": a.Domain,
		"data_directory": ACMEDir(s),
	}
	if a.Email != "" {
		p["email"] = a.Email
	}
	if a.Provider != "" && a.Provider != "letsencrypt" {
		p["provider"] = a.Provider
	}
	if a.HTTPPort > 0 {
		p["alternative_http_port"] = a.HTTPPort
	}
	switch a.DNSProvider {
	case "cloudflare":
		p["dns01_challenge"] = obj{"provider": "cloudflare", "api_token": a.DNSToken}
	case "alidns":
		p["dns01_challenge"] = obj{"provider": "alidns", "access_key_id": a.DNSToken, "access_key_secret": a.DNSSecret}
	}
	return p
}

// ACMEDir is where the ACME account and certificates are kept.
func ACMEDir(s *store.Store) string { return s.Path("acme") }

// Account maps a user to the credentials the protocols use.
func Account(u *store.User) protocols.Account {
	return protocols.Account{Name: u.Name, UUID: u.UUID, Password: u.Password, SSKey: u.SSKey}
}

func accounts(users []*store.User) []protocols.Account {
	out := make([]protocols.Account, len(users))
	for i, u := range users {
		out[i] = Account(u)
	}
	return out
}

// Build returns the sing-box config for the active users. Paths are
// resolved against s.Dir. Protocols with no active users are left out
// (sing-box needs at least one user per inbound).
func Build(st *store.State, s *store.Store, now time.Time) obj {
	users := st.ActiveUsers(now)
	env := Env(st, s)
	var inbounds, endpoints []any
	for _, key := range store.AllProtocols {
		p := st.Server.Protocols[key]
		proto, ok := protocols.Get(key)
		if p == nil || !p.Enabled || len(users) == 0 || !ok {
			continue
		}
		cfg := proto.Inbounds(env, p.Port, accounts(users))
		for _, in := range cfg.Inbounds {
			inbounds = append(inbounds, in)
		}
		for _, ep := range cfg.Endpoints {
			endpoints = append(endpoints, ep)
		}
	}
	cfg := obj{
		"log":       obj{"level": "warn", "timestamp": true},
		"inbounds":  inbounds,
		"outbounds": []any{obj{"type": "direct", "tag": "direct"}},
	}
	if env.CertProvider != "" && usesCertificate(st) {
		cfg["certificate_providers"] = []any{ACMEProvider(st, s)}
	}
	if len(endpoints) > 0 {
		cfg["endpoints"] = endpoints
	}
	return cfg
}

// usesCertificate reports whether an enabled protocol needs the TLS certificate.
func usesCertificate(st *store.State) bool {
	for _, key := range store.AllProtocols {
		p, ok := protocols.Get(key)
		if ok && st.Server.Protocols[key] != nil && st.Server.Protocols[key].Enabled && p.Spec().NeedsCertificate {
			return true
		}
	}
	return false
}

// OVPN returns a single-file OpenVPN client config for the user, with the
// credentials inline (OpenVPN 2.6+, Omnitun). proto is store.OpenVPNUDP or store.OpenVPNTCP.
func OVPN(st *store.State, u *store.User, proto string) (string, error) {
	p := st.Server.Protocols[proto]
	pr, ok := protocols.Get(proto)
	if !store.IsOpenVPN(proto) || p == nil || !ok {
		return "", fmt.Errorf("unknown OpenVPN protocol %q", proto)
	}
	if !p.Enabled {
		return "", fmt.Errorf("%s is disabled (enable it with: proto enable %s)", proto, proto)
	}
	fp, ok := pr.(protocols.FileProtocol)
	if !ok {
		return "", fmt.Errorf("%s has no config file", proto)
	}
	_, content, err := fp.File(protocols.Env{
		Name: st.Server.Name, Host: st.Server.PublicHost,
		OpenVPN: protocols.OpenVPN{CA: st.Server.OpenVPN.CA, TLSCrypt: st.Server.OpenVPN.TLSCrypt},
	}, p.Port, Account(u))
	return content, err
}

// Profiles returns the client profiles of every enabled protocol for the
// user, in failover order (including OpenVPN, which has no share link).
func Profiles(st *store.State, s *store.Store, u *store.User) ([]*profile.Profile, error) {
	env := Env(st, s)
	var out []*profile.Profile
	for _, key := range store.AllProtocols {
		p := st.Server.Protocols[key]
		proto, ok := protocols.Get(key)
		if p == nil || !p.Enabled || !ok {
			continue
		}
		prof, err := proto.Profile(env, p.Port, Account(u))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out = append(out, prof)
	}
	return out, nil
}

// Link is one share link for a user.
type Link struct {
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

// Links returns the user's share links for every enabled protocol, in
// failover-friendly order (Reality first, UDP protocols last). Protocols
// without a share link (OpenVPN) are left out; see OVPN and OVPNURL.
func Links(st *store.State, u *store.User) []Link {
	env := protocols.Env{
		Name: st.Server.Name, Host: st.Server.PublicHost, SelfSigned: st.Server.SelfSigned && !st.Server.ACMEActive(),
		Reality: protocols.Reality{
			PublicKey: st.Server.Reality.PublicKey, ShortID: st.Server.Reality.ShortID, Handshake: st.Server.Reality.Handshake,
		},
		SSServerKey: st.Server.SSServerKey,
	}
	var out []Link
	for _, key := range store.AllProtocols {
		p := st.Server.Protocols[key]
		proto, ok := protocols.Get(key)
		if p == nil || !p.Enabled || !ok || !proto.Spec().ShareLink {
			continue
		}
		prof, err := proto.Profile(env, p.Port, Account(u))
		if err != nil {
			continue
		}
		link, err := links.Export(prof)
		if err != nil {
			continue
		}
		out = append(out, Link{Protocol: key, URL: link})
	}
	return out
}

// OVPNURL is where the user downloads the .ovpn file for proto, or "" if
// subscriptions are disabled.
func OVPNURL(st *store.State, u *store.User, proto string) string {
	sub := SubscriptionURL(st, u)
	if sub == "" {
		return ""
	}
	return sub + "/" + proto + ".ovpn"
}

// SubscriptionURL is the user's subscription link, or "" if disabled.
func SubscriptionURL(st *store.State, u *store.User) string {
	sub := st.Server.Subscription
	if !sub.Enabled {
		return ""
	}
	scheme := "http"
	if st.Server.ACMEActive() {
		scheme = "https" // the subscription endpoint serves the real certificate too
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, net.JoinHostPort(st.Server.PublicHost, strconv.Itoa(sub.Port)), sub.Path, u.SubToken)
}
