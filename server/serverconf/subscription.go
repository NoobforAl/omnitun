package serverconf

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NoobforAl/omnitun/core/singbox"
	"github.com/NoobforAl/omnitun/server/stats"
	"github.com/NoobforAl/omnitun/server/store"
)

func wantsSingBox(r *http.Request) bool {
	switch r.URL.Query().Get("format") {
	case "sing-box", "singbox", "json":
		return true
	case "base64", "links":
		return false
	}
	ua := strings.ToLower(r.UserAgent())
	return strings.Contains(ua, "omnitun") || strings.Contains(ua, "sing-box") || strings.Contains(ua, "sfa") || strings.Contains(ua, "sfi")
}

// SingBoxSubscription is the user's profiles as a sing-box config (outbounds
// and endpoints), including OpenVPN, which has no share-link format.
func SingBoxSubscription(st *store.State, s *store.Store, u *store.User) ([]byte, error) {
	profiles, err := Profiles(st, s, u)
	if err != nil {
		return nil, err
	}
	outbounds, endpoints := []any{}, []any{}
	for _, p := range profiles {
		out, isEndpoint, extras, err := singbox.Outbound(p, p.Name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Name, err)
		}
		if isEndpoint {
			endpoints = append(endpoints, out)
		} else {
			outbounds = append(outbounds, out)
		}
		for _, x := range extras {
			outbounds = append(outbounds, x)
		}
	}
	return json.MarshalIndent(map[string]any{"outbounds": outbounds, "endpoints": endpoints}, "", "  ")
}

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
		if len(parts) < 2 || len(parts) > 3 || parts[0] != st.Server.Subscription.Path {
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
		// /<path>/<token>/openvpn-tcp.ovpn: a single OpenVPN config file.
		if len(parts) == 3 {
			proto, ok := strings.CutSuffix(parts[2], ".ovpn")
			if !ok {
				http.NotFound(w, r)
				return
			}
			text, err := OVPN(st, user, proto)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/x-openvpn-profile")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", st.Server.Name+"-"+proto+".ovpn"))
			w.Header().Set("Cache-Control", "no-store")
			io.WriteString(w, text)
			return
		}
		info := ""
		var up, down int64
		if snap, err := stats.Read(s.Dir); err == nil && snap.Users[user.Name] != nil {
			up, down = snap.Users[user.Name].MonthUp, snap.Users[user.Name].MonthDown
		}
		if user.QuotaGB > 0 {
			info += fmt.Sprintf("total=%d; ", int64(user.QuotaGB*(1<<30)))
		}
		if user.ExpiresAt != nil {
			info += fmt.Sprintf("expire=%d; ", user.ExpiresAt.Unix())
		}
		w.Header().Set("Subscription-Userinfo", fmt.Sprintf("upload=%d; download=%d", up, down)+strings.TrimSuffix("; "+info, "; "))
		w.Header().Set("Profile-Update-Interval", "12")
		w.Header().Set("Profile-Title", "base64:"+base64.StdEncoding.EncodeToString([]byte(st.Server.Name)))
		w.Header().Set("Cache-Control", "no-store")
		// Omnitun (and other sing-box clients) get a sing-box config, which can
		// also carry OpenVPN; everyone else gets the usual base64 link list.
		if wantsSingBox(r) {
			body, err := SingBoxSubscription(st, s, user)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
			return
		}
		var lines []string
		for _, l := range Links(st, user) {
			lines = append(lines, l.URL)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n"))))
	})
}
