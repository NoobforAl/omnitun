//go:build with_quic && with_utls && with_wireguard && with_gvisor

package engine

import (
	"os"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
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
	e := New(Config{DataDir: t.TempDir(), CertificateStore: "mozilla", OnLog: func(l, m string) { t.Log(l, m) }})
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
