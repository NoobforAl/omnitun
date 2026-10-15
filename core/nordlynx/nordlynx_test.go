package nordlynx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProfiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/users/services/credentials", func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "token" || p != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"nordlynx_private_key":"yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk="}`))
	})
	mux.HandleFunc("/v1/servers/countries", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":81,"code":"DE"},{"id":228,"code":"US"}]`))
	})
	mux.HandleFunc("/v1/servers/recommendations", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("filters[country_id]") != "81" || r.URL.Query().Get("filters[servers_technologies][identifier]") != "wireguard_udp" {
			t.Errorf("query = %v", r.URL.RawQuery)
		}
		w.Write([]byte(`[
		 {"name":"Germany #1","hostname":"de1.nordvpn.com","station":"1.2.3.4","technologies":[
		   {"identifier":"openvpn_udp"},
		   {"identifier":"wireguard_udp","metadata":[{"name":"public_key","value":"xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="}]}]},
		 {"name":"no wg","hostname":"x.nordvpn.com","station":"5.6.7.8","technologies":[{"identifier":"openvpn_udp"}]}
		]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	BaseURL = srv.URL

	ps, err := Profiles(t.Context(), srv.Client(), "tok", "de", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 {
		t.Fatalf("profiles = %d", len(ps))
	}
	p := ps[0]
	if p.Server != "1.2.3.4" || p.Port != 51820 || p.WireGuard.PeerPublicKey == "" || p.WireGuard.Address[0] != "10.5.0.2/32" {
		t.Fatalf("profile: %+v %+v", p, p.WireGuard)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := Profiles(t.Context(), srv.Client(), "bad", "", 5); err != ErrAuth {
		t.Fatalf("bad token err = %v", err)
	}
}
