package subscription

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var links = strings.Join([]string{
	"trojan://pw@a.example.com:443#A",
	"ss://aes-128-gcm:pw@b.example.com:8388#B",
}, "\n")

func newServer(t *testing.T, check func(r *http.Request) bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !check(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Subscription-Userinfo", "upload=100; download=200; total=1000; expire=1893456000")
		w.Header().Set("Profile-Update-Interval", "12")
		w.Header().Set("Profile-Title", "base64:"+base64.StdEncoding.EncodeToString([]byte("My Provider")))
		w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(links))))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchAuthTypes(t *testing.T) {
	cases := []struct {
		name  string
		auth  Auth
		check func(r *http.Request) bool
	}{
		{"none", Auth{Type: AuthNone}, func(*http.Request) bool { return true }},
		{"basic", Auth{Type: AuthBasic, Username: "u", Password: "p"}, func(r *http.Request) bool {
			u, p, ok := r.BasicAuth()
			return ok && u == "u" && p == "p"
		}},
		{"bearer", Auth{Type: AuthBearer, Token: "t0k"}, func(r *http.Request) bool {
			return r.Header.Get("Authorization") == "Bearer t0k"
		}},
		{"header", Auth{Type: AuthHeader, Header: "X-Api-Key", Token: "k"}, func(r *http.Request) bool {
			return r.Header.Get("X-Api-Key") == "k"
		}},
		{"query", Auth{Type: AuthQuery, QueryKey: "key", Token: "q1"}, func(r *http.Request) bool {
			return r.URL.Query().Get("key") == "q1"
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := newServer(t, c.check)
			pool := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
			res, err := Fetch(t.Context(), Request{URL: srv.URL + "/sub?id=1", Auth: c.auth}, pool)
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Profiles) != 2 || res.Profiles[0].Name != "A" {
				t.Fatalf("profiles: %+v", res.Profiles)
			}
			want := Info{Upload: 100, Download: 200, Total: 1000, Expire: 1893456000, UpdateIntervalHours: 12, Title: "My Provider"}
			if res.Info != want {
				t.Fatalf("info = %+v, want %+v", res.Info, want)
			}
		})
	}
}

func TestFetchWrongCredentials(t *testing.T) {
	srv := newServer(t, func(r *http.Request) bool { return r.Header.Get("Authorization") == "Bearer right" })
	pool := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	_, err := Fetch(t.Context(), Request{URL: srv.URL, Auth: Auth{Type: AuthBearer, Token: "wrong"}}, pool)
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("err = %v, want ErrAuth", err)
	}
}

func TestFetchRejectsBadInput(t *testing.T) {
	for _, u := range []string{"", "ftp://x", "not a url", "https://"} {
		if _, err := Fetch(t.Context(), Request{URL: u}, nil); err == nil {
			t.Errorf("%q: expected error", u)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("proxies:\n  - name: x\n"))
	}))
	defer srv.Close()
	// A Clash config whose only proxy has no usable type yields an error, not an empty group.
	if _, err := Fetch(t.Context(), Request{URL: srv.URL}, nil); err == nil || !strings.Contains(err.Error(), "unsupported proxy type") {
		t.Fatalf("err = %v, want unsupported proxy type", err)
	}
}

// FetchVia opens its connections with the dialer it is given (the running VPN).
func TestFetchViaUsesTheDialer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "provider.invalid" {
			http.Error(w, "wrong host "+r.Host, http.StatusBadRequest)
			return
		}
		w.Write([]byte("trojan://pw@a.example.com:443#A"))
	}))
	defer srv.Close()
	var dialed []string
	dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialed = append(dialed, addr)
		var d net.Dialer
		return d.DialContext(ctx, network, srv.Listener.Addr().String()) // the name only exists on the far side
	}
	res, err := FetchVia(t.Context(), Request{URL: "http://provider.invalid/sub"}, nil, dial)
	if err != nil || len(res.Profiles) != 1 {
		t.Fatalf("%v %+v", err, res)
	}
	if len(dialed) != 1 || dialed[0] != "provider.invalid:80" {
		t.Fatalf("dialed %v", dialed)
	}
}
