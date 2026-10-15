package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// State files written before OpenVPN existed get keys and disabled OpenVPN
// protocols on load, and keep them across loads.
func TestMigrateAddsOpenVPN(t *testing.T) {
	s := Open(t.TempDir())
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
	s := Open(t.TempDir())
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
