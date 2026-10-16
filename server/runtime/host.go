package runtime

import (
	"time"

	"github.com/NoobforAl/omnitun/server/hostproto"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/store"
	"github.com/NoobforAl/omnitun/server/tlsfiles"
)

// startHostLocked starts the host-level protocols (SSTP, IKEv2, AmneziaWG) that are
// enabled and have users, and sets up NAT for their client networks. A
// failure is logged and leaves the others running: these protocols need the
// host's cooperation (Linux, CAP_NET_ADMIN, nftables or iptables) that the
// sing-box ones do not.
func (s *Server) startHostLocked(st *store.State) {
	users := st.ActiveUsers(time.Now())
	if len(users) == 0 {
		return
	}
	var pools []string
	for _, key := range hostproto.Keys() {
		p := st.Server.Protocols[key]
		if p == nil || !p.Enabled {
			continue
		}
		cfg := hostproto.Config{
			Env: serverconf.Env(st, s.store), Port: p.Port, Users: serverconf.Accounts(users), DataDir: s.store.Dir,
			Runner: s.runner, Logf: s.logf,
		}
		if key == "sstp" || key == "ikev2" {
			domain := ""
			if st.Server.ACMEActive() {
				domain = st.Server.ACME.Domain
			}
			src := tlsfiles.New(serverconf.ACMEDir(s.store), domain, s.store.Path(st.Server.CertFile), s.store.Path(st.Server.KeyFile))
			var err error
			if cfg.CertFile, cfg.KeyFile, err = src.Paths(); err != nil {
				s.logf("%s: no certificate: %v", key, err)
				continue
			}
			if cfg.CertPEM, cfg.KeyPEM, err = src.PEM(); err != nil {
				s.logf("%s: no certificate: %v", key, err)
				continue
			}
		}
		srv, err := hostproto.Start(key, cfg)
		if err != nil {
			s.logf("%s did not start: %v (it needs Linux and CAP_NET_ADMIN; see `omnitun-server install`)", key, err)
			continue
		}
		if s.hosts == nil {
			s.hosts = map[string]hostproto.Server{}
		}
		s.hosts[key] = srv
		if m, ok := srv.(hostproto.Meter); ok {
			s.meters.add(key, m)
		}
		pools = append(pools, srv.Pool())
		s.logf("serving %s on port %d", key, p.Port)
	}
	if len(pools) == 0 {
		return
	}
	undo, err := hostproto.EnableNAT(s.runner, s.store.Dir, pools)
	if err != nil {
		s.logf("NAT for the clients of SSTP, IKEv2 and AmneziaWG failed: %v (they can connect but will not reach the internet)", err)
		return
	}
	s.undoNAT = undo
	if !hostproto.ForwardingEnabled() {
		s.logf("warning: IP forwarding is off; enable it with `sysctl -w net.ipv4.ip_forward=1` or run `omnitun-server install`")
	}
}

// stopHostLocked stops the host-level servers and removes their NAT rules.
func (s *Server) stopHostLocked() {
	for key, srv := range s.hosts {
		s.meters.remove(key, s.tracker) // the last reading, while the server still counts
		srv.Close()
		delete(s.hosts, key)
	}
	if s.undoNAT != nil {
		s.undoNAT()
		s.undoNAT = nil
	}
}
