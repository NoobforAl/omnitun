package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
)

func newTestServer(t *testing.T, cfg ServerConfig) (*Server, string) {
	t.Helper()
	cfg.DataDir = t.TempDir()
	cfg.SkipHandshake = true // example.com servers are unreachable
	if cfg.NetCheck == nil {
		cfg.NetCheck = func(context.Context) error { return nil }
	}
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := s.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go s.Serve()
	t.Cleanup(func() { s.Close() })
	return s, addr.String()
}

func TestRejectsBadToken(t *testing.T) {
	_, addr := newTestServer(t, ServerConfig{})
	_, err := Dial(t.Context(), addr, "wrong")
	var ce *Error
	if !errors.As(err, &ce) || ce.Code != CodeUnauthorized {
		t.Fatalf("err = %v, want UNAUTHORIZED", err)
	}
}

func TestImportAndUnknownMethod(t *testing.T) {
	s, addr := newTestServer(t, ServerConfig{})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var res importer.Result
	if err := c.Call(t.Context(), "import", ImportParams{Text: "trojan://pw@example.com:443#t"}, &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Profiles) != 1 || res.Profiles[0].Name != "t" {
		t.Fatalf("import result: %+v", res)
	}
	var ce *Error
	if err := c.Call(t.Context(), "bogus", nil, nil); !errors.As(err, &ce) || ce.Code != CodeUnknownMethod {
		t.Fatalf("err = %v", err)
	}
}

func TestStartReportsNoNetwork(t *testing.T) {
	s, addr := newTestServer(t, ServerConfig{
		NetCheck: func(context.Context) error { return errors.New("offline") },
	})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var res importer.Result
	c.Call(t.Context(), "import", ImportParams{Text: "trojan://pw@example.com:443#t"}, &res)
	var ce *Error
	err = c.Call(t.Context(), "start", StartParams{Profiles: res.Profiles}, nil)
	if !errors.As(err, &ce) || ce.Code != CodeNoNetwork {
		t.Fatalf("err = %v, want NO_NETWORK", err)
	}
}

// When the last UI disconnects, the tunnel stops after the grace period.
func TestTunnelStopsWhenUIDisconnects(t *testing.T) {
	s, addr := newTestServer(t, ServerConfig{Grace: 200 * time.Millisecond})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	var res importer.Result
	if err := c.Call(t.Context(), "import", ImportParams{Text: "trojan://pw@example.com:443#t"}, &res); err != nil {
		t.Fatal(err)
	}
	var st engine.State
	if err := c.Call(t.Context(), "start", StartParams{Profiles: res.Profiles, ListenPort: freePort(t)}, &st); err != nil {
		t.Fatal(err)
	}
	if st.Status != engine.StatusRunning {
		t.Fatalf("status = %s", st.Status)
	}
	// A state event is pushed for the transition.
	select {
	case ev := <-c.Events():
		if ev.Event != "state" && ev.Event != "log" {
			t.Fatalf("unexpected event %q", ev.Event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event received")
	}

	c.Close()
	deadline := time.Now().Add(3 * time.Second)
	for s.Engine().State().Status != engine.StatusStopped {
		if time.Now().After(deadline) {
			t.Fatal("tunnel still running after UI disconnected")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// With the lease disabled (mobile), the tunnel keeps running without a UI.
func TestNoLeaseKeepsTunnel(t *testing.T) {
	s, addr := newTestServer(t, ServerConfig{Grace: -1})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	var res importer.Result
	if err := c.Call(t.Context(), "import", ImportParams{Text: "trojan://pw@example.com:443#t"}, &res); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(t.Context(), "start", StartParams{Profiles: res.Profiles, ListenPort: freePort(t)}, nil); err != nil {
		t.Fatal(err)
	}
	c.Close()
	time.Sleep(500 * time.Millisecond)
	if st := s.Engine().State().Status; st != engine.StatusRunning {
		t.Fatalf("status = %s, want running", st)
	}
}
