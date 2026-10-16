//go:build with_quic && with_utls && with_wireguard && with_gvisor

package runtime

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/stats"
	"github.com/NoobforAl/omnitun/server/store"
)

func waitFor(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Traffic through a real client connection is counted per user, written to
// the stats file, and a user over the monthly quota is blocked until the
// quota is raised or the usage is reset.
func TestUserAccountingAndQuota(t *testing.T) {
	oldInterval, oldEvery := accountingInterval, enforceEvery
	accountingInterval, enforceEvery = 100*time.Millisecond, 2
	defer func() { accountingInterval, enforceEvery = oldInterval, oldEvery }()

	s, srv := setup(t)
	st, _ := s.Load()
	// `run` watches the state file and reloads; do the same here.
	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	go srv.Watch(watchCtx)

	const payload = 300 << 10
	web := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, strings.Repeat("x", payload))
	}))
	web.Listener.Close()
	var err error
	if web.Listener, err = net.Listen("tcp", net.JoinHostPort(lanIPv4(t), "0")); err != nil {
		t.Fatal(err)
	}
	web.Start()
	defer web.Close()

	// Connect with the user's Trojan link.
	var trojan string
	for _, l := range serverconf.Links(st, st.User("alice")) {
		if l.Protocol == store.Trojan {
			trojan = l.URL
		}
	}
	res, err := importer.Import(trojan)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Profiles[0]
	proxyPort := freeTCP(t)
	e := engine.New(engine.Config{DataDir: t.TempDir()})
	if err := e.Start(p.AsSlice(), singbox.Options{
		ListenPort: proxyPort, LogLevel: "error", ProxyPrivate: true, Selected: p.ID, TestURL: web.URL + "/probe",
	}); err != nil {
		t.Fatal(err)
	}
	defer e.Stop()
	waitFor(t, "client to connect", 15*time.Second, func() bool { return e.State().Status == engine.StatusRunning })

	proxy, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(proxyPort))
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true}, Timeout: 10 * time.Second}
	get := func() error {
		resp, err := client.Get(web.URL + "/data")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, err = io.Copy(io.Discard, resp.Body)
		return err
	}
	if err := get(); err != nil {
		t.Fatal(err)
	}

	// The download is counted against alice (down is traffic to the client).
	waitFor(t, "the traffic to be counted", 10*time.Second, func() bool { return srv.tracker.MonthTotal("alice") >= payload })
	if err := srv.tracker.Save(); err != nil {
		t.Fatal(err)
	}
	snap, err := stats.Read(s.Dir)
	if err != nil || snap.Users["alice"] == nil || snap.Users["alice"].Down < payload || !snap.Running(time.Now()) {
		t.Fatalf("stats file: %+v err %v", snap.Users["alice"], err)
	}

	// Quota below what she already used: blocked within a few ticks.
	if err := s.Update(func(st *store.State) error { st.User("alice").QuotaGB = 100 << 10 / float64(stats.GiB); return nil }); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "quota block", 10*time.Second, func() bool {
		st, _ := s.Load()
		return st.User("alice").QuotaExceeded
	})
	if st, _ := s.Load(); st.User("alice").Status(time.Now()) != "over-quota" {
		t.Fatalf("status = %s", st.User("alice").Status(time.Now()))
	}
	waitFor(t, "the server to drop the blocked user", 10*time.Second, func() bool {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		return srv.instance == nil
	})

	// Raising the quota lets her back in.
	if err := s.Update(func(st *store.State) error { st.User("alice").QuotaGB = 10; return nil }); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "quota unblock", 10*time.Second, func() bool {
		st, _ := s.Load()
		return !st.User("alice").QuotaExceeded
	})

	// A usage reset restarts the monthly count (lifetime totals stay).
	before := snapUser(t, s.Dir)
	now := time.Now().UTC()
	if err := s.Update(func(st *store.State) error { st.User("alice").UsageResetAt = &now; return nil }); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "usage reset", 10*time.Second, func() bool { return srv.tracker.MonthTotal("alice") == 0 })
	srv.tracker.Save()
	after := snapUser(t, s.Dir)
	if after.Down < before.Down || after.MonthTotal() != 0 {
		t.Fatalf("lifetime %d -> %d, month %d", before.Down, after.Down, after.MonthTotal())
	}
}

func snapUser(t *testing.T, dir string) stats.User {
	t.Helper()
	snap, err := stats.Read(dir)
	if err != nil || snap.Users["alice"] == nil {
		t.Fatalf("no stats for alice: %v", err)
	}
	return *snap.Users["alice"]
}
