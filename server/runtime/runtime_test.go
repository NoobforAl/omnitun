//go:build with_quic && with_utls && with_wireguard && with_gvisor

package runtime

import (
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/store"
)

func freeTCP(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func freeUDP(t *testing.T) int {
	t.Helper()
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

// setup initializes a server on localhost with every protocol and one user.
func setup(t *testing.T) (*store.Store, *Server) {
	t.Helper()
	// Reality impersonates this local TLS server instead of a real website.
	decoy := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(decoy.Close)
	_, decoyPort, _ := net.SplitHostPort(decoy.Listener.Addr().String())

	s := store.Open(t.TempDir())
	_, err := store.Init(s, store.InitOptions{
		Name:       "Test",
		PublicHost: "127.0.0.1",
		Ports: map[string]int{
			store.VLESSReality: freeTCP(t),
			store.Shadowsocks:  freeTCP(t),
			store.Trojan:       freeTCP(t),
			store.Hysteria2:    freeUDP(t),
			store.TUIC:         freeUDP(t),
		},
		RealityHandshake: "localhost:" + decoyPort,
		SubPort:          freeTCP(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(st *store.State) error {
		// Shadowsocks over UDP would need a free UDP port with the same number; keep it TCP-only here.
		st.Server.Subscription.Listen = "127.0.0.1"
		_, err := store.AddUser(st, "alice")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	srv := New(s, t.Logf)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.Close() })
	return s, srv
}

// Every generated link must work with the Omnitun client engine against the generated server.
func TestGeneratedLinksConnect(t *testing.T) {
	s, _ := setup(t)
	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok:"+r.URL.Path)
	}))
	defer web.Close()

	links := serverconf.Links(st, st.User("alice"))
	if len(links) != len(store.AllProtocols) {
		t.Fatalf("got %d links", len(links))
	}
	for _, l := range links {
		t.Run(l.Protocol, func(t *testing.T) {
			p, err := importer.ParseLink(l.URL)
			if err != nil {
				t.Fatalf("client cannot parse %s: %v", l.URL, err)
			}
			proxyPort := freeTCP(t)
			e := engine.New(engine.Config{DataDir: t.TempDir()})
			if err := e.Start(p.AsSlice(), singbox.Options{
				ListenPort: proxyPort, LogLevel: "error", ProxyPrivate: true,
				Selected: p.ID, TestURL: web.URL + "/probe",
			}); err != nil {
				t.Fatal(err)
			}
			defer e.Stop()
			deadline := time.Now().Add(15 * time.Second)
			for e.State().Status != engine.StatusRunning {
				if st := e.State(); st.Status == engine.StatusError || time.Now().After(deadline) {
					t.Fatalf("not connected: %+v", st)
				}
				time.Sleep(50 * time.Millisecond)
			}
			proxy, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(proxyPort))
			client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}, Timeout: 10 * time.Second}
			resp, err := client.Get(web.URL + "/data")
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if string(body) != "ok:/data" {
				t.Fatalf("body = %q", body)
			}
		})
	}
}

func TestSubscriptionEndpoint(t *testing.T) {
	s, srv := setup(t)
	st, _ := s.Load()
	u := st.User("alice")
	subURL := serverconf.SubscriptionURL(st, u)

	resp, err := http.Get(subURL)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		t.Fatal(err)
	}
	res, err := importer.Import(string(decoded))
	if err != nil || len(res.Profiles) != len(store.AllProtocols) {
		t.Fatalf("import: %v, %d profiles", err, len(res.Profiles))
	}

	// Wrong token: not found. Disabled user: forbidden and removed from sing-box on reload.
	if r, _ := http.Get(subURL + "x"); r.StatusCode != http.StatusNotFound {
		t.Fatalf("bad token status %d", r.StatusCode)
	}
	if err := s.Update(func(st *store.State) error { st.User("alice").Enabled = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if r, _ := http.Get(subURL); r.StatusCode != http.StatusForbidden {
		t.Fatalf("disabled user status %d", r.StatusCode)
	}
	if err := srv.Reload(); err != nil {
		t.Fatal(err)
	}
	srv.mu.Lock()
	running := srv.instance != nil
	srv.mu.Unlock()
	if running {
		t.Fatal("sing-box still running with no active users")
	}
}
