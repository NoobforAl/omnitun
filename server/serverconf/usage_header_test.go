package serverconf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NoobforAl/omnitun/server/stats"
	"github.com/NoobforAl/omnitun/server/store"
)

// Clients show the subscription's usage bar from Subscription-Userinfo: it
// must carry this month's real traffic, the quota and the expiry.
func TestSubscriptionUserinfoHasUsage(t *testing.T) {
	s := store.Open(t.TempDir())
	if _, err := store.Init(s, store.InitOptions{PublicHost: "203.0.113.7"}); err != nil {
		t.Fatal(err)
	}
	var token, path string
	if err := s.Update(func(st *store.State) error {
		u, err := store.AddUser(st, "alice")
		u.QuotaGB = 2
		token, path = u.SubToken, st.Server.Subscription.Path
		return err
	}); err != nil {
		t.Fatal(err)
	}
	tr := stats.NewTracker(s.Dir)
	tr.Add("alice", 1000, 5000)
	if err := tr.Save(); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	SubscriptionHandler(s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+path+"/"+token, nil))
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	got := rec.Header().Get("Subscription-Userinfo")
	for _, want := range []string{"upload=1000", "download=5000", "total=2147483648"} {
		if !strings.Contains(got, want) {
			t.Errorf("Subscription-Userinfo = %q, missing %q", got, want)
		}
	}
}

func TestSubscriptionServesConfigFiles(t *testing.T) {
	s := store.Open(t.TempDir())
	if _, err := store.Init(s, store.InitOptions{PublicHost: "203.0.113.7"}); err != nil {
		t.Fatal(err)
	}
	var token, path string
	s.Update(func(st *store.State) error {
		u, err := store.AddUser(st, "alice")
		token, path = u.SubToken, st.Server.Subscription.Path
		return err
	})
	get := func(file string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		SubscriptionHandler(s).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+path+"/"+token+"/"+file, nil))
		return rec
	}
	if rec := get("wireguard.conf"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "[Interface]") ||
		!strings.Contains(rec.Header().Get("Content-Disposition"), "wireguard.conf") {
		t.Fatalf("wireguard.conf: %d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}
	if rec := get("openvpn-udp.ovpn"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "auth-user-pass") {
		t.Fatalf("openvpn-udp.ovpn: %d", rec.Code)
	}
	for _, bad := range []string{"trojan.conf", "wireguard.ovpn", "wireguard", "../state.json"} {
		if rec := get(bad); rec.Code != 404 {
			t.Errorf("%s -> %d, want 404", bad, rec.Code)
		}
	}
}
