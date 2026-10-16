//go:build with_quic && with_utls && with_wireguard && with_gvisor

package control

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
)

// The exit lookup goes through the tunnel: the "service" only exists as a name
// the tunnel's server maps to a local web server, so it cannot be reached
// directly, and what it answers is parsed.
func TestExitInfoThroughTheTunnel(t *testing.T) {
	web := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ip":"203.0.113.77","country":"nl"}`)
	})}
	webLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go web.Serve(webLn)
	t.Cleanup(func() { web.Close() })

	ssPort := freePort(t)
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	t.Cleanup(cancel)
	raw := fmt.Sprintf(`{
	  "log": {"level": "error"},
	  "inbounds": [{"type": "shadowsocks", "listen": "127.0.0.1", "listen_port": %d, "method": "aes-128-gcm", "password": "test-pass", "network": "tcp"}],
	  "outbounds": [{"type": "direct", "tag": "direct"}],
	  "route": {"rules": [{"action": "route-options", "override_address": "127.0.0.1", "override_port": %d}]}
	}`, ssPort, webLn.Addr().(*net.TCPAddr).Port)
	opts, err := sjson.UnmarshalExtendedContext[option.Options](ctx, []byte(raw))
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
	s, addr := newTestServer(t, ServerConfig{Grace: -1})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	err = c.Call(t.Context(), "start", StartParams{Profiles: res.Profiles, ListenPort: freePort(t), Selected: res.Profiles[0].ID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for s.Engine().State().Status != engine.StatusRunning {
		if time.Now().After(deadline) {
			t.Fatalf("tunnel did not come up: %+v", s.Engine().State())
		}
		time.Sleep(50 * time.Millisecond)
	}

	var info ExitInfo
	if err := c.Call(t.Context(), "exit_info", ExitInfoParams{URL: "http://geo.invalid/"}, &info); err != nil {
		t.Fatal(err)
	}
	if info.IP != "203.0.113.77" || info.Country != "NL" {
		t.Fatalf("exit info: %+v", info)
	}
	var ce *Error
	if err := c.Call(t.Context(), "exit_info", ExitInfoParams{URL: "ftp://x"}, nil); err == nil {
		t.Fatal("a non-http address was accepted")
	} else if !errorsAs(err, &ce) || ce.Code != CodeLookupFailed {
		t.Fatalf("err = %v", err)
	}
}
