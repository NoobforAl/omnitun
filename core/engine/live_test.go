//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"os"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
)

// TestLiveLatency measures a real profile over the internet. Opt-in:
//
//	OMNITUN_LIVE_LINK='ss://...' go test -run TestLiveLatency ./core/engine
func TestLiveLatency(t *testing.T) {
	link := os.Getenv("OMNITUN_LIVE_LINK")
	if link == "" {
		t.Skip("OMNITUN_LIVE_LINK not set")
	}
	res, err := importer.Import(link)
	if err != nil {
		t.Fatal(err)
	}
	e := New(Config{DataDir: tempDir(t), CertificateStore: "mozilla", OnLog: func(l, m string) { t.Log(l, m) }})
	results, err := e.TestLatency(t.Context(), res.Profiles, os.Getenv("OMNITUN_LIVE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		t.Logf("%s: %d ms %s", r.ID, r.Ms, r.Error)
		if r.Ms < 0 {
			t.Errorf("unreachable: %s", r.Error)
		}
	}
}

// TestLiveNetCheck checks the "is the device online" probe through a running
// tunnel's direct outbound. Opt-in: OMNITUN_LIVE_NET=1.
func TestLiveNetCheck(t *testing.T) {
	if os.Getenv("OMNITUN_LIVE_NET") == "" {
		t.Skip("OMNITUN_LIVE_NET not set")
	}
	res, err := importer.Import("ss://aes-128-gcm:x@127.0.0.1:1#dead")
	if err != nil {
		t.Fatal(err)
	}
	e := New(Config{DataDir: tempDir(t), SkipHandshake: true, OnLog: func(l, m string) { t.Log(l, m) }})
	if err := e.Start(res.Profiles, singbox.Options{LogLevel: "error"}); err != nil {
		t.Fatal(err)
	}
	defer e.Stop()
	e.mu.Lock()
	instance := e.instance
	e.mu.Unlock()
	if err := e.netCheck(t.Context(), instance); err != nil {
		t.Fatalf("netCheck via direct outbound: %v", err)
	}
}
