package store

import (
	"encoding/base64"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

func TestWireGuardKeysAndAddresses(t *testing.T) {
	s := Open(tempDir(t))
	st, err := Init(s, InitOptions{PublicHost: "203.0.113.7"})
	if err != nil {
		t.Fatal(err)
	}
	if raw, err := base64.StdEncoding.DecodeString(st.Server.WireGuard.PrivateKey); err != nil || len(raw) != 32 {
		t.Fatalf("server key: %q (%v)", st.Server.WireGuard.PrivateKey, err)
	}

	seen := map[string]bool{}
	for i := range 5 {
		u, err := AddUser(st, string(rune('a'+i))+"user")
		if err != nil {
			t.Fatal(err)
		}
		if len(u.WGPrivateKey) == 0 || len(u.WGPublicKey) == 0 || u.WGPrivateKey == u.WGPublicKey {
			t.Fatalf("keys: %+v", u)
		}
		pf, err := netip.ParsePrefix(u.WGAddress)
		if err != nil || pf.Bits() != 32 || !netip.MustParsePrefix(WireGuardSubnet).Contains(pf.Addr()) {
			t.Fatalf("address %q", u.WGAddress)
		}
		if pf.Addr().String() == "10.30.0.1" {
			t.Fatal("a user got the server's address")
		}
		if seen[u.WGAddress] {
			t.Fatalf("address %s given twice", u.WGAddress)
		}
		seen[u.WGAddress] = true
	}

	// A deleted user's address is reused; resetting keeps the address and
	// rotates the keys.
	first := st.Users[0]
	oldAddr, oldKey := first.WGAddress, first.WGPrivateKey
	ResetCredentials(first)
	if first.WGAddress != oldAddr || first.WGPrivateKey == oldKey {
		t.Fatalf("reset: address %s -> %s, key unchanged: %v", oldAddr, first.WGAddress, first.WGPrivateKey == oldKey)
	}
	DeleteUser(st, st.Users[2].Name)
	again, _ := AddUser(st, "reuse")
	if !seen[again.WGAddress] {
		t.Fatalf("a freed address was not reused: %s", again.WGAddress)
	}
}

// A state file from before WireGuard existed gets a server key and keys and
// addresses for every user when it is loaded.
func TestMigrationGivesWireGuardToOldStates(t *testing.T) {
	dir := tempDir(t)
	s := Open(dir)
	st, err := Init(s, InitOptions{PublicHost: "203.0.113.7"})
	if err != nil {
		t.Fatal(err)
	}
	AddUser(st, "alice")
	AddUser(st, "bob")
	// Rewrite the file as an old version would have: no WireGuard anywhere.
	raw, _ := json.Marshal(st)
	var m map[string]any
	json.Unmarshal(raw, &m)
	delete(m["server"].(map[string]any), "wireguard")
	for _, u := range m["users"].([]any) {
		um := u.(map[string]any)
		delete(um, "wg_private_key")
		delete(um, "wg_public_key")
		delete(um, "wg_address")
	}
	old, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), old, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server.WireGuard.PrivateKey == "" || loaded.Server.WireGuard.PublicKey == "" {
		t.Fatal("no server key after migration")
	}
	addrs := map[string]bool{}
	for _, u := range loaded.Users {
		if u.WGPrivateKey == "" || u.WGAddress == "" || addrs[u.WGAddress] {
			t.Fatalf("user %s: %+v", u.Name, u)
		}
		addrs[u.WGAddress] = true
	}
	// And it was saved: the next load sees the same keys.
	again, _ := s.Load()
	if again.Server.WireGuard.PrivateKey != loaded.Server.WireGuard.PrivateKey || again.Users[0].WGAddress != loaded.Users[0].WGAddress {
		t.Fatal("the migration was not persisted")
	}
}
