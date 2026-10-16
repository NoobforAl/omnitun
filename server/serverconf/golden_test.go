package serverconf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/server/store"
)

// goldenState is a server with fixed keys and one user, so generated output
// is reproducible.
func goldenState(t *testing.T) (*store.State, *store.Store) {
	t.Helper()
	s := store.Open(t.TempDir())
	st := &store.State{
		Version: 1,
		Server: store.Server{
			Name: "Lab", PublicHost: "203.0.113.7",
			Protocols: map[string]*store.Proto{},
			Reality: store.Reality{PrivateKey: "REALITYPRIV", PublicKey: "REALITYPUB", ShortID: "ab12cd34",
				Handshake: "www.microsoft.com:443"},
			SSServerKey: "c2VydmVyLWtleS0xMjM0NQ==",
			CertFile:    "tls.crt", KeyFile: "tls.key", SelfSigned: true,
			Subscription: store.Subscription{Enabled: true, Listen: "0.0.0.0", Port: 2096, Path: "subpath"},
			OpenVPN:      store.OpenVPN{CA: "CA-PEM\n", ServerCert: "SERVER-CERT", ServerKey: "SERVER-KEY", TLSCrypt: "TLS-CRYPT\n"},
			WireGuard:    store.WireGuard{PrivateKey: "c2VydmVyLXByaXZhdGU=", PublicKey: "c2VydmVyLXB1YmxpYw=="},
			AmneziaWG: store.AmneziaWG{PrivateKey: "YXdnLXByaXZhdGU=", PublicKey: "YXdnLXB1YmxpYw==", Params: map[string]string{
				"jc": "5", "jmin": "42", "jmax": "105", "s1": "30", "s2": "94",
				"h1": "1043474670", "h2": "1207032647", "h3": "682630448", "h4": "1230328733",
			}},
		},
		Users: []*store.User{{
			Name: "alice", UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Password: "alicepassword",
			SSKey: "YWxpY2Uta2V5LTEyMzQ1Ng==", SubToken: "alicetoken", Enabled: true,
			WGPrivateKey: "YWxpY2UtcHJpdmF0ZQ==", WGPublicKey: "YWxpY2UtcHVibGlj", WGAddress: "10.30.0.2/32",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}},
	}
	for _, p := range store.AllProtocols {
		st.Server.Protocols[p] = &store.Proto{Enabled: true, Port: store.DefaultPorts[p]}
	}
	os.WriteFile(s.Path("tls.crt"), []byte("CERT-PEM\n"), 0o600)
	return st, s
}

// TestGoldenOutput pins what the server generates (sing-box config, share
// links and the client profiles of the subscription). Regenerate the files
// with UPDATE_GOLDEN=1 only for a deliberate change.
func TestGoldenOutput(t *testing.T) {
	st, s := goldenState(t)
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	sub, err := SingBoxSubscription(st, s, st.Users[0])
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]any{
		"build":        Build(st, s, now),
		"links":        Links(st, s, st.Users[0]),
		"subscription": json.RawMessage(sub),
		"ovpn_udp":     mustOVPN(t, st, store.OpenVPNUDP),
	}
	raw, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// Paths of the temp dir differ per run.
	raw = []byte(replaceAll(string(raw), filepath.ToSlash(s.Dir), "$DIR"))
	raw = []byte(replaceAll(string(raw), jsonEscape(s.Dir), "$DIR"))
	golden := filepath.Join("testdata", "golden.json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	json.Unmarshal(want, &a)
	json.Unmarshal(raw, &b)
	if !deepEqualJSON(a, b) {
		os.WriteFile(filepath.Join(t.TempDir(), "got.json"), raw, 0o644)
		t.Fatalf("generated output differs from testdata/golden.json; run with UPDATE_GOLDEN=1 if intended.\n got: %s", raw)
	}
}

func mustOVPN(t *testing.T, st *store.State, proto string) string {
	t.Helper()
	text, err := OVPN(st, st.Users[0], proto)
	if err != nil {
		t.Fatal(err)
	}
	return text
}
