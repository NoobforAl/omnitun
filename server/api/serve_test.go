package api

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/store"
	"github.com/NoobforAl/omnitun/server/tlsfiles"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestServeOverTLS(t *testing.T) {
	dir := tempDir(t)
	cert, key := filepath.Join(dir, "tls.crt"), filepath.Join(dir, "tls.key")
	if err := store.WriteSelfSignedCert(cert, key, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	m := manager.New(store.Open(dir))
	h := New(Options{Manager: m, Token: func() string { return "t" }})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	port := freePort(t)
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, "127.0.0.1", port, h, tlsfiles.New(filepath.Join(dir, "acme"), "", cert, key).TLSConfig(), t.Logf)
	}()

	client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	var resp *http.Response
	var err error
	for range 40 {
		if resp, err = client.Get("https://127.0.0.1:" + strconv.Itoa(port) + "/health"); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("HTTPS request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.TLS == nil {
		t.Fatalf("status %d tls %v", resp.StatusCode, resp.TLS != nil)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not stop")
	}
}
