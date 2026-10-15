package control

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/netcheck"
	"github.com/NoobforAl/omnitun/core/singbox"
)

const (
	maxLine = 8 << 20
	// readTimeout closes clients that stop sending (the UI pings every few seconds).
	readTimeout = 15 * time.Second
	sendBuffer  = 256
)

// ServerConfig configures the daemon.
type ServerConfig struct {
	// DataDir is passed to the engine.
	DataDir string
	// Grace is how long the tunnel survives with no UI connected. Zero means 5s.
	Grace time.Duration
	// Token authenticates clients. Empty generates a random one.
	Token string
	// Logf receives daemon diagnostics.
	Logf func(format string, args ...any)
	// NetCheck reports whether the underlying network is up. Nil uses netcheck.Online.
	NetCheck func(ctx context.Context) error
	// CertificateStore is passed to the engine (see engine.Config).
	CertificateStore string
	// Platform is passed to the engine (mobile only).
	Platform adapter.PlatformInterface
	// OnState is called on every engine state change, after clients are notified.
	OnState func(engine.State)
}

// Server owns the engine and serves UI clients.
type Server struct {
	cfg    ServerConfig
	engine *engine.Engine

	mu         sync.Mutex
	clients    map[*client]struct{}
	graceTimer *time.Timer
	ln         net.Listener
	closed     bool
}

type client struct {
	conn net.Conn
	send chan any
	done chan struct{}
	once sync.Once
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}

// NewServer creates a server and its engine.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.Grace == 0 {
		cfg.Grace = 5 * time.Second
	}
	if cfg.Token == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		cfg.Token = hex.EncodeToString(b)
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	if cfg.NetCheck == nil {
		cfg.NetCheck = netcheck.Online
	}
	s := &Server{cfg: cfg, clients: map[*client]struct{}{}}
	s.engine = engine.New(engine.Config{
		DataDir:          cfg.DataDir,
		Platform:         cfg.Platform,
		CertificateStore: cfg.CertificateStore,
		OnState: func(st engine.State) {
			s.broadcast(Event{Event: "state", Data: st})
			if cfg.OnState != nil {
				cfg.OnState(st)
			}
		},
		OnLog: func(level, msg string) {
			s.broadcast(Event{Event: "log", Data: LogEntry{Time: time.Now(), Level: level, Message: msg}})
		},
	})
	return s, nil
}

// Token returns the client authentication token.
func (s *Server) Token() string { return s.cfg.Token }

// Engine exposes the engine (for tests and the CLI).
func (s *Server) Engine() *engine.Engine { return s.engine }

// Listen binds addr (use 127.0.0.1:0 for a random port).
func (s *Server) Listen(addr string) (net.Addr, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()
	return ln.Addr(), nil
}

// WriteControlFile stores the address and token for clients, readable only by the current user.
func (s *Server) WriteControlFile(path string) error {
	s.mu.Lock()
	ln := s.ln
	s.mu.Unlock()
	if ln == nil {
		return errors.New("not listening")
	}
	data, err := json.Marshal(ControlFile{Addr: ln.Addr().String(), Token: s.cfg.Token, PID: os.Getpid()})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Serve accepts clients until Close.
func (s *Server) Serve() error {
	s.mu.Lock()
	ln := s.ln
	s.mu.Unlock()
	if ln == nil {
		return errors.New("not listening")
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return nil
			}
			return err
		}
		go s.handle(conn)
	}
}

// Close stops the tunnel, disconnects clients and stops listening.
func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	if s.ln != nil {
		s.ln.Close()
	}
	if s.graceTimer != nil {
		s.graceTimer.Stop()
	}
	clients := s.clients
	s.clients = map[*client]struct{}{}
	s.mu.Unlock()
	for c := range clients {
		c.close()
	}
	return s.engine.Stop()
}

// broadcast queues an event for every client without blocking; slow clients drop events.
func (s *Server) broadcast(ev Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.clients {
		select {
		case c.send <- ev:
		default:
		}
	}
}

func (s *Server) addClient(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = struct{}{}
	if s.graceTimer != nil {
		s.graceTimer.Stop()
		s.graceTimer = nil
	}
}

// removeClient starts the grace timer when the last client leaves: a tunnel
// must never outlive the UI that controls it.
func (s *Server) removeClient(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[c]; !ok {
		return
	}
	delete(s.clients, c)
	if len(s.clients) > 0 || s.closed {
		return
	}
	s.graceTimer = time.AfterFunc(s.cfg.Grace, func() {
		s.mu.Lock()
		idle := len(s.clients) == 0
		s.mu.Unlock()
		if idle && s.engine.State().Status != engine.StatusStopped {
			s.cfg.Logf("no UI connected for %s, stopping tunnel", s.cfg.Grace)
			s.engine.Stop()
		}
	})
}

