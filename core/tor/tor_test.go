package tor

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestBridgeLines(t *testing.T) {
	obfs4 := "Bridge obfs4 192.0.2.1:443 D9A82D2F9C2F65A18407B1D2B764F130847F8B5D cert=abc iat-mode=0"
	if Transport(obfs4) != "obfs4" || !IsBridgeLine(obfs4) {
		t.Fatalf("obfs4: %q %v", Transport(obfs4), IsBridgeLine(obfs4))
	}
	vanilla := "192.0.2.2:9001 D9A82D2F9C2F65A18407B1D2B764F130847F8B5D"
	if Transport(vanilla) != "" || !IsBridgeLine(vanilla) {
		t.Fatal("vanilla bridge not recognized")
	}
	if IsBridgeLine("vless://x@y:1") || IsBridgeLine("conjure 192.0.2.3:80 x") {
		t.Fatal("non-bridge accepted")
	}
	got := TransportsFor([]string{obfs4, vanilla, "snowflake 192.0.2.3:80 X", "obfs4 192.0.2.4:1 Y"})
	if len(got) != 2 || got[0] != "obfs4" || got[1] != "snowflake" {
		t.Fatalf("transports: %v", got)
	}
	for _, kind := range BuiltinKinds() {
		bs := Builtin(kind)
		if len(bs) == 0 {
			t.Fatalf("no built-in %s bridges", kind)
		}
		for _, b := range bs {
			if !IsBridgeLine(b) {
				t.Errorf("built-in %s bridge not recognized: %s", kind, b)
			}
		}
	}
}

func TestStartTransports(t *testing.T) {
	pts, err := StartTransports(t.TempDir(), []string{"obfs4", "snowflake", "webtunnel", "meek_lite"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pts.Close()
	if len(pts.Addrs) != 4 {
		t.Fatalf("addrs: %v", pts.Addrs)
	}
	if _, err := StartTransports(t.TempDir(), []string{"nope"}, nil); err == nil {
		t.Fatal("unknown transport accepted")
	}
}

// TestLiveTor bootstraps through real built-in bridges when
// OMNITUN_LIVE_TOR is set to a kind (obfs4, snowflake, meek, or "direct")
// and OMNITUN_TOR_BIN points at the omnitun-tor binary.
func TestLiveTor(t *testing.T) {
	kind := os.Getenv("OMNITUN_LIVE_TOR")
	if kind == "" {
		t.Skip("set OMNITUN_LIVE_TOR=obfs4|snowflake|meek|direct and OMNITUN_TOR_BIN")
	}
	var bridges []string
	if kind != "direct" {
		bridges = Builtin(kind)
	}
	dir := os.Getenv("OMNITUN_LIVE_TOR_DATA")
	if dir == "" {
		dir = t.TempDir()
	}
	p, err := Start(Config{
		DataDir: dir, Bridges: bridges,
		OnBootstrap: func(pct int, status string) { t.Logf("bootstrap %d%%: %s", pct, status) },
		Logf:        t.Logf,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	if err := p.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	proxy, _ := url.Parse("socks5h://" + p.SocksAddr)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}, Timeout: 90 * time.Second}
	resp, err := client.Get("https://check.torproject.org/api/ip")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		IsTor bool   `json:"IsTor"`
		IP    string `json:"IP"`
	}
	if json.Unmarshal(body, &r) != nil || !r.IsTor {
		t.Fatalf("not via Tor: %s", body)
	}
	t.Logf("Tor exit %s", r.IP)
}

func TestTransportLogKeepsOnlyProblems(t *testing.T) {
	var got []string
	SetTransportLog(func(line string) { got = append(got, line) })
	defer SetTransportLog(nil)
	for _, line := range []string{
		"WebRTC: Collecting a new Snowflake. Currently at [1/1]",
		"Rendezvous using Broker at: https://snowflake-broker.example/",
		"Domain fronting using a randomly selected domain from: [a b]",
		"NAT Type: restricted",
		"Received answer: {\"sdp\":\"...\"}",
	} {
		log.Print(line)
	}
	if len(got) != 0 {
		t.Fatalf("routine lines got through: %v", got)
	}
	log.Print("WebRTC: Unable to get a snowflake\nrendezvous failed: timeout")
	if len(got) != 2 || got[0] != "transport: WebRTC: Unable to get a snowflake" || got[1] != "transport: rendezvous failed: timeout" {
		t.Fatalf("problem lines: %q", got)
	}
}
