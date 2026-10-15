//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/netcheck"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// failoverSetup returns a dead profile, a working profile and a URL served behind the working one.
func failoverSetup(t *testing.T) (dead, live *profile.Profile, testURL string) {
	t.Helper()
	web := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go web.Serve(ln)
	t.Cleanup(func() { web.Close() })

	port := freePort(t)
	startServer(t, map[string]any{
		"type": "shadowsocks", "listen": "127.0.0.1", "listen_port": port,
		"method": "aes-128-gcm", "password": "pw", "network": "tcp",
	})
	res, err := importer.Import(fmt.Sprintf(
		"ss://aes-128-gcm:pw@127.0.0.1:%d#dead\nss://aes-128-gcm:pw@127.0.0.1:%d#live", freePort(t), port))
	if err != nil {
		t.Fatal(err)
	}
	return res.Profiles[0], res.Profiles[1], "http://" + ln.Addr().String() + "/"
}

type stateLog struct {
	mu     sync.Mutex
	states []State
}

func (l *stateLog) add(s State) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.states = append(l.states, s)
}

func (l *stateLog) waitFor(t *testing.T, what string, ok func(State) bool) State {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		l.mu.Lock()
		for _, s := range l.states {
			if ok(s) {
				l.mu.Unlock()
				return s
			}
		}
		l.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
	return State{}
}

// The dead selected profile is replaced by the live one, either during the
// connect handshake or, when connected without one, by the health-check loop.
func TestFailoverSwitchesToWorkingProfile(t *testing.T) {
	for _, skipHandshake := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipHandshake=%v", skipHandshake), func(t *testing.T) {
			dead, live, testURL := failoverSetup(t)
			var log stateLog
			e := New(Config{
				DataDir:       t.TempDir(),
				OnState:       log.add,
				NetCheck:      func(context.Context, netcheck.DialFunc) error { return nil },
				SkipHandshake: skipHandshake,
			})
			err := e.Start([]*profile.Profile{dead, live}, singbox.Options{
				ListenPort: freePort(t), LogLevel: "error", ProxyPrivate: true,
				Selected: dead.ID, TestURL: testURL,
				Failover: true, FailoverInterval: 200 * time.Millisecond,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { e.Stop() })

			s := log.waitFor(t, "failover", func(s State) bool { return s.Failover != nil })
			if s.Selected != live.ID || s.Failover.From != dead.ID || s.Failover.To != live.ID {
				t.Fatalf("unexpected failover state: selected=%s event=%+v", s.Selected, s.Failover)
			}
			log.waitFor(t, "connected", func(s State) bool { return s.Status == StatusRunning && s.Selected == live.ID })
		})
	}
}

// Connecting reports every stage in order and only says "connected" after the handshake.
func TestHandshakeProgress(t *testing.T) {
	_, live, testURL := failoverSetup(t)
	var log stateLog
	e := New(Config{DataDir: t.TempDir(), OnState: log.add})
	if err := e.Start(live.AsSlice(), singbox.Options{
		ListenPort: freePort(t), LogLevel: "error", ProxyPrivate: true, Selected: live.ID, TestURL: testURL,
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Stop() })
	log.waitFor(t, "connected", func(s State) bool { return s.Status == StatusRunning })

	var stages []string
	log.mu.Lock()
	for _, s := range log.states {
		if s.Progress != nil && (len(stages) == 0 || stages[len(stages)-1] != s.Progress.Stage) {
			stages = append(stages, s.Progress.Stage)
		}
		if s.Status == StatusRunning && s.Progress.Stage != StageConnected {
			t.Errorf("running before handshake finished: stage %s", s.Progress.Stage)
		}
	}
	log.mu.Unlock()
	want := []string{StagePrepare, StageEngineStart, StageTunnelUp, StageHandshake, StageConnected}
	if !slices.Equal(stages, want) {
		t.Fatalf("stages = %v, want %v", stages, want)
	}
}

// When no profile answers, the connection fails at the handshake stage and the tunnel is torn down.
func TestHandshakeFailureStopsTunnel(t *testing.T) {
	dead, _, testURL := failoverSetup(t)
	var log stateLog
	e := New(Config{
		DataDir:  t.TempDir(),
		OnState:  log.add,
		NetCheck: func(context.Context, netcheck.DialFunc) error { return nil },
	})
	if err := e.Start(dead.AsSlice(), singbox.Options{
		ListenPort: freePort(t), LogLevel: "error", ProxyPrivate: true, Selected: dead.ID, TestURL: testURL, Failover: true,
	}); err != nil {
		t.Fatal(err)
	}
	s := log.waitFor(t, "error", func(s State) bool { return s.Status == StatusError })
	if s.Progress == nil || s.Progress.Stage != StageHandshake || s.NoNetwork {
		t.Fatalf("unexpected failure state: %+v", s)
	}
	if !strings.HasPrefix(s.Error, "Contacting server failed") {
		t.Fatalf("error = %q", s.Error)
	}
	if e.State().Status != StatusError {
		t.Fatalf("final status = %s", e.State().Status)
	}
}

func TestFailoverWaitsWhenOffline(t *testing.T) {
	dead, live, testURL := failoverSetup(t)
	var log stateLog
	e := New(Config{
		DataDir:       t.TempDir(),
		OnState:       log.add,
		NetCheck:      func(context.Context, netcheck.DialFunc) error { return errors.New("offline") },
		SkipHandshake: true, // test the running-tunnel health check, not the connect handshake
	})
	err := e.Start([]*profile.Profile{dead, live}, singbox.Options{
		ListenPort: freePort(t), LogLevel: "error", ProxyPrivate: true,
		Selected: dead.ID, TestURL: testURL,
		Failover: true, FailoverInterval: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Stop() })

	s := log.waitFor(t, "no-network state", func(s State) bool { return s.NoNetwork })
	if s.Selected != dead.ID || s.Failover != nil {
		t.Fatalf("switched while offline: selected=%s event=%+v", s.Selected, s.Failover)
	}
}

func TestRotateAfter(t *testing.T) {
	order := []string{"a", "b", "c", "d"}
	if got := rotateAfter(order, "c"); !slices.Equal(got, []string{"d", "a", "b"}) {
		t.Fatalf("got %v", got)
	}
	if got := rotateAfter(order, "x"); !slices.Equal(got, order) {
		t.Fatalf("unknown current: got %v", got)
	}
}
