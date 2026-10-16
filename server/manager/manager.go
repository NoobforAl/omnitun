// Package manager is the one place that knows how to administer an Omnitun
// server: initialise it, switch protocols, manage users, produce share links.
// The CLI, the TUI and the web API are thin front ends over it, so they all
// behave the same and return the same data (the types here are JSON-ready).
package manager

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/NoobforAl/omnitun/core/protocols"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/stats"
	"github.com/NoobforAl/omnitun/server/store"
)

// Error kinds, so front ends can pick an exit code or HTTP status.
var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
	ErrConflict = errors.New("already exists")
)

func notFound(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrNotFound, fmt.Sprintf(format, a...))
}

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, a...))
}

// Manager administers the server whose state lives in Store.
type Manager struct {
	Store *store.Store
	// Now is the clock (tests replace it).
	Now func() time.Time
}

// New returns a manager for s.
func New(s *store.Store) *Manager { return &Manager{Store: s, Now: time.Now} }

// ServerInfo describes the server.
type ServerInfo struct {
	Name        string `json:"name" doc:"Server name shown in clients"`
	PublicHost  string `json:"public_host" doc:"IP or domain clients connect to"`
	Users       int    `json:"users"`
	ActiveUsers int    `json:"active_users"`
	// SubscriptionPort is 0 when the subscription endpoint is disabled.
	SubscriptionPort int  `json:"subscription_port"`
	SelfSigned       bool `json:"self_signed_certificate"`
	Protocols        int  `json:"enabled_protocols"`
	// Running is true while `run` is serving (it writes stats every few seconds).
	Running bool `json:"running"`
	// Connections is the number of live client connections.
	Connections int `json:"connections"`
	// StartedAt is when the running server started.
	StartedAt *time.Time `json:"started_at,omitempty"`
}

