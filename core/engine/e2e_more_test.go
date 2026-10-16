//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// checkThrough imports link, connects the engine in proxy mode and fetches a
// page from a local web server through it.
func checkThrough(t *testing.T, link string) {
	t.Helper()
	var hits atomic.Int32
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data" {
			hits.Add(1)
		}
		io.WriteString(w, "ok")
	}))
	t.Cleanup(web.Close)

	res, err := importer.Import(link)
	if err != nil {
		t.Fatalf("import %s: %v", link, err)
	}
	p := res.Profiles[0]
	proxyPort := freePort(t)
	e := New(Config{DataDir: tempDir(t)})
	if err := e.Start(res.Profiles, singbox.Options{
		ListenPort: proxyPort, LogLevel: "error", Selected: p.ID,
		TestURL: web.URL + "/probe", ProxyPrivate: true,
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Stop() })
	deadline := time.Now().Add(15 * time.Second)
	for e.State().Status != StatusRunning {
		if st := e.State(); st.Status == StatusError || time.Now().After(deadline) {
			t.Fatalf("not connected: %+v", st)
		}
		time.Sleep(50 * time.Millisecond)
	}
	proxyURL, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(proxyPort))
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}, Timeout: 10 * time.Second}
	resp, err := client.Get(web.URL + "/data")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok" || hits.Load() != 1 {
		t.Fatalf("body %q, hits %d", body, hits.Load())
	}
}

func TestEndToEndHysteria(t *testing.T) {
	pki := newTestPKI(t)
	port := freeUDPPort(t)
	startServer(t, map[string]any{
		"type": "hysteria", "listen": "127.0.0.1", "listen_port": port,
		"up_mbps": 100, "down_mbps": 100, "obfs": "obfs-pw",
		"users": []any{map[string]any{"auth_str": "hy-pw"}},
		"tls":   map[string]any{"enabled": true, "certificate": pki.serverCert, "key": pki.serverKey, "alpn": []string{"hysteria"}},
	})
	checkThrough(t, fmt.Sprintf("hysteria://127.0.0.1:%d?protocol=udp&auth=hy-pw&peer=server&insecure=1&upmbps=10&downmbps=50&obfsParam=obfs-pw#hy1", port))
}

func TestEndToEndAnyTLS(t *testing.T) {
	pki := newTestPKI(t)
	port := freePort(t)
	startServer(t, map[string]any{
		"type": "anytls", "listen": "127.0.0.1", "listen_port": port,
		"users": []any{map[string]any{"password": "any-pw"}},
		"tls":   map[string]any{"enabled": true, "certificate": pki.serverCert, "key": pki.serverKey},
	})
	checkThrough(t, fmt.Sprintf("anytls://any-pw@127.0.0.1:%d/?sni=server&insecure=1#anytls", port))
}

func TestEndToEndShadowTLS(t *testing.T) {
	// The site ShadowTLS borrows its TLS handshake from.
	decoy := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	decoy.StartTLS()
	t.Cleanup(decoy.Close)
	_, decoyPort, _ := net.SplitHostPort(decoy.Listener.Addr().String())
	dp, _ := strconv.Atoi(decoyPort)

	port := freePort(t)
	startServerInbounds(t, []any{
		map[string]any{
			"type": "shadowtls", "listen": "127.0.0.1", "listen_port": port, "version": 3,
			"users":     []any{map[string]any{"password": "stls-pw"}},
			"handshake": map[string]any{"server": "127.0.0.1", "server_port": dp},
			"detour":    "ss-in",
		},
		map[string]any{
			"type": "shadowsocks", "tag": "ss-in",
			"method": "2022-blake3-aes-128-gcm", "password": "AAAAAAAAAAAAAAAAAAAAAA==",
		},
	})
	plugin := url.QueryEscape("shadow-tls;host=example.com;password=stls-pw;version=3;insecure=1")
	checkThrough(t, fmt.Sprintf("ss://2022-blake3-aes-128-gcm:%s@127.0.0.1:%d?plugin=%s#stls",
		url.QueryEscape("AAAAAAAAAAAAAAAAAAAAAA=="), port, plugin))
}

