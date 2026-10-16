package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func num(t *testing.T, a AmneziaWG, key string) int {
	t.Helper()
	n, err := strconv.Atoi(a.Params[key])
	if err != nil {
		t.Fatalf("%s = %q", key, a.Params[key])
	}
	return n
}

// Generated parameters follow AmneziaWG's rules for every server.
func TestNewAmneziaWGParametersAreValid(t *testing.T) {
	for range 300 {
		a, err := NewAmneziaWG()
		if err != nil {
			t.Fatal(err)
		}
		if a.PrivateKey == "" || a.PublicKey == "" || a.PrivateKey == a.PublicKey {
			t.Fatalf("keys: %+v", a)
		}
		jc, jmin, jmax := num(t, a, "jc"), num(t, a, "jmin"), num(t, a, "jmax")
		if jc < 1 || jc > 128 || jmin >= jmax || jmax > 1280 {
			t.Fatalf("junk parameters jc=%d jmin=%d jmax=%d", jc, jmin, jmax)
		}
		s1, s2 := num(t, a, "s1"), num(t, a, "s2")
		if s1 > 1132 || s2 > 1188 || s1+56 == s2 {
			t.Fatalf("padding s1=%d s2=%d (init and response would be the same size)", s1, s2)
		}
		seen := map[int]bool{}
		for _, k := range []string{"h1", "h2", "h3", "h4"} {
			h := num(t, a, k)
			if h <= 4 || seen[h] {
				t.Fatalf("header %s=%d is WireGuard's own or repeated", k, h)
			}
			seen[h] = true
		}
	}
}

// A new server and an old state file both end up with an AmneziaWG identity
// that stays the same across loads, and the protocol is off.
func TestAmneziaWGIdentityIsCreatedOnceAndOff(t *testing.T) {
	s := Open(tempDir(t))
	old := map[string]any{
		"version": 1,
		"server":  map[string]any{"name": "Old", "public_host": "1.2.3.4", "protocols": map[string]any{}},
		"users":   []any{},
	}
	data, _ := json.Marshal(old)
	if err := os.WriteFile(filepath.Join(s.Dir, "state.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.Server.AmneziaWG.PrivateKey == "" || len(st.Server.AmneziaWG.Params) != 9 {
		t.Fatalf("no AmneziaWG identity after migration: %+v", st.Server.AmneziaWG)
	}
	if p := st.Server.Protocols[AmneziaWGKey]; p == nil || p.Enabled || p.Port != 51821 {
		t.Fatalf("amneziawg = %+v", p)
	}
	again, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if again.Server.AmneziaWG.PublicKey != st.Server.AmneziaWG.PublicKey {
		t.Fatal("the identity was regenerated on the second load")
	}
}
