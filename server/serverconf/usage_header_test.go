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
