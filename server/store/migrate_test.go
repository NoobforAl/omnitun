package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// tempDir is t.TempDir with a cleanup that retries: on Windows a virus
// scanner or a just-released file lock can briefly keep a file open, which
// fails t.TempDir's own cleanup.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "omnitun-store-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 40 && os.RemoveAll(dir) != nil; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	})
	return dir
}

// State files written before OpenVPN existed get keys and disabled OpenVPN
// protocols on load, and keep them across loads.
func TestMigrateAddsOpenVPN(t *testing.T) {
	s := Open(tempDir(t))
	old := map[string]any{
		"version": 1,
		"server": map[string]any{
			"name": "Old", "public_host": "1.2.3.4",
			"protocols": map[string]any{"trojan": map[string]any{"enabled": true, "port": 8443}},
		},
		"users": []any{},
	}
	data, _ := json.Marshal(old)
	if err := os.WriteFile(filepath.Join(s.Dir, "state.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.Server.OpenVPN.CA == "" || st.Server.OpenVPN.TLSCrypt == "" {
		t.Fatal("no OpenVPN keys after migration")
	}
	if p := st.Server.Protocols[OpenVPNTCP]; p == nil || p.Enabled || p.Port != 1194 {
		t.Fatalf("openvpn-tcp = %+v", p)
	}
	if !st.Server.Protocols[Trojan].Enabled {
		t.Fatal("existing protocol changed")
	}
	again, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if again.Server.OpenVPN.CA != st.Server.OpenVPN.CA {
		t.Fatal("keys regenerated on the second load")
	}
}

// Old 10-year self-signed certificates are replaced; fresh ones are kept.
func TestRenewSelfSignedCert(t *testing.T) {
	s := Open(tempDir(t))
	st := &State{Server: Server{SelfSigned: true, CertFile: "tls.crt", KeyFile: "tls.key", PublicHost: "1.2.3.4"}}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if renewed, err := RenewSelfSignedCert(s, st); err != nil || !renewed {
		t.Fatalf("missing cert: renewed=%v err=%v", renewed, err)
	}
	if renewed, _ := RenewSelfSignedCert(s, st); renewed {
		t.Fatal("fresh certificate renewed again")
	}
}

// Updates from several goroutines (standing in for the CLI, the API and the
// running server) must not lose each other's changes.
func TestConcurrentUpdatesKeepEverything(t *testing.T) {
	s := Open(tempDir(t))
	if _, err := Init(s, InitOptions{PublicHost: "203.0.113.7"}); err != nil {
		t.Fatal(err)
	}
	const n = 12
	errs := make(chan error, n)
	for i := range n {
		go func() {
			errs <- s.Update(func(st *State) error {
				_, err := AddUser(st, fmt.Sprintf("user%d", i))
				return err
			})
		}()
	}
	for range n {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	st, err := s.Load()
	if err != nil || len(st.Users) != n {
		t.Fatalf("users = %d, want %d (err %v)", len(st.Users), n, err)
	}
}
