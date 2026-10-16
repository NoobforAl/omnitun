//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
	"github.com/NoobforAl/omnitun/core/subscription"
)

// A provider that is blocked here can be reached through the VPN: the request
// goes out of the tunnel's server, which is where the provider is reachable.
// The "provider" below only exists as a name the tunnel's server maps to a
// local web server, so a direct request cannot work.
func TestSubscriptionIsFetchedThroughTheTunnel(t *testing.T) {
	body := base64.StdEncoding.EncodeToString([]byte("trojan://pw@a.example.com:443#From the provider"))
	web := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Subscription-Userinfo", "upload=1; download=2; total=100")
		fmt.Fprint(w, body)
	})}
	webLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go web.Serve(webLn)
	t.Cleanup(func() { web.Close() })

	// The tunnel's server sends every connection to that web server.
	ssPort := freePort(t)
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	t.Cleanup(cancel)
	opts, err := parseOptions(ctx, map[string]any{
		"log": map[string]any{"level": "error"},
		"inbounds": []any{map[string]any{
			"type": "shadowsocks", "listen": "127.0.0.1", "listen_port": ssPort,
			"method": "aes-128-gcm", "password": "test-pass", "network": "tcp",
		}},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
		"route": map[string]any{"rules": []any{map[string]any{
			"action": "route-options", "override_address": "127.0.0.1", "override_port": webLn.Addr().(*net.TCPAddr).Port,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := box.New(box.Options{Context: ctx, Options: opts})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })

	res, err := importer.Import(fmt.Sprintf("ss://aes-128-gcm:test-pass@127.0.0.1:%d#local", ssPort))
	if err != nil {
		t.Fatal(err)
	}
	e := New(Config{DataDir: tempDir(t), CertificateStore: "mozilla"})
	if _, err := e.TunnelDialer(); !errors.Is(err, ErrNoTunnel) {
		t.Fatalf("a dialer without a tunnel: %v", err)
	}
	err = e.Start(res.Profiles, singbox.Options{
		ListenPort: freePort(t), LogLevel: "error", Selected: res.Profiles[0].ID, ProxyPrivate: true,
		TestURL: "http://provider.invalid/probe",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Stop() })

	req := subscription.Request{URL: "http://provider.invalid/sub", Auth: subscription.Auth{Type: subscription.AuthNone}}
	fetchCtx, stop := context.WithTimeout(t.Context(), 15*time.Second)
	defer stop()

	// Directly the name does not exist (a short wait: resolving a made-up name can be slow).
	directCtx, stopDirect := context.WithTimeout(t.Context(), 2*time.Second)
	defer stopDirect()
	if _, err := subscription.Fetch(directCtx, req, nil); err == nil {
		t.Fatal("the unreachable provider was reached directly")
	}

	dial, err := e.TunnelDialer()
	if err != nil {
		t.Fatal(err)
	}
	got, err := subscription.FetchVia(fetchCtx, req, nil, dial)
	if err != nil {
		t.Fatalf("through the tunnel: %v", err)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Name != "From the provider" || got.Info.Total != 100 {
		t.Fatalf("result: %+v %+v", got.Profiles, got.Info)
	}

	e.Stop()
	if _, err := e.TunnelDialer(); !errors.Is(err, ErrNoTunnel) {
		t.Fatalf("a dialer after the tunnel stopped: %v", err)
	}
}
