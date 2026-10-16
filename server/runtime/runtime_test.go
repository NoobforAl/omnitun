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

// freeBoth returns a port free for both TCP and UDP (Shadowsocks uses both;
// on Windows a port free for one can lie in a reserved range for the other).
func freeBoth(t *testing.T) int {
	t.Helper()
	for range 50 {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		c, err := net.ListenPacket("udp", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
		l.Close()
		if err == nil {
			c.Close()
			return port
		}
	}
	t.Fatal("no port free for both TCP and UDP")
	return 0
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

	s := store.Open(tempDir(t))
	_, err := store.Init(s, store.InitOptions{
		Name:       "Test",
		PublicHost: "127.0.0.1",
		Ports: map[string]int{
			store.VLESSReality: freeTCP(t),
			store.Shadowsocks:  freeBoth(t),
			store.Trojan:       freeTCP(t),
			store.Hysteria2:    freeUDP(t),
			store.TUIC:         freeUDP(t),
			store.AnyTLS:       freeTCP(t),
			store.ShadowTLS:    freeTCP(t),
			store.Naive:        freeTCP(t),
			store.OpenVPNUDP:   freeUDP(t),
			store.OpenVPNTCP:   freeTCP(t),
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

// lanIPv4 returns a non-loopback IPv4 address: packets inside an OpenVPN
// tunnel can't go to loopback (the server's netstack drops them).
func lanIPv4(t *testing.T) string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && !ipn.IP.IsLoopback() && !ipn.IP.IsLinkLocalUnicast() {
			return ipn.IP.String()
		}
	}
	t.Skip("no LAN IPv4 address")
	return ""
}

// Every profile the server hands out (the sing-box subscription, which also
// carries OpenVPN) must work with the Omnitun client engine against the server.
func TestGeneratedProfilesConnect(t *testing.T) {
	s, _ := setup(t)
	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok:"+r.URL.Path)
	}))
	web.Listener.Close()
	if web.Listener, err = net.Listen("tcp", net.JoinHostPort(lanIPv4(t), "0")); err != nil {
		t.Fatal(err)
	}
	web.Start()
	defer web.Close()

	body, err := serverconf.SingBoxSubscription(st, s, st.User("alice"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := importer.Import(string(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profiles) != len(store.DefaultEnabled()) {
		t.Fatalf("got %d profiles, errors %v", len(res.Profiles), res.Errors)
	}
	for _, p := range res.Profiles {
		t.Run(p.Name, func(t *testing.T) {
			if p.Protocol == "naive" && !naiveIncluded {
				t.Skip("naive client needs -tags with_naive_outbound (see naive_test.go)")
			}
			proxyPort := freeTCP(t)
			e := engine.New(engine.Config{DataDir: tempDir(t)})
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
	// Share links exist for everything except OpenVPN.
	res, err := importer.Import(string(decoded))
	if err != nil || len(res.Profiles) != len(store.DefaultEnabled())-2 {
		t.Fatalf("import: %v, %d profiles", err, len(res.Profiles))
	}

	// The Omnitun client gets a sing-box config including both OpenVPN variants.
	req, _ := http.NewRequest(http.MethodGet, subURL, nil)
	req.Header.Set("User-Agent", "Omnitun/0.1 v2rayN/7.0")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	res, err = importer.Import(string(body))
	if err != nil || len(res.Profiles) != len(store.DefaultEnabled()) {
		t.Fatalf("sing-box import: %v, %d profiles %v", err, len(res.Profiles), res.Errors)
	}
	ovpns := 0
	for _, p := range res.Profiles {
		if p.Protocol == "openvpn" && p.Username == "alice" && p.Password == u.Password {
			ovpns++
		}
	}
	if ovpns != 2 {
		t.Fatalf("openvpn profiles with credentials: %d", ovpns)
	}

	// The .ovpn download works in any OpenVPN client (and in Omnitun).
	resp, err = http.Get(serverconf.OVPNURL(st, u, store.OpenVPNTCP))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if res, err := importer.Import(string(body)); err != nil || res.Profiles[0].OpenVPN.Network != "tcp" {
		t.Fatalf(".ovpn import: %v", err)
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

// A server with only OpenVPN enabled still starts.
func TestOnlyOpenVPN(t *testing.T) {
	s, srv := setup(t)
	if err := s.Update(func(st *store.State) error {
		for _, p := range store.AllProtocols {
			st.Server.Protocols[p].Enabled = store.IsOpenVPN(p)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := srv.Reload(); err != nil {
		t.Fatal(err)
	}
	srv.mu.Lock()
	running := srv.instance != nil
	srv.mu.Unlock()
	if !running {
		t.Fatal("sing-box not running with only OpenVPN enabled")
	}
}
