//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
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

func TestFailoverSwitchesToWorkingProfile(t *testing.T) {
	dead, live, testURL := failoverSetup(t)
	var log stateLog
	e := New(Config{
		DataDir:  t.TempDir(),
		OnState:  log.add,
		NetCheck: func(context.Context, netcheck.DialFunc) error { return nil },
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
}

func TestFailoverWaitsWhenOffline(t *testing.T) {
	dead, live, testURL := failoverSetup(t)
	var log stateLog
	e := New(Config{
		DataDir:  t.TempDir(),
		OnState:  log.add,
		NetCheck: func(context.Context, netcheck.DialFunc) error { return errors.New("offline") },
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