// startSSHServer runs an SSH server that only forwards TCP (direct-tcpip).
func startSSHServer(t *testing.T, user, password string) (port int, hostKey string) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{PasswordCallback: func(c ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
		if c.User() == user && string(pw) == password {
			return nil, nil
		}
		return nil, fmt.Errorf("denied")
	}}
	cfg.AddHostKey(signer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(c, cfg)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for nc := range chans {
					if nc.ChannelType() != "direct-tcpip" {
						nc.Reject(ssh.UnknownChannelType, "only direct-tcpip")
						continue
					}
					extra := nc.ExtraData()
					hostLen := binary.BigEndian.Uint32(extra)
					host := string(extra[4 : 4+hostLen])
					port := binary.BigEndian.Uint32(extra[4+hostLen:])
					target, err := net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(int(port))))
					if err != nil {
						nc.Reject(ssh.ConnectionFailed, err.Error())
						continue
					}
					ch, chReqs, err := nc.Accept()
					if err != nil {
						target.Close()
						continue
					}
					go ssh.DiscardRequests(chReqs)
					go func() { io.Copy(ch, target); ch.CloseWrite() }()
					go func() { io.Copy(target, ch); target.Close() }()
				}
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
}

func TestEndToEndSSH(t *testing.T) {
	port, hostKey := startSSHServer(t, "tun", "ssh-pw")
	// Host key pinned, as a careful user would.
	checkThrough(t, fmt.Sprintf("ssh://tun:ssh-pw@127.0.0.1:%d?hk=%s#ssh", port, url.QueryEscape(hostKey)))
}

// A pinned host key that doesn't match must stop the connection (no MITM).
func TestSSHRejectsWrongHostKey(t *testing.T) {
	port, _ := startSSHServer(t, "tun", "ssh-pw")
	_, other := startSSHServer(t, "x", "y")
	res, err := importer.Import(fmt.Sprintf("ssh://tun:ssh-pw@127.0.0.1:%d?hk=%s#ssh", port, url.QueryEscape(other)))
	if err != nil {
		t.Fatal(err)
	}
	e := New(Config{DataDir: tempDir(t)})
	if err := e.Start(res.Profiles, singbox.Options{
		ListenPort: freePort(t), LogLevel: "error", Selected: res.Profiles[0].ID, TestURL: "http://127.0.0.1:1/",
	}); err != nil {
		t.Fatal(err)
	}
	defer e.Stop()
	deadline := time.Now().Add(20 * time.Second)
	for e.State().Status != StatusError {
		if e.State().Status == StatusRunning || time.Now().After(deadline) {
			t.Fatalf("state: %+v", e.State())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Profiles exported as sing-box JSON (the server's Omnitun subscription) import back unchanged.
func TestNewProtocolsJSONRoundTrip(t *testing.T) {
	links := []string{
		"hysteria://h.example.com:443?auth=pw&peer=h.example.com&upmbps=20&downmbps=80&obfsParam=ob#hy1",
		"anytls://pw@a.example.com:443/?sni=a.example.com&fp=chrome#anytls",
		"ssh://u:pw@s.example.com:2222?hk=" + url.QueryEscape("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAbc") + "#ssh",
		"ss://2022-blake3-aes-128-gcm:" + url.QueryEscape("AAAAAAAAAAAAAAAAAAAAAA==") + "@t.example.com:443?plugin=" +
			url.QueryEscape("shadow-tls;host=www.microsoft.com;password=stls;version=3") + "#stls",
	}
	for _, link := range links {
		p, err := importer.ParseLink(link)
		if err != nil {
			t.Fatalf("%s: %v", link, err)
		}
		cfg, skipped, err := singbox.Build(p.AsSlice(), singbox.Options{})
		if err != nil || len(skipped) > 0 {
			t.Fatalf("%s: build %v %v", p.Name, err, skipped)
		}
		if err := CheckConfig(cfg); err != nil {
			t.Fatalf("%s: schema: %v", p.Name, err)
		}
		out, isEndpoint, extras, err := singbox.Outbound(p, p.Name)
		if err != nil {
			t.Fatal(err)
		}
		doc := map[string]any{"outbounds": append([]any{out}, func() []any {
			var x []any
			for _, e := range extras {
				x = append(x, e)
			}
			return x
		}()...)}
		if isEndpoint {
			t.Fatalf("%s: unexpected endpoint", p.Name)
		}
		raw, _ := json.Marshal(doc)
		res, err := importer.Import(string(raw))
		if err != nil || len(res.Profiles) != 1 {
			t.Fatalf("%s: reimport %v %+v", p.Name, err, res)
		}
		q := res.Profiles[0]
		if q.Protocol != p.Protocol || q.Server != p.Server || q.Port != p.Port || q.Password != p.Password ||
			q.Username != p.Username || q.Plugin != p.Plugin {
			t.Fatalf("%s: round trip changed the profile:\n%+v\n%+v", p.Name, p, q)
		}
	}
}
