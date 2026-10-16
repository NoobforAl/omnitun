//go:build with_acme && with_quic && with_utls && with_wireguard && with_gvisor

package runtime

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"

	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/store"
)

// The generated ACME configuration must be accepted by sing-box (provider
// registered, inbounds referencing it).
func TestACMEConfigIsAcceptedBySingBox(t *testing.T) {
	s, _ := setup(t)
	m := manager.New(s)
	domain := "vpn.example.com"
	if _, err := m.EnableACME(manager.ACMEPatch{Domain: &domain}); err != nil {
		t.Fatal(err)
	}
	st, _ := s.Load()
	cfg := serverconf.Build(st, s, time.Now())
	raw, _ := json.Marshal(cfg)
	ctx := include.Context(context.Background())
	opts, err := sjson.UnmarshalExtendedContext[option.Options](ctx, raw)
	if err != nil {
		t.Fatalf("sing-box rejects the ACME config: %v\n%s", err, raw)
	}
	if len(opts.CertificateProviders) != 1 {
		t.Fatalf("providers = %d", len(opts.CertificateProviders))
	}
	// Creating the instance wires inbounds to the provider (it does not
	// contact the CA until started).
	instance, err := box.New(box.Options{Context: ctx, Options: opts})
	if err != nil {
		t.Fatalf("box.New: %v", err)
	}
	instance.Close()
}

// The subscription endpoint serves HTTPS under ACME, with the self-signed
// certificate until the first real one exists.
func TestSubscriptionOverHTTPSWithACME(t *testing.T) {
	s := store.Open(tempDir(t))
	m := manager.New(s)
	if _, err := m.Init(manager.InitOptions{PublicHost: "127.0.0.1", SubPort: freeTCP(t)}); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(st *store.State) error { st.Server.Subscription.Listen = "127.0.0.1"; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddUser(manager.NewUser{Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	off := false
	m.UpdateUser("alice", manager.UserPatch{Enabled: &off}) // no active users: sing-box (and the CA) stay out of this test
	domain := "vpn.example.com"
	if _, err := m.EnableACME(manager.ACMEPatch{Domain: &domain}); err != nil {
		t.Fatal(err)
	}

	srv := New(s, t.Logf)
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	st, _ := s.Load()
	port := strconv.Itoa(st.Server.Subscription.Port)

	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // the fallback certificate is self-signed
	}}
	u := st.Users[0]
	resp, err := client.Get("https://127.0.0.1:" + port + "/" + st.Server.Subscription.Path + "/" + u.SubToken)
	if err != nil {
		t.Fatalf("HTTPS request: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || resp.TLS == nil {
		t.Fatalf("status %d, TLS %v (a disabled account answers 403 over TLS)", resp.StatusCode, resp.TLS != nil)
	}
	// Plain HTTP no longer works on that port.
	if r, err := (&http.Client{Timeout: 3 * time.Second}).Get("http://127.0.0.1:" + port + "/"); err == nil && r.StatusCode == 200 {
		t.Fatal("the endpoint still answers plain HTTP")
	}
}
