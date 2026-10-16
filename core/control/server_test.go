package control

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/subscription"
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

func TestSubscribe(t *testing.T) {
	sub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Subscription-Userinfo", "upload=1; download=2; total=3")
		w.Write([]byte("trojan://pw@a.example.com:443#A\ntrojan://pw@b.example.com:443#B"))
	}))
	defer sub.Close()

	s, addr := newTestServer(t, ServerConfig{})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var res subscription.Result
	err = c.Call(t.Context(), "subscribe", subscription.Request{
		URL: sub.URL, Auth: subscription.Auth{Type: subscription.AuthBearer, Token: "good"},
	}, &res)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profiles) != 2 || res.Info.Total != 3 {
		t.Fatalf("result: %+v", res)
	}

	var ce *Error
	err = c.Call(t.Context(), "subscribe", subscription.Request{
		URL: sub.URL, Auth: subscription.Auth{Type: subscription.AuthBearer, Token: "bad"},
	}, nil)
	if !errors.As(err, &ce) || ce.Code != CodeAuthFailed {
		t.Fatalf("err = %v, want AUTH_FAILED", err)
	}
}

// The profile editor: forms, building, editing and exporting go through the
// same protocol as everything else, so every app edits profiles the same way.
func TestProfileEditorMethods(t *testing.T) {
	s, addr := newTestServer(t, ServerConfig{})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var forms []profile.ProtocolForm
	if err := c.Call(t.Context(), "forms", nil, &forms); err != nil {
		t.Fatal(err)
	}
	if len(forms) < 10 || forms[0].Protocol != profile.VLESS || len(forms[0].Fields) == 0 {
		t.Fatalf("forms: %d, first %+v", len(forms), forms[0])
	}

	var p profile.Profile
	err = c.Call(t.Context(), "build_profile", BuildProfileParams{Protocol: profile.Trojan,
		Values: map[string]string{"server": "vpn.example.com", "password": "pw", "name": "mine"}}, &p)
	if err != nil || p.Name != "mine" || p.Port != 443 || p.ID == "" {
		t.Fatalf("build: %v %+v", err, p)
	}
	var link string
	if err := c.Call(t.Context(), "export_link", ProfileParams{Profile: &p}, &link); err != nil || !strings.HasPrefix(link, "trojan://pw@vpn.example.com:443") {
		t.Fatalf("export: %v %q", err, link)
	}
	var values map[string]string
	if err := c.Call(t.Context(), "profile_values", ProfileParams{Profile: &p}, &values); err != nil || values["server"] != "vpn.example.com" || values["password"] != "pw" {
		t.Fatalf("values: %v %v", err, values)
	}
	var edited profile.Profile
	if err := c.Call(t.Context(), "build_profile", BuildProfileParams{Profile: &p, Values: map[string]string{"port": "8443"}}, &edited); err != nil || edited.Port != 8443 || edited.Password != "pw" {
		t.Fatalf("edit: %v %+v", err, edited)
	}
	if p.Port != 443 {
		t.Fatal("editing changed the original")
	}

	var ce *Error
	if err := c.Call(t.Context(), "build_profile", BuildProfileParams{Protocol: profile.Trojan, Values: map[string]string{"server": "x"}}, nil); !errors.As(err, &ce) || ce.Code != CodeBadRequest || !strings.Contains(ce.Message, "Password is required") {
		t.Fatalf("missing field: %v", err)
	}
	if err := c.Call(t.Context(), "export_link", ProfileParams{}, nil); !errors.As(err, &ce) || ce.Code != CodeBadRequest {
		t.Fatalf("export without a profile: %v", err)
	}
}
