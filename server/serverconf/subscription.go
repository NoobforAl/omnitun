package serverconf

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NoobforAl/omnitun/server/store"
)

// SubscriptionHandler serves GET /<path>/<token> with the user's links,
// base64-encoded, plus the standard usage/expiry headers. It reloads the
// state on every request, so user changes apply immediately.
func SubscriptionHandler(s *store.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		st, err := s.Load()
		if err != nil {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 2 || parts[0] != st.Server.Subscription.Path {
			http.NotFound(w, r)
			return
		}
		var user *store.User
		for _, u := range st.Users {
			if u.SubToken != "" && u.SubToken == parts[1] {
				user = u
				break
			}
		}
		if user == nil {
			http.NotFound(w, r)
			return
		}
		if !user.Active(time.Now()) {
			http.Error(w, "account disabled or expired", http.StatusForbidden)
			return
		}
		var lines []string
		for _, l := range Links(st, user) {
			lines = append(lines, l.URL)
		}
		info := ""
		if user.QuotaGB > 0 {
			info += fmt.Sprintf("total=%d; ", int64(user.QuotaGB*(1<<30)))
		}
		if user.ExpiresAt != nil {
			info += fmt.Sprintf("expire=%d; ", user.ExpiresAt.Unix())
		}
		if info != "" {
			w.Header().Set("Subscription-Userinfo", "upload=0; download=0; "+strings.TrimSuffix(info, "; "))
		}
		w.Header().Set("Profile-Update-Interval", "12")
		w.Header().Set("Profile-Title", "base64:"+base64.StdEncoding.EncodeToString([]byte(st.Server.Name)))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprint(w, base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n"))))
	})
}
