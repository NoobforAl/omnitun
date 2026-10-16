package manager

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/server/store"
)

func xrayServerConfig(t *testing.T) (config, pub string) {
	t.Helper()
	priv, pub, err := store.RealityKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(`{
  "log": {"loglevel": "warning"},
  "inbounds": [
    {"tag": "vless-in", "port": 8443, "protocol": "vless",
     "settings": {"clients": [
       {"id": "11111111-1111-4111-8111-111111111111", "email": "alice@example.com", "flow": "xtls-rprx-vision"},
       {"id": "22222222-2222-4222-8222-222222222222", "email": "bob@example.com"},
       {"id": "33333333-3333-4333-8333-333333333333", "email": "bob@other.org"}],
       "decryption": "none"},
     "streamSettings": {"network": "tcp", "security": "reality",
       "realitySettings": {"dest": "www.cloudflare.com:443", "serverNames": ["www.cloudflare.com"], "privateKey": %q, "shortIds": ["deadbeef", "cafe"]}}},
    {"tag": "trojan-in", "port": 9443, "protocol": "trojan",
     "settings": {"clients": [{"password": "trojan-secret", "email": "carol"}]}},
    {"tag": "ss-in", "port": 8388, "protocol": "shadowsocks", "settings": {"method": "aes-256-gcm", "password": "x"}},
    {"tag": "api", "port": 10085, "protocol": "dokodemo-door", "settings": {"address": "127.0.0.1"}}
  ],
  "outbounds": [{"protocol": "freedom"}]
}`, priv), pub
}

func TestImportXrayCreatesServerAndKeepsLinksWorking(t *testing.T) {
	cfg, pub := xrayServerConfig(t)
	m := New(store.Open(tempDir(t)))

	if _, err := m.ImportXray(cfg, ImportXrayOptions{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("no host, no server: %v", err)
	}
	res, err := m.ImportXray(cfg, ImportXrayOptions{Host: "203.0.113.7", Name: "Moved"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.CreatedServer || !res.AdoptedReality || len(res.UsersAdded) != 4 {
		t.Fatalf("result = %+v", res)
	}
	// Names come from the e-mails; the second "bob" gets a suffix.
	want := []string{"alice", "bob", "bob-2", "carol"}
	for i, n := range want {
		if res.UsersAdded[i] != n {
			t.Fatalf("users = %v, want %v", res.UsersAdded, want)
		}
	}
	if len(res.Notes) < 2 {
		t.Errorf("notes = %v, want the shadowsocks and dokodemo inbounds mentioned", res.Notes)
	}

	// The old client links still work: same UUID, public key, short ID and site.
	set, err := m.Links("alice")
	if err != nil {
		t.Fatal(err)
	}
	vless := set.Links[0].URL
	for _, part := range []string{"vless://11111111-1111-4111-8111-111111111111@203.0.113.7:8443", "pbk=" + pub, "sid=deadbeef", "sni=www.cloudflare.com"} {
		if !strings.Contains(vless, part) {
			t.Errorf("link %q lacks %q", vless, part)
		}
	}
	st, _ := m.Store.Load()
	if st.User("carol").Password != "trojan-secret" {
		t.Errorf("trojan password not kept: %q", st.User("carol").Password)
	}
	if st.Server.Protocols[store.VLESSReality].Port != 8443 {
		t.Errorf("vless port = %d", st.Server.Protocols[store.VLESSReality].Port)
	}

	// Importing again adds nobody; names are reported as skipped.
	again, err := m.ImportXray(cfg, ImportXrayOptions{})
	if err != nil || len(again.UsersAdded) != 0 || len(again.UsersSkipped) != 4 || again.AdoptedReality || again.CreatedServer {
		t.Fatalf("second import = %+v err %v", again, err)
	}
}

func TestImportXrayIntoExistingServerKeepsItsReality(t *testing.T) {
	cfg, pub := xrayServerConfig(t)
	m := newManager(t)
	before, _ := m.Store.Load()
	old := before.Server.Reality.PublicKey

	if _, err := m.ImportXray(cfg, ImportXrayOptions{}); err != nil {
		t.Fatal(err)
	}
	after, _ := m.Store.Load()
	if after.Server.Reality.PublicKey != old || len(after.Users) != 4 {
		t.Fatalf("reality changed without being asked: %s -> %s, users %d", old, after.Server.Reality.PublicKey, len(after.Users))
	}
	if _, err := m.ImportXray(cfg, ImportXrayOptions{AdoptReality: true}); err != nil {
		t.Fatal(err)
	}
	if after, _ = m.Store.Load(); after.Server.Reality.PublicKey != pub {
		t.Fatal("AdoptReality did not take over the imported identity")
	}
}

func TestImportXrayRejectsGarbage(t *testing.T) {
	m := newManager(t)
	for name, text := range map[string]string{
		"not json":   "hello",
		"no clients": `{"inbounds":[{"protocol":"dokodemo-door","port":1}]}`,
		"bad key":    `{"inbounds":[{"protocol":"vless","port":443,"settings":{"clients":[]},"streamSettings":{"security":"reality","realitySettings":{"privateKey":"!!"}}}]}`,
	} {
		if _, err := m.ImportXray(text, ImportXrayOptions{}); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestImportXrayRefusesUnsafeSecrets(t *testing.T) {
	m := newManager(t)
	cfg := `{"inbounds":[
	  {"protocol":"trojan","port":443,"settings":{"clients":[
	    {"password":"good-secret.1","email":"ok"},
	    {"password":"bad\nremote evil 1194","email":"newline"},
	    {"password":"bad;host=x","email":"semicolon"},
	    {"password":"has space","email":"space"}]}},
	  {"protocol":"vless","port":8443,"settings":{"clients":[{"id":"not-a-uuid\nremote x","email":"badid"},{"id":"11111111-1111-4111-8111-111111111111","email":"fine"}]}}]}`
	res, err := m.ImportXray(cfg, ImportXrayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.UsersAdded) != 2 || res.UsersAdded[0] != "fine" && res.UsersAdded[0] != "ok" {
		t.Fatalf("added = %v, want only ok and fine", res.UsersAdded)
	}
	if len(res.Notes) < 4 {
		t.Errorf("notes = %v, want each refused client explained", res.Notes)
	}
	st, _ := m.Store.Load()
	for _, u := range st.Users {
		if u.Name == "newline" || u.Name == "semicolon" || u.Name == "space" || u.Name == "badid" {
			t.Errorf("user %q should not have been imported", u.Name)
		}
	}
}
