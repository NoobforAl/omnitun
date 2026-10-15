//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// startServer runs a sing-box server with the given inbound on localhost.
func startServer(t *testing.T, inbound map[string]any) {
	t.Helper()
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	t.Cleanup(cancel)
	opts, err := parseOptions(ctx, map[string]any{
		"log":       map[string]any{"level": "error"},
		"inbounds":  []any{inbound},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
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
}

// TestEndToEndShadowsocks sends an HTTP request through the client engine's
// proxy port, over Shadowsocks, to a local web server.
func TestEndToEndShadowsocks(t *testing.T) {
	var hits atomic.Int32
	web := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data" {
			hits.Add(1)
		}
		io.WriteString(w, "hello through the tunnel")
	})}
	webLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go web.Serve(webLn)
	t.Cleanup(func() { web.Close() })

	ssPort := freePort(t)
	startServer(t, map[string]any{
		"type": "shadowsocks", "listen": "127.0.0.1", "listen_port": ssPort,
		"method": "aes-128-gcm", "password": "test-pass", "network": "tcp",
	})

	link := fmt.Sprintf("ss://aes-128-gcm:test-pass@127.0.0.1:%d#local", ssPort)
	res, err := importer.Import(link)
	if err != nil {
		t.Fatal(err)
	}
	proxyPort := freePort(t)
	e := New(Config{DataDir: t.TempDir(), CertificateStore: "mozilla"})
	err = e.Start(res.Profiles, singbox.Options{
		ListenPort:   proxyPort,
		LogLevel:     "error",
		Selected:     res.Profiles[0].ID,
		TestURL:      "http://" + webLn.Addr().String() + "/probe",
		ProxyPrivate: true, // the web server is on loopback; force it through the tunnel
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Stop() })

	proxyURL, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(proxyPort))
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   10 * time.Second,
	}
	resp, err := client.Get("http://" + webLn.Addr().String() + "/data")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "hello through the tunnel" {
		t.Fatalf("body = %q", body)
	}
	if hits.Load() != 1 {
		t.Fatalf("web server hits = %d", hits.Load())
	}

	if err := e.Select(singbox.TagAuto); err != nil {
		t.Fatalf("select auto: %v", err)
	}
	if err := e.Select("missing"); err == nil {
		t.Fatal("selecting an unknown profile should fail")
	}

	// Latency through the Shadowsocks server, plus one dead profile.
	dead, err := importer.ParseLink(fmt.Sprintf("ss://aes-128-gcm:x@127.0.0.1:%d#dead", freePort(t)))
	if err != nil {
		t.Fatal(err)
	}
	results, err := e.TestLatency(t.Context(), append(res.Profiles, dead), "http://"+webLn.Addr().String()+"/")
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Ms < 0 {
		t.Fatalf("live profile unreachable: %+v", results[0])
	}
	if results[1].Ms != -1 {
		t.Fatalf("dead profile reported reachable: %+v", results[1])
	}
}

// startServerInbounds runs a sing-box server with several inbounds (for
// layered protocols such as ShadowTLS + Shadowsocks).
func startServerInbounds(t *testing.T, inbounds []any) {
	t.Helper()
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	t.Cleanup(cancel)
	opts, err := parseOptions(ctx, map[string]any{
		"log":       map[string]any{"level": "error"},
		"inbounds":  inbounds,
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
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
}