// Usage is a user's traffic. Bytes are from the server's point of view:
// "up" is sent by the client, "down" is sent to it.
type Usage struct {
	MonthBytes  int64      `json:"month_bytes" doc:"Traffic this calendar month (UTC); counted against the quota"`
	MonthUp     int64      `json:"month_up_bytes"`
	MonthDown   int64      `json:"month_down_bytes"`
	TotalBytes  int64      `json:"total_bytes" doc:"All traffic since the user was created"`
	Connections int        `json:"connections" doc:"Live connections"`
	UpBps       int64      `json:"up_bps"`
	DownBps     int64      `json:"down_bps"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
}

// ProtocolInfo is one protocol's settings.
type ProtocolInfo struct {
	Key         string `json:"key" doc:"Protocol key, e.g. vless-reality"`
	Name        string `json:"name" doc:"Display name"`
	Description string `json:"description"`
	Port        int    `json:"port"`
	DefaultPort int    `json:"default_port"`
	Enabled     bool   `json:"enabled"`
	Transport   string `json:"transport" enum:"tcp,udp,tcp+udp"`
	// NeedsCertificate is true for protocols that use the server's TLS certificate.
	NeedsCertificate bool `json:"needs_certificate"`
	// ShareLink is false for protocols that travel as a config file (OpenVPN).
	ShareLink bool `json:"share_link"`
}

// UserInfo is one account without its secrets.
type UserInfo struct {
	Name      string     `json:"name"`
	Status    string     `json:"status" enum:"active,disabled,expired,over-quota"`
	Enabled   bool       `json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	QuotaGB   float64    `json:"quota_gb" doc:"Data cap in GB, 0 = unlimited"`
	Note      string     `json:"note,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Usage     Usage      `json:"usage"`
}

// Link is a share link for one protocol.
type Link struct {
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

// LinkSet is everything a user needs to connect.
type LinkSet struct {
	User string `json:"user"`
	// Subscription is the auto-updating URL for all protocols (empty if disabled).
	Subscription string `json:"subscription,omitempty"`
	Links        []Link `json:"links"`
	// OVPN lists OpenVPN config files: protocol key to download URL (empty
	// when subscriptions are off; use OVPN to get the file itself).
	OVPN map[string]string `json:"ovpn,omitempty"`
}

// InitOptions configures a new server.
type InitOptions = store.InitOptions

// Init creates a new server in the data directory.
func (m *Manager) Init(o InitOptions) (*ServerInfo, error) {
	if o.PublicHost == "" {
		return nil, invalid("a public host (IP or domain) is required")
	}
	if m.Store.Exists() {
		return nil, fmt.Errorf("%w: already initialized in %s", ErrConflict, m.Store.Dir)
	}
	st, err := store.Init(m.Store, o)
	if err != nil {
		return nil, err
	}
	return m.serverInfo(st), nil
}

// snapshot reads the traffic numbers written by the running server (empty
// when it never ran).
func (m *Manager) snapshot() stats.Snapshot {
	snap, err := stats.Read(m.Store.Dir)
	if err != nil {
		return stats.Snapshot{Users: map[string]*stats.User{}}
	}
	return snap
}

func (m *Manager) serverInfo(st *store.State) *ServerInfo {
	snap := m.snapshot()
	info := &ServerInfo{
		Running:     snap.Running(m.Now()),
		Name:        st.Server.Name,
		PublicHost:  st.Server.PublicHost,
		Users:       len(st.Users),
		ActiveUsers: len(st.ActiveUsers(m.Now())),
		SelfSigned:  st.Server.SelfSigned,
	}
	if st.Server.Subscription.Enabled {
		info.SubscriptionPort = st.Server.Subscription.Port
	}
	if info.Running {
		started := snap.StartedAt
		info.StartedAt = &started
		for _, u := range snap.Users {
			info.Connections += u.Connections
		}
	}
	for _, p := range st.Server.Protocols {
		if p.Enabled {
			info.Protocols++
		}
	}
	return info
}

// Server returns the server's summary.
func (m *Manager) Server() (*ServerInfo, error) {
	st, err := m.Store.Load()
	if err != nil {
		return nil, err
	}
	return m.serverInfo(st), nil
}

func protocolInfo(st *store.State, key string) ProtocolInfo {
	p := st.Server.Protocols[key]
	info := ProtocolInfo{Key: key, Name: key, Port: p.Port, Enabled: p.Enabled, Transport: "tcp"}
	if proto, ok := protocols.Get(key); ok {
		s := proto.Spec()
		info.Name, info.Description, info.DefaultPort = s.Name, s.Description, s.DefaultPort
		info.Transport, info.NeedsCertificate, info.ShareLink = s.Transport, s.NeedsCertificate, s.ShareLink
	}
	return info
}

// Protocols lists every protocol in display order.
func (m *Manager) Protocols() ([]ProtocolInfo, error) {
	st, err := m.Store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]ProtocolInfo, 0, len(store.AllProtocols))
	for _, k := range store.AllProtocols {
		out = append(out, protocolInfo(st, k))
	}
	return out, nil
}

// ProtocolPatch changes a protocol; nil fields are left alone.
type ProtocolPatch struct {
	Enabled *bool `json:"enabled,omitempty"`
	Port    *int  `json:"port,omitempty" minimum:"1" maximum:"65535"`
}

// SetProtocol applies a patch to one protocol.
func (m *Manager) SetProtocol(key string, patch ProtocolPatch) (*ProtocolInfo, error) {
	if !slices.Contains(store.AllProtocols, key) {
		return nil, notFound("unknown protocol %q (have: %s)", key, strings.Join(store.AllProtocols, ", "))
	}
	if patch.Port != nil && (*patch.Port < 1 || *patch.Port > 65535) {
		return nil, invalid("port %d is out of range", *patch.Port)
	}
	var out ProtocolInfo
	err := m.Store.Update(func(st *store.State) error {
		p := st.Server.Protocols[key]
		if patch.Enabled != nil {
			p.Enabled = *patch.Enabled
		}
		if patch.Port != nil {
			p.Port = *patch.Port
		}
		out = protocolInfo(st, key)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func userInfo(u *store.User, now time.Time, snap stats.Snapshot) UserInfo {
	info := UserInfo{
		Name: u.Name, Status: u.Status(now), Enabled: u.Enabled, ExpiresAt: u.ExpiresAt,
		QuotaGB: u.QuotaGB, Note: u.Note, CreatedAt: u.CreatedAt,
	}
	if s := snap.Users[u.Name]; s != nil {
		info.Usage = Usage{
			MonthBytes: s.MonthTotal(), MonthUp: s.MonthUp, MonthDown: s.MonthDown,
			TotalBytes: s.Up + s.Down, Connections: s.Connections, UpBps: s.UpBps, DownBps: s.DownBps,
		}
		if !s.LastSeen.IsZero() {
			seen := s.LastSeen
			info.Usage.LastSeen = &seen
		}
		if !snap.Running(now) { // numbers of a stopped server are not live
			info.Usage.Connections, info.Usage.UpBps, info.Usage.DownBps = 0, 0, 0
		}
	}
	return info
}

// Users lists all accounts.
func (m *Manager) Users() ([]UserInfo, error) {
	st, err := m.Store.Load()
	if err != nil {
		return nil, err
	}
	snap := m.snapshot()
	out := make([]UserInfo, 0, len(st.Users))
	for _, u := range st.Users {
		out = append(out, userInfo(u, m.Now(), snap))
	}
	return out, nil
}

// User returns one account.
func (m *Manager) User(name string) (*UserInfo, error) {
	st, err := m.Store.Load()
	if err != nil {
		return nil, err
	}
	u := st.User(name)
	if u == nil {
		return nil, notFound("no user %q", name)
	}
	info := userInfo(u, m.Now(), m.snapshot())
	return &info, nil
}

// NewUser describes an account to create.
type NewUser struct {
	Name    string  `json:"name" minLength:"1" maxLength:"32" pattern:"^[A-Za-z0-9._-]+$"`
	Days    int     `json:"days,omitempty" minimum:"0" doc:"Lifetime in days, 0 = never expires"`
	QuotaGB float64 `json:"quota_gb,omitempty" minimum:"0"`
	Note    string  `json:"note,omitempty" maxLength:"200"`
}

// AddUser creates an account with fresh credentials for every protocol.
func (m *Manager) AddUser(n NewUser) (*UserInfo, error) {
	if n.Days < 0 || n.QuotaGB < 0 {
		return nil, invalid("days and quota cannot be negative")
	}
	var out UserInfo
	err := m.Store.Update(func(st *store.State) error {
		if st.User(n.Name) != nil {
			return fmt.Errorf("%w: user %q", ErrConflict, n.Name)
		}
		u, err := store.AddUser(st, n.Name)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		if n.Days > 0 {
			t := m.Now().AddDate(0, 0, n.Days).UTC()
			u.ExpiresAt = &t
		}
		u.QuotaGB, u.Note = n.QuotaGB, n.Note
		out = userInfo(u, m.Now(), stats.Snapshot{})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UserPatch changes an account; nil fields are left alone.
type UserPatch struct {
	Enabled *bool `json:"enabled,omitempty"`
	// ExpireDays sets the expiry to now plus this many days; 0 removes the expiry.
	ExpireDays *int     `json:"expire_days,omitempty" minimum:"0"`
	QuotaGB    *float64 `json:"quota_gb,omitempty" minimum:"0"`
	Note       *string  `json:"note,omitempty" maxLength:"200"`
}

// UpdateUser applies a patch to an account.
func (m *Manager) UpdateUser(name string, p UserPatch) (*UserInfo, error) {
	if (p.ExpireDays != nil && *p.ExpireDays < 0) || (p.QuotaGB != nil && *p.QuotaGB < 0) {
		return nil, invalid("days and quota cannot be negative")
	}
	var out UserInfo
	snap := m.snapshot()
	err := m.Store.Update(func(st *store.State) error {
		u := st.User(name)
		if u == nil {
			return notFound("no user %q", name)
		}
		if p.Enabled != nil {
			u.Enabled = *p.Enabled
		}
		if p.ExpireDays != nil {
			if *p.ExpireDays == 0 {
				u.ExpiresAt = nil
			} else {
				t := m.Now().AddDate(0, 0, *p.ExpireDays).UTC()
				u.ExpiresAt = &t
			}
		}
		if p.QuotaGB != nil {
			u.QuotaGB = *p.QuotaGB
			// Apply the new cap at once; the running server re-checks it too.
			used := float64(monthTotal(snap, u.Name))
			u.QuotaExceeded = u.QuotaGB > 0 && used >= u.QuotaGB*stats.GiB
		}
		if p.Note != nil {
			u.Note = *p.Note
		}
		out = userInfo(u, m.Now(), snap)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteUser removes an account.
func (m *Manager) DeleteUser(name string) error {
	return m.Store.Update(func(st *store.State) error {
		if err := store.DeleteUser(st, name); err != nil {
			return notFound("no user %q", name)
		}
		return nil
	})
}

// ResetUser issues new credentials; the old links stop working.
func (m *Manager) ResetUser(name string) (*UserInfo, error) {
	var out UserInfo
	err := m.Store.Update(func(st *store.State) error {
		u := st.User(name)
		if u == nil {
			return notFound("no user %q", name)
		}
		store.ResetCredentials(u)
		out = userInfo(u, m.Now(), m.snapshot())
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ResetUsage restarts a user's count for this month (lifetime totals stay)
// and lets them back in if they were blocked for exceeding the quota. A
// running server applies it within seconds.
func (m *Manager) ResetUsage(name string) (*UserInfo, error) {
	var out UserInfo
	err := m.Store.Update(func(st *store.State) error {
		u := st.User(name)
		if u == nil {
			return notFound("no user %q", name)
		}
		now := m.Now().UTC()
		u.UsageResetAt = &now
		u.QuotaExceeded = false
		out = userInfo(u, m.Now(), m.snapshot())
		out.Usage.MonthBytes, out.Usage.MonthUp, out.Usage.MonthDown = 0, 0, 0
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Links returns the user's share links.
func (m *Manager) Links(name string) (*LinkSet, error) {
	st, err := m.Store.Load()
	if err != nil {
		return nil, err
	}
	u := st.User(name)
	if u == nil {
		return nil, notFound("no user %q", name)
	}
	set := &LinkSet{User: u.Name, Subscription: serverconf.SubscriptionURL(st, u), Links: []Link{}}
	for _, l := range serverconf.Links(st, u) {
		set.Links = append(set.Links, Link{Protocol: l.Protocol, URL: l.URL})
	}
	for _, proto := range []string{store.OpenVPNUDP, store.OpenVPNTCP} {
		if p := st.Server.Protocols[proto]; p != nil && p.Enabled {
			if set.OVPN == nil {
				set.OVPN = map[string]string{}
			}
			set.OVPN[proto] = serverconf.OVPNURL(st, u, proto)
		}
	}
	return set, nil
}

// OVPN returns the user's OpenVPN config file for proto ("openvpn-udp" or
// "openvpn-tcp"; "udp" and "tcp" are accepted).
func (m *Manager) OVPN(name, proto string) (string, error) {
	st, err := m.Store.Load()
	if err != nil {
		return "", err
	}
	u := st.User(name)
	if u == nil {
		return "", notFound("no user %q", name)
	}
	proto = "openvpn-" + strings.TrimPrefix(proto, "openvpn-")
	if !store.IsOpenVPN(proto) {
		return "", invalid("protocol must be udp or tcp")
	}
	text, err := serverconf.OVPN(st, u, proto)
	if err != nil {
		return "", invalid("%v", err)
	}
	return text, nil
}

// API returns the web API settings, token included.
func (m *Manager) API() (store.API, error) {
	st, err := m.Store.Load()
	if err != nil {
		return store.API{}, err
	}
	return st.Server.API, nil
}

// APIPatch changes the web API settings; nil fields are left alone.
type APIPatch struct {
	Enabled *bool
	Listen  *string
	Port    *int
	// NewToken issues a fresh token.
	NewToken bool
}

// SetAPI updates the web API settings.
func (m *Manager) SetAPI(p APIPatch) (store.API, error) {
	var out store.API
	err := m.Store.Update(func(st *store.State) error {
		a := &st.Server.API
		if p.Enabled != nil {
			a.Enabled = *p.Enabled
		}
		if p.Listen != nil {
			a.Listen = *p.Listen
		}
		if p.Port != nil {
			if *p.Port < 1 || *p.Port > 65535 {
				return invalid("port %d is out of range", *p.Port)
			}
			a.Port = *p.Port
		}
		if p.NewToken {
			a.Token = store.RandomHex(24)
		}
		out = *a
		return nil
	})
	return out, err
}

func monthTotal(snap stats.Snapshot, name string) int64 {
	if u := snap.Users[name]; u != nil {
		return u.MonthTotal()
	}
	return 0
}