func (s *Server) handle(conn net.Conn) {
	c := &client{conn: conn, send: make(chan any, sendBuffer), done: make(chan struct{})}
	defer c.close()

	go func() {
		enc := json.NewEncoder(conn)
		for {
			select {
			case <-c.done:
				return
			case msg := <-c.send:
				if m, ok := msg.(flushMarker); ok {
					close(m)
					continue
				}
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := enc.Encode(msg); err != nil {
					c.close()
					return
				}
			}
		}
	}()

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 64<<10), maxLine)
	authed := false
	defer func() {
		if authed {
			s.removeClient(c)
		}
	}()
	for {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		if !sc.Scan() {
			return
		}
		var req Request
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			s.replyAndClose(c, Response{Error: &Error{Code: CodeBadRequest, Message: err.Error()}})
			return
		}
		if !authed {
			var p HelloParams
			if req.Method != "hello" || json.Unmarshal(req.Params, &p) != nil ||
				subtle.ConstantTimeCompare([]byte(p.Token), []byte(s.cfg.Token)) != 1 {
				s.replyAndClose(c, Response{ID: req.ID, Error: &Error{Code: CodeUnauthorized, Message: "invalid token"}})
				return
			}
			authed = true
			s.addClient(c)
			s.reply(c, Response{ID: req.ID, Result: map[string]any{"version": Version, "state": s.engine.State()}})
			continue
		}
		// Requests run concurrently so a slow latency test doesn't block pings.
		go func(req Request) {
			result, err := s.dispatch(req)
			resp := Response{ID: req.ID, Result: result}
			if err != nil {
				var ce *Error
				if !errors.As(err, &ce) {
					ce = &Error{Code: CodeBadRequest, Message: err.Error()}
				}
				resp.Error, resp.Result = ce, nil
			}
			s.reply(c, resp)
		}(req)
	}
}

// replyAndClose flushes a final response through the writer before the connection closes.
func (s *Server) replyAndClose(c *client, r Response) {
	flushed := make(chan struct{})
	s.reply(c, r)
	s.reply(c, flushMarker(flushed))
	select {
	case <-flushed:
	case <-time.After(2 * time.Second):
	}
}

// flushMarker is queued after a message; the writer closes it instead of sending it.
type flushMarker chan struct{}

func (s *Server) reply(c *client, r any) {
	select {
	case c.send <- r:
	case <-c.done:
	}
}

func decode[T any](raw json.RawMessage) (T, error) {
	var v T
	if len(raw) == 0 {
		return v, nil
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, &Error{Code: CodeBadRequest, Message: err.Error()}
	}
	return v, nil
}

func (s *Server) dispatch(req Request) (any, error) {
	switch req.Method {
	case "ping":
		return "pong", nil
	case "state":
		return s.engine.State(), nil
	case "import":
		p, err := decode[ImportParams](req.Params)
		if err != nil {
			return nil, err
		}
		res, err := importer.Import(p.Text)
		if err != nil {
			return nil, &Error{Code: CodeImportFailed, Message: err.Error()}
		}
		return res, nil
	case "start":
		p, err := decode[StartParams](req.Params)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.cfg.NetCheck(ctx); err != nil {
			return nil, &Error{Code: CodeNoNetwork, Message: err.Error()}
		}
		opts := singbox.Options{
			Mode:         singbox.Mode(p.Mode),
			ListenPort:   p.ListenPort,
			Selected:     p.Selected,
			ProxyPrivate: p.ProxyPrivate,
			SystemProxy:  p.SystemProxy,
			DNS:          p.DNS,
			TestURL:      p.TestURL,
		}
		if err := s.engine.Start(p.Profiles, opts); err != nil {
			return nil, &Error{Code: CodeStartFailed, Message: err.Error()}
		}
		return s.engine.State(), nil
	case "stop":
		return s.engine.State(), s.engine.Stop()
	case "select":
		p, err := decode[SelectParams](req.Params)
		if err != nil {
			return nil, err
		}
		if err := s.engine.Select(p.ID); err != nil {
			return nil, &Error{Code: CodeNotRunning, Message: err.Error()}
		}
		return s.engine.State(), nil
	case "test":
		p, err := decode[TestParams](req.Params)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return s.engine.TestLatency(ctx, p.Profiles, p.URL)
	default:
		return nil, &Error{Code: CodeUnknownMethod, Message: fmt.Sprintf("unknown method %q", req.Method)}
	}
}
