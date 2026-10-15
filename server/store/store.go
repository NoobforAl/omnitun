// Package store keeps the server's state (settings, keys, users) in one JSON
// file in the data directory. Writes are atomic (temp file + rename), so the
// CLI, the TUI and a running server can share it; the server reloads it when
// it changes.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"
)

const fileName = "state.json"

// Protocol keys.
const (
	VLESSReality = "vless-reality"
	Shadowsocks  = "shadowsocks"
	Trojan       = "trojan"
	Hysteria2    = "hysteria2"
	TUIC         = "tuic"
	AnyTLS       = "anytls"
	ShadowTLS    = "shadowtls" // ShadowTLS v3 in front of Shadowsocks-2022
	Naive        = "naive"     // NaiveProxy (HTTP/2 CONNECT over TLS)
	OpenVPNUDP   = "openvpn-udp"
	OpenVPNTCP   = "openvpn-tcp"
)

// AllProtocols in display order.
var AllProtocols = []string{VLESSReality, Shadowsocks, Trojan, Hysteria2, TUIC, AnyTLS, ShadowTLS, Naive, OpenVPNUDP, OpenVPNTCP}

// IsOpenVPN reports whether a protocol key is one of the OpenVPN variants.
func IsOpenVPN(proto string) bool { return proto == OpenVPNUDP || proto == OpenVPNTCP }

// Proto is one inbound.
type Proto struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// Reality holds the VLESS+Reality key material and the site it impersonates.
type Reality struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	ShortID    string `json:"short_id"`
	// Handshake is the real TLS site whose certificate unauthenticated
	// visitors see, host[:port].
	Handshake string `json:"handshake"`
}

// Subscription is the HTTP endpoint clients refresh from.
type Subscription struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
	Port    int    `json:"port"`
	// Path is a random prefix so the endpoint isn't trivially discoverable.
	Path string `json:"path"`
}

// Server settings.
type Server struct {
	Name string `json:"name"`
	// PublicHost is the IP or domain clients connect to.
	PublicHost string            `json:"public_host"`
	Protocols  map[string]*Proto `json:"protocols"`
	Reality    Reality           `json:"reality"`
	// SSServerKey is the Shadowsocks-2022 server key (base64, 16 bytes).
	SSServerKey string `json:"ss_server_key"`
	// CertFile/KeyFile are the TLS certificate for Trojan, Hysteria2 and TUIC,
	// relative to the data directory. Self-signed unless replaced.
	CertFile     string       `json:"cert_file"`
	KeyFile      string       `json:"key_file"`
	SelfSigned   bool         `json:"self_signed"`
	Subscription Subscription `json:"subscription"`
	OpenVPN      OpenVPN      `json:"openvpn"`
}

// User is one account. Every protocol uses the same user with its own credentials.
type User struct {
	Name      string     `json:"name"`
	UUID      string     `json:"uuid"`
	Password  string     `json:"password"`
	SSKey     string     `json:"ss_key"`
	SubToken  string     `json:"sub_token"`
	Enabled   bool       `json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// QuotaGB is the data cap reported to clients; 0 = unlimited.
	QuotaGB   float64   `json:"quota_gb,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Active reports whether the user may connect now.
func (u *User) Active(now time.Time) bool {
	return u.Enabled && (u.ExpiresAt == nil || now.Before(*u.ExpiresAt))
}

// Status is "active", "disabled" or "expired".
func (u *User) Status(now time.Time) string {
	switch {
	case !u.Enabled:
		return "disabled"
	case u.ExpiresAt != nil && !now.Before(*u.ExpiresAt):
		return "expired"
	}
	return "active"
}

// State is the whole file.
type State struct {
	Version int     `json:"version"`
	Server  Server  `json:"server"`
	Users   []*User `json:"users"`
}

// User finds a user by name.
func (s *State) User(name string) *User {
	for _, u := range s.Users {
		if u.Name == name {
			return u
		}
	}
	return nil
}

// ActiveUsers returns users that may connect now.
func (s *State) ActiveUsers(now time.Time) []*User {
	var out []*User
	for _, u := range s.Users {
		if u.Active(now) {
			out = append(out, u)
		}
	}
	return out
}

// Store is a state file in a data directory.
type Store struct{ Dir string }

// Open returns the store in dir; it does not need to exist yet.
func Open(dir string) *Store { return &Store{Dir: dir} }

func (s *Store) path() string { return filepath.Join(s.Dir, fileName) }

// Path returns an absolute path for a file name relative to the data directory.
func (s *Store) Path(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(s.Dir, name)
}

// ErrNotInitialized means `omnitun-server init` has not been run.
var ErrNotInitialized = errors.New("server not initialized (run: omnitun-server init)")

// Exists reports whether the state file exists.
func (s *Store) Exists() bool {
	_, err := os.Stat(s.path())
	return err == nil
}

// ModTime of the state file, to detect changes.
func (s *Store) ModTime() time.Time {
	fi, err := os.Stat(s.path())
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// Load reads the state.
func (s *Store) Load() (*State, error) {
	data, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotInitialized
	}
	if err != nil {
		return nil, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("read %s: %w", s.path(), err)
	}
	changed, err := st.migrate()
	if err != nil {
		return nil, err
	}
	if changed {
		if err := s.Save(&st); err != nil {
			return nil, err
		}
	}
	return &st, nil
}

// Save writes the state atomically, readable only by the owner (it holds keys).
func (s *Store) Save(st *State) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	// On Windows the rename fails while another process (a reader, antivirus)
	// briefly has the file open; retry for up to ~1 s.
	for attempt := 0; ; attempt++ {
		err = os.Rename(tmp, s.path())
		if err == nil || attempt == 20 {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Update loads, applies f and saves.
func (s *Store) Update(f func(*State) error) error {
	st, err := s.Load()
	if err != nil {
		return err
	}
	if err := f(st); err != nil {
		return err
	}
	return s.Save(st)
}

var validName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,32}$`)

// AddUser creates a user with fresh credentials.
func AddUser(st *State, name string) (*User, error) {
	if !validName.MatchString(name) {
		return nil, fmt.Errorf("invalid user name %q (letters, digits, . _ -; up to 32)", name)
	}
	if st.User(name) != nil {
		return nil, fmt.Errorf("user %q already exists", name)
	}
	u := &User{
		Name:      name,
		UUID:      NewUUID(),
		Password:  RandomHex(16),
		SSKey:     RandomBase64(16),
		SubToken:  RandomHex(16),
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}
	st.Users = append(st.Users, u)
	return u, nil
}

// DeleteUser removes a user.
func DeleteUser(st *State, name string) error {
	i := slices.IndexFunc(st.Users, func(u *User) bool { return u.Name == name })
	if i < 0 {
		return fmt.Errorf("no user %q", name)
	}
	st.Users = slices.Delete(st.Users, i, i+1)
	return nil
}

// ResetCredentials issues new credentials and a new subscription token
// (e.g. after a link leaked); old links stop working.
func ResetCredentials(u *User) {
	u.UUID, u.Password, u.SSKey, u.SubToken = NewUUID(), RandomHex(16), RandomBase64(16), RandomHex(16)
}

// RandomHex returns n random bytes as hex.
func RandomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// NewUUID returns a random (v4) UUID.
func NewUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
