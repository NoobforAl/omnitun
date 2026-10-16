// Package runtime runs the VPN server: a sing-box instance built from the
// state file plus the subscription endpoint. It reloads when the state file
// changes, so `omnitun-server user ...` and the TUI apply without a restart.
package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/common/trafficcontrol"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/service"

	"github.com/NoobforAl/omnitun/server/hostproto"
	"github.com/NoobforAl/omnitun/server/logs"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/stats"
	"github.com/NoobforAl/omnitun/server/store"
	"github.com/NoobforAl/omnitun/server/tlsfiles"
)

// Server is a running VPN server.
type Server struct {
	store *store.Store
	logf  func(format string, args ...any)

	mu       sync.Mutex
	instance *box.Box
	cancel   context.CancelFunc
	sub      *http.Server
	loaded   time.Time // state file mod time of the running config

	file    *logs.Log      // server.log
	tracker *stats.Tracker // per-user traffic, shared by every sing-box run
	accMu   sync.Mutex     // guards the collector's tick against the final flush
	coll    *collector     // the running instance's collector (guarded by accMu)
	accStop context.CancelFunc

	hosts   map[string]hostproto.Server // running SSTP / IKEv2 servers
	undoNAT func()
	runner  hostproto.Runner // runs nft/iptables; tests replace it

	wgMu    sync.RWMutex
	wgUsers map[netip.Addr]string // WireGuard tunnel address -> user name
}

// setWireGuardUsers remembers which user owns which tunnel address.
func (s *Server) setWireGuardUsers(st *store.State) {
	m := map[netip.Addr]string{}
	for _, u := range st.Users {
		if p, err := netip.ParsePrefix(u.WGAddress); err == nil {
			m[p.Addr()] = u.Name
		}
	}
	s.wgMu.Lock()
	s.wgUsers = m
	s.wgMu.Unlock()
}

func (s *Server) userByWireGuardIP(a netip.Addr) string {
	s.wgMu.RLock()
	defer s.wgMu.RUnlock()
	return s.wgUsers[a.Unmap()]
}

// New prepares a server for the store.
func New(s *store.Store, logf func(string, ...any)) *Server {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	// Everything the server says also goes to server.log, where the CLI, the
	// TUI and the API can read it.
	file := logs.Open(s.Dir)
	srv := &Server{store: s, file: file, runner: hostproto.ExecRunner{}}
	srv.logf = func(format string, args ...any) {
		logf(format, args...)
		file.Write("info", fmt.Sprintf(format, args...))
	}
	return srv
}

// Start loads the state and starts sing-box and the subscription endpoint.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	s.tracker = stats.NewTracker(s.store.Dir)
	s.setWireGuardUsers(st)
	accCtx, accCancel := context.WithCancel(context.Background())
	s.accStop = accCancel
	go s.accountingLoop(accCtx)
	if renewed, err := store.RenewSelfSignedCert(s.store, st); err != nil {
		return fmt.Errorf("renew certificate: %w", err)
	} else if renewed {
		s.logf("issued a new self-signed certificate (valid %d days)", int(store.SelfSignedValidity.Hours()/24))
	}
	if err := s.startBoxLocked(st); err != nil {
		return err
	}
	s.startHostLocked(st)
	if sub := st.Server.Subscription; sub.Enabled {
		ln, err := net.Listen("tcp", net.JoinHostPort(sub.Listen, strconv.Itoa(sub.Port)))
		if err != nil {
			s.stopBoxLocked()
			return fmt.Errorf("subscription endpoint: %w", err)
		}
		s.sub = &http.Server{Handler: serverconf.SubscriptionHandler(s.store), ReadHeaderTimeout: 10 * time.Second}
		if st.Server.ACMEActive() {
			src := tlsfiles.New(serverconf.ACMEDir(s.store), st.Server.ACME.Domain, s.store.Path(st.Server.CertFile), s.store.Path(st.Server.KeyFile))
			s.sub.TLSConfig = src.TLSConfig()
			go s.sub.ServeTLS(ln, "", "")
			s.logf("subscriptions on https://%s", ln.Addr())
		} else {
			go s.sub.Serve(ln)
			s.logf("subscriptions on %s", ln.Addr())
		}
	}
	return nil
}

