//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/singbox"
)

type testPKI struct {
	ca, serverCert, serverKey, clientCert, clientKey string
}

func newTestPKI(t *testing.T) testPKI {
	t.Helper()
	pemOf := func(typ string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}))
	}
	keyPEM := func(k *ecdsa.PrivateKey) string {
		der, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			t.Fatal(err)
		}
		return pemOf("PRIVATE KEY", der)
	}
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Omnitun Test CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := x509.ParseCertificate(caDER)
	leaf := func(serial int64, cn string, eku x509.ExtKeyUsage) (string, string) {
		k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: cn},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
			KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
			ExtKeyUsage: []x509.ExtKeyUsage{eku},
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &k.PublicKey, caKey)
		if err != nil {
			t.Fatal(err)
		}
		return pemOf("CERTIFICATE", der), keyPEM(k)
	}
	p := testPKI{ca: pemOf("CERTIFICATE", caDER)}
	p.serverCert, p.serverKey = leaf(2, "server", x509.ExtKeyUsageServerAuth)
	p.clientCert, p.clientKey = leaf(3, "client", x509.ExtKeyUsageClientAuth)
	return p
}

// staticKey returns a 2048-bit OpenVPN static key file (tls-auth / tls-crypt).
func staticKey(t *testing.T) string {
	raw := make([]byte, 256)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	h := hex.EncodeToString(raw)
	var b strings.Builder
	b.WriteString("-----BEGIN OpenVPN Static key V1-----\n")
	for i := 0; i < len(h); i += 32 {
		b.WriteString(h[i:i+32] + "\n")
	}
	b.WriteString("-----END OpenVPN Static key V1-----")
	return b.String()
}

func startServerEndpoint(t *testing.T, endpoint map[string]any) {
	t.Helper()
	ctx, cancel := context.WithCancel(include.Context(context.Background()))
	t.Cleanup(cancel)
	opts, err := parseOptions(ctx, map[string]any{
		"log":       map[string]any{"level": "error"},
		"endpoints": []any{endpoint},
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

// lanIPv4 returns a non-loopback IPv4 address of this machine.
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

func freeUDPPort(t *testing.T) int {
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

// TestEndToEndOpenVPN imports .ovpn files and sends HTTP requests through a
// local sing-box OpenVPN server.
func TestEndToEndOpenVPN(t *testing.T) {
	pki := newTestPKI(t)
	var hits atomic.Int32
	web := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data" {
			hits.Add(1)
		}
		io.WriteString(w, "hello over openvpn")
	})}
	// Packets inside the tunnel can't go to loopback (the server's netstack
	// drops them), so the web server listens on a LAN address.
	webLn, err := net.Listen("tcp", net.JoinHostPort(lanIPv4(t), "0"))
	if err != nil {
		t.Fatal(err)
	}
	go web.Serve(webLn)
	t.Cleanup(func() { web.Close() })

	cases := []struct {
		name    string
		network string
		wrap    string // tls-auth / tls-crypt
		creds   bool   // username/password instead of a client certificate
	}{
		{"tcp cert tls-crypt", "tcp", "tls-crypt", false},
		{"udp userpass tls-auth", "udp", "tls-auth", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := staticKey(t)
			port := freePort(t)
			if tc.network == "udp" {
				port = freeUDPPort(t)
			}
			serverTLS := map[string]any{"certificate": pki.serverCert, "key": pki.serverKey}
			wrap := map[string]any{"type": strings.ReplaceAll(tc.wrap, "-", "_"), "key": key}
			if tc.wrap == "tls-auth" {
				wrap["direction"] = "server"
			}
			serverTLS["control_wrap"] = wrap
			server := map[string]any{
				"type": "openvpn-server", "tag": "ovpn", "listen": "127.0.0.1", "listen_port": port,
				"network": tc.network, "address": []string{"10.9.0.1/24"},
				"push": map[string]any{"redirect_gateway": true},
			}
			if tc.creds {
				server["users"] = []any{map[string]any{"username": "alice", "password": "secret"}}
				serverTLS["verify_client_certificate"] = "none"
			} else {
				serverTLS["client_certificate"] = pki.ca
				serverTLS["verify_client_certificate"] = "require"
			}
			server["tls"] = serverTLS
			startServerEndpoint(t, server)

			var ovpn strings.Builder
			fmt.Fprintf(&ovpn, "client\ndev tun\nproto %s\nremote 127.0.0.1 %d\nnobind\npersist-key\nremote-cert-tls server\nverb 3\n", tc.network, port)
			fmt.Fprintf(&ovpn, "<ca>\n%s</ca>\n", pki.ca)
			if tc.creds {
				ovpn.WriteString("auth-user-pass\n")
			} else {
				fmt.Fprintf(&ovpn, "<cert>\n%s</cert>\n<key>\n%s</key>\n", pki.clientCert, pki.clientKey)
			}
			if tc.wrap == "tls-auth" {
				ovpn.WriteString("key-direction 1\n")
			}
			fmt.Fprintf(&ovpn, "<%s>\n%s\n</%s>\n", tc.wrap, key, tc.wrap)

			res, err := importer.Import(ovpn.String())
			if err != nil {
				t.Fatal(err)
			}
			p := res.Profiles[0]
			if p.Protocol != profile.OpenVPN {
				t.Fatalf("protocol = %s", p.Protocol)
			}
			if tc.creds {
				// Without credentials the profile can't connect yet.
				if _, skipped, _ := singbox.Build(p.AsSlice(), singbox.Options{}); len(skipped) == 0 {
					t.Fatal("profile without credentials should be skipped")
				}
				p.Username, p.Password = "alice", "secret"
			}

			proxyPort := freePort(t)
			var running atomic.Bool
			e := New(Config{DataDir: tempDir(t), OnState: func(s State) {
				if s.Status == StatusRunning {
					running.Store(true)
				}
			}})
			err = e.Start(res.Profiles, singbox.Options{
				ListenPort: proxyPort, LogLevel: "error", Selected: p.ID,
				TestURL:      "http://" + webLn.Addr().String() + "/probe",
				ProxyPrivate: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { e.Stop() })
			deadline := time.Now().Add(20 * time.Second)
			for !running.Load() {
				if time.Now().After(deadline) {
					t.Fatalf("not connected: %+v", e.State())
				}
				time.Sleep(100 * time.Millisecond)
			}

			proxyURL, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(proxyPort))
			client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}, Timeout: 10 * time.Second}
			before := hits.Load()
			resp, err := client.Get("http://" + webLn.Addr().String() + "/data")
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if string(body) != "hello over openvpn" || hits.Load() != before+1 {
				t.Fatalf("body = %q, hits = %d", body, hits.Load()-before)
			}
		})
	}
}
