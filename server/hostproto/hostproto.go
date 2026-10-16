// Package hostproto runs the protocols that sing-box does not serve: SSTP
// and IKEv2/IPsec (in-process veepin servers) and AmneziaWG (amneziawg-go). They need a TUN device and
// NAT on the host, so they work on Linux with CAP_NET_ADMIN (or root); the
// installer grants the capability and enables IP forwarding.
package hostproto

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/xen0bit/veepin/ikev2"
	"github.com/xen0bit/veepin/sstp"

	"github.com/NoobforAl/omnitun/core/protocols"
)

// Config is what a host server needs.
type Config struct {
	// Env carries the server's address and keys.
	Env  protocols.Env
	Port int
	// Users are the active accounts (name and password).
	Users []protocols.Account
	// DataDir holds files the server needs (the IKEv2 user list).
	DataDir string
	// CertPEM and KeyPEM are the TLS certificate and key (SSTP).
	CertPEM, KeyPEM []byte
	// CertFile and KeyFile are the same as files (IKEv2 authenticates the
	// server with its certificate; there is no shared secret).
	CertFile, KeyFile string
	// IKEPort is the UDP port IKE itself listens on (default 500, which
	// clients and the standards fix; tests use another).
	IKEPort int
	// Runner runs host commands (the AmneziaWG interface address); nil means
	// real commands.
	Runner Runner
	// Logf receives the servers' own error messages.
	Logf func(format string, args ...any)
}

// Server is a running host-level server.
type Server interface {
	// Close stops it and releases its TUN device.
	Close() error
	// Pool is the tunnel network its clients get (NAT has to cover it).
	Pool() string
}

// Usage is what one user moved through a host server since it started:
// Up from the client, Down to the client, in bytes of tunnelled traffic.
type Usage struct {
	User     string
	Up, Down uint64
}

// Meter is implemented by the host servers that count traffic per user.
// Usage is cumulative since the server started (it never goes down while the
// server runs), so a reader works out what is new by keeping the last values.
type Meter interface {
	Usage() []Usage
}

// Keys lists the protocols this package serves.
func Keys() []string { return []string{"sstp", "ikev2", "amneziawg"} }

// Start starts the host server for a protocol key. The server runs in the
// background; Start returns once it is listening or has failed.
func Start(key string, cfg Config) (Server, error) {
	if len(cfg.Users) == 0 {
		return nil, fmt.Errorf("%s: no active users", key)
	}
	switch key {
	case "sstp":
		return startSSTP(cfg)
	case "ikev2":
		return startIKEv2(cfg)
	case "amneziawg":
		return startAWG(cfg)
	}
	return nil, fmt.Errorf("%q is not a host-level protocol", key)
}

// publicDNS is pushed to clients.
var publicDNS = []net.IP{net.ParseIP("1.1.1.1"), net.ParseIP("8.8.8.8")}

type sstpServer struct{ s *sstp.Server }

func (x sstpServer) Close() error { return x.s.Close() }
func (sstpServer) Pool() string   { return protocols.SSTPPool }

func (x sstpServer) Usage() []Usage {
	var out []Usage
	for _, t := range x.s.Traffic() {
		out = append(out, Usage{User: t.User, Up: t.Up, Down: t.Down})
	}
	return out
}

func startSSTP(cfg Config) (Server, error) {
	if len(cfg.CertPEM) == 0 || len(cfg.KeyPEM) == 0 {
		return nil, fmt.Errorf("sstp: the server has no TLS certificate")
	}
	users := map[string]string{}
	for _, u := range cfg.Users {
		users[u.Name] = u.Password
	}
	srv, err := sstp.NewServer(sstp.ServerConfig{
		Cert: cfg.CertPEM, Key: cfg.KeyPEM, ListenPort: cfg.Port, Pool: protocols.SSTPPool,
		Users: users, DNS: publicDNS,
	})
	if err != nil {
		return nil, err
	}
	go srv.ListenAndServe()
	return sstpServer{srv}, nil
}

type ikev2Server struct {
	s     *ikev2.Server
	users string
	m     *ikeMeter
}

// Usage sums the live peers by the user they signed in as. A peer's counters
// belong to its tunnel, which ends with the session, so what a user moved is
// the total of the changes seen since the server started: traffic that came
// after the last reading and before a peer left is not seen.
func (x ikev2Server) Usage() []Usage {
	cur := map[string][2]uint64{} // by user and address: what each live tunnel carried
	for _, p := range x.s.Peers() {
		if p.Name == "" {
			continue
		}
		k := p.Name + "\x00" + p.Address
		v := cur[k]
		cur[k] = [2]uint64{v[0] + p.RxBytes, v[1] + p.TxBytes}
	}
	return x.m.add(cur)
}

// ikeMeter turns the live tunnels' counters into per-user totals.
type ikeMeter struct {
	mu    sync.Mutex
	last  map[string][2]uint64
	total map[string][2]uint64
}

func (m *ikeMeter) add(cur map[string][2]uint64) []Usage {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range cur {
		prev := m.last[k]
		d := [2]uint64{v[0] - prev[0], v[1] - prev[1]}
		for i := range d {
			if v[i] < prev[i] { // a new tunnel at the same address
				d[i] = v[i]
			}
		}
		user := k[:strings.IndexByte(k, 0)]
		t := m.total[user]
		m.total[user] = [2]uint64{t[0] + d[0], t[1] + d[1]}
	}
	m.last = cur
	out := make([]Usage, 0, len(m.total))
	for user, t := range m.total {
		out = append(out, Usage{User: user, Up: t[0], Down: t[1]})
	}
	return out
}

func (x ikev2Server) Close() error {
	err := x.s.Close()
	os.Remove(x.users)
	return err
}
func (ikev2Server) Pool() string { return protocols.IKEv2Pool }

// defaultIKEPort is where IKE itself listens; clients and the standards fix it.
const defaultIKEPort = 500

func startIKEv2(cfg Config) (Server, error) {
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("ikev2: the server has no certificate")
	}
	// The user list (name:password per line) is a file veepin reads; it holds
	// secrets, so it is private and removed when the server stops.
	names := make([]string, 0, len(cfg.Users))
	pw := map[string]string{}
	for _, u := range cfg.Users {
		if strings.ContainsAny(u.Name+u.Password, ":\r\n") {
			return nil, fmt.Errorf("ikev2: user %q cannot be written to the user list (':' or a line break in the name or password)", u.Name)
		}
		names = append(names, u.Name)
		pw[u.Name] = u.Password
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "%s:%s\n", n, pw[n])
	}
	file := filepath.Join(cfg.DataDir, "ikev2-users")
	if err := os.WriteFile(file, []byte(b.String()), 0o600); err != nil {
		return nil, err
	}
	ikePort := cfg.IKEPort
	if ikePort == 0 {
		ikePort = defaultIKEPort
	}
	srv, err := ikev2.NewServer(ikev2.ServerConfig{
		Port500: ikePort, Port4500: cfg.Port, CertFile: cfg.CertFile, KeyFile: cfg.KeyFile, LocalID: cfg.Env.Host,
		PublicIP: net.ParseIP(cfg.Env.Host), Pool: protocols.IKEv2Pool, DNS: publicDNS,
		EAPUsers: file, TCP: true,
	})
	if err != nil {
		os.Remove(file)
		return nil, err
	}
	go srv.ListenAndServe()
	return ikev2Server{s: srv, users: file, m: &ikeMeter{last: map[string][2]uint64{}, total: map[string][2]uint64{}}}, nil
}