func (s *Server) startBoxLocked(st *store.State) error {
	s.loaded = s.store.ModTime()
	cfg := serverconf.Build(st, s.store, time.Now())
	inbounds, _ := cfg["inbounds"].([]any)
	endpoints, _ := cfg["endpoints"].([]any)
	if len(inbounds)+len(endpoints) == 0 {
		s.logf("no active users or protocols; waiting for changes")
		return nil
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	opts, err := sjson.UnmarshalExtendedContext[option.Options](ctx, raw)
	if err == nil {
		var instance *box.Box
		// The platform writer makes sing-box create its traffic manager,
		// which the accounting reads.
		instance, err = box.New(box.Options{Context: ctx, Options: opts, PlatformLogWriter: sbLog{s.file}})
		if err == nil {
			if err = instance.Start(); err == nil {
				s.instance, s.cancel = instance, cancel
				if tm := service.PtrFromContext[trafficcontrol.Manager](ctx); tm != nil {
					s.accMu.Lock()
					s.coll = newCollector(tm, s.tracker, s.userByWireGuardIP)
					s.accMu.Unlock()
				}
				s.logf("serving %d protocol(s) for %d user(s)", len(inbounds)+len(endpoints), len(st.ActiveUsers(time.Now())))
				return nil
			}
			instance.Close()
		}
	}
	cancel()
	return fmt.Errorf("start sing-box: %w", err)
}

func (s *Server) stopBoxLocked() {
	// Count what the connections moved since the last tick before they close.
	s.accMu.Lock()
	if s.coll != nil {
		s.coll.tick()
		s.coll = nil
	}
	s.accMu.Unlock()
	if s.instance != nil {
		s.instance.Close()
		s.cancel()
		s.instance, s.cancel = nil, nil
	}
}

// Reload restarts sing-box with the current state (brief interruption for
// connected clients).
func (s *Server) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	s.stopBoxLocked()
	s.stopHostLocked()
	err = s.startBoxLocked(st)
	s.startHostLocked(st)
	return err
}

// Watch reloads when the state file changes and once a minute (to apply
// expirations), until ctx is done.
func (s *Server) Watch(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	lastMinute := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		s.mu.Lock()
		changed := !s.store.ModTime().Equal(s.loaded)
		s.mu.Unlock()
		if changed || time.Since(lastMinute) >= time.Minute {
			lastMinute = time.Now()
			if err := s.reloadIfNeeded(changed); err != nil {
				s.logf("reload failed: %v", err)
			}
		}
	}
}

// reloadIfNeeded reloads on file changes, or when the set of active users
// changed because someone expired.
func (s *Server) reloadIfNeeded(changed bool) error {
	if changed {
		s.logf("state changed, reloading")
		return s.Reload()
	}
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	s.mu.Lock()
	running := s.instance != nil
	s.mu.Unlock()
	if !running && len(st.ActiveUsers(time.Now())) > 0 {
		return s.Reload()
	}
	for _, u := range st.Users {
		if u.Enabled && u.ExpiresAt != nil && time.Since(*u.ExpiresAt) >= 0 && time.Since(*u.ExpiresAt) < time.Minute+5*time.Second {
			s.logf("user %s expired, reloading", u.Name)
			return s.Reload()
		}
	}
	return nil
}

// Close stops everything.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopBoxLocked()
	s.stopHostLocked()
	defer s.file.Close()
	if s.accStop != nil {
		s.accStop()
	}
	if s.tracker != nil {
		if err := s.tracker.Stop(); err != nil {
			s.logf("save stats: %v", err)
		}
	}
	var err error
	if s.sub != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err = s.sub.Shutdown(ctx)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
	}
	return err
}
