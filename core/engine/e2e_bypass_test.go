//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// Every shape of a bypass list produces a configuration sing-box accepts.
func TestBypassListConfigsPassSchema(t *testing.T) {
	list, issues := singbox.ParseBypassList("example.com\nfull:a.example.com\nkeyword:bank\nregexp:^ads\\.\n203.0.113.0/24\n2001:db8::/32\n198.51.100.7\n")
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	for _, link := range testLinks() {
		p, err := importer.ParseLink(link)
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []singbox.Mode{singbox.ModeProxy, singbox.ModeTUN} {
			for _, only := range []bool{false, true} {
				cfg, _, err := singbox.Build(p.AsSlice(), singbox.Options{Mode: mode, Bypass: list, BypassOnly: only})
				if err != nil {
					t.Fatalf("%s: %v", p.Name, err)
				}
				if err := CheckConfig(cfg); err != nil {
					t.Fatalf("%s (%s, only=%v): sing-box rejected the config: %v", p.Name, mode, only, err)
				}
			}
		}
	}
}

// The list really decides where traffic goes: the VPN here is dead (nothing
// listens at its server), so a request only works if it goes around it.
func TestBypassListDecidesTheRoute(t *testing.T) {
	serve := func(host, body string) string {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Skipf("cannot listen on %s: %v", host, err)
		}
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })}
		go srv.Serve(ln)
		t.Cleanup(func() { srv.Close() })
		return ln.Addr().String()
	}
	one := serve("127.0.0.1", "one")
	two := serve("127.0.0.2", "two")

	dead := freePort(t) // a Shadowsocks server that is not there
	res, err := importer.Import(fmt.Sprintf("ss://aes-128-gcm:pw@127.0.0.1:%d#dead", dead))
	if err != nil {
		t.Fatal(err)
	}

	fetch := func(proxyPort int, addr string) (string, error) {
		proxy, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(proxyPort))
		client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}, Timeout: 6 * time.Second}
		resp, err := client.Get("http://" + addr + "/")
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return string(b), nil
	}
	run := func(list string, only bool) int {
		parsed, _ := singbox.ParseBypassList(list)
		port := freePort(t)
		e := New(Config{DataDir: tempDir(t), SkipHandshake: true})
		err := e.Start(res.Profiles, singbox.Options{
			ListenPort: port, LogLevel: "error", Selected: res.Profiles[0].ID, ProxyPrivate: true,
			Bypass: parsed, BypassOnly: only,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { e.Stop() })
		return port
	}

	// No list: everything goes into the dead VPN.
	port := run("", false)
	if body, err := fetch(port, one); err == nil {
		t.Fatalf("reached the server through a dead VPN: %q", body)
	}

	// A list with 127.0.0.1: that one goes direct, the other still uses the VPN.
	port = run("127.0.0.1\n", false)
	if body, err := fetch(port, one); err != nil || body != "one" {
		t.Fatalf("a bypassed address did not go direct: %q %v", body, err)
	}
	if body, err := fetch(port, two); err == nil {
		t.Fatalf("an address that is not listed went direct: %q", body)
	}

	// Only these use the VPN: 127.0.0.1 is listed (dead VPN: fails), the rest goes direct.
	port = run("127.0.0.1\n", true)
	if body, err := fetch(port, one); err == nil {
		t.Fatalf("a listed address went direct in 'only these' mode: %q", body)
	}
	if body, err := fetch(port, two); err != nil || body != "two" {
		t.Fatalf("an unlisted address did not go direct in 'only these' mode: %q %v", body, err)
	}
}
