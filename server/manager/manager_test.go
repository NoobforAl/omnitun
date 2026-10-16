package manager

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/server/stats"
	"github.com/NoobforAl/omnitun/server/store"
)

// tempDir is t.TempDir with a cleanup that retries: on Windows a virus
// scanner or a just-released file lock can briefly keep a file open, which
// fails t.TempDir's own cleanup.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "omnitun-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 40 && os.RemoveAll(dir) != nil; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	})
	return dir
}

func newManager(t *testing.T) *Manager {
	t.Helper()
	m := New(store.Open(tempDir(t)))
	if _, err := m.Init(InitOptions{PublicHost: "203.0.113.7", Name: "Test"}); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestInitTwiceAndWithoutHost(t *testing.T) {
	m := New(store.Open(tempDir(t)))
	if _, err := m.Init(InitOptions{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("no host: %v", err)
	}
	if _, err := m.Init(InitOptions{PublicHost: "example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Init(InitOptions{PublicHost: "example.com"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("second init: %v", err)
	}
	if _, err := New(store.Open(tempDir(t))).Server(); !errors.Is(err, store.ErrNotInitialized) {
		t.Fatalf("uninitialised: %v", err)
	}
}

func TestProtocols(t *testing.T) {
	m := newManager(t)
	list, err := m.Protocols()
	if err != nil || len(list) != len(store.AllProtocols) {
		t.Fatalf("list = %v err = %v", list, err)
	}
	off, port := false, 9443
	p, err := m.SetProtocol(store.Trojan, ProtocolPatch{Enabled: &off, Port: &port})
	if err != nil || p.Enabled || p.Port != 9443 || p.Transport != "tcp" {
		t.Fatalf("patched = %+v err = %v", p, err)
	}
	if _, err := m.SetProtocol("nope", ProtocolPatch{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown protocol: %v", err)
	}
	bad := 70000
	if _, err := m.SetProtocol(store.Trojan, ProtocolPatch{Port: &bad}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad port: %v", err)
	}
	if info, _ := m.Server(); info.Protocols != len(store.AllProtocols)-1 {
		t.Fatalf("enabled protocols = %d", info.Protocols)
	}
}

func TestUserLifecycle(t *testing.T) {
	m := newManager(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	m.Now = func() time.Time { return now }

	u, err := m.AddUser(NewUser{Name: "alice", Days: 30, QuotaGB: 50, Note: "phone"})
	if err != nil || u.Status != "active" || u.ExpiresAt == nil || !u.ExpiresAt.Equal(now.AddDate(0, 0, 30)) {
		t.Fatalf("added = %+v err = %v", u, err)
	}
	if _, err := m.AddUser(NewUser{Name: "alice"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := m.AddUser(NewUser{Name: "bad name!"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid name: %v", err)
	}

	off, never, quota := false, 0, 0.0
	u, err = m.UpdateUser("alice", UserPatch{Enabled: &off, ExpireDays: &never, QuotaGB: &quota})
	if err != nil || u.Status != "disabled" || u.ExpiresAt != nil || u.QuotaGB != 0 {
		t.Fatalf("updated = %+v err = %v", u, err)
	}
	if _, err := m.UpdateUser("ghost", UserPatch{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}

	before, _ := m.Links("alice")
	if _, err := m.ResetUser("alice"); err != nil {
		t.Fatal(err)
	}
	after, _ := m.Links("alice")
	if before.Subscription == after.Subscription || before.Links[0].URL == after.Links[0].URL {
		t.Fatal("reset did not change the links")
	}

	if err := m.DeleteUser("alice"); err != nil {
		t.Fatal(err)
	}
	if err := m.DeleteUser("alice"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
	if users, _ := m.Users(); len(users) != 0 {
		t.Fatalf("users left: %v", users)
	}
}

func TestLinksAndOVPN(t *testing.T) {
	m := newManager(t)
	if _, err := m.AddUser(NewUser{Name: "bob"}); err != nil {
		t.Fatal(err)
	}
	set, err := m.Links("bob")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(set.Subscription, "http://203.0.113.7:") || len(set.Links) == 0 {
		t.Fatalf("links = %+v", set)
	}
	if set.OVPN[store.OpenVPNUDP] == "" || set.OVPN[store.OpenVPNTCP] == "" {
		t.Fatalf("ovpn = %v", set.OVPN)
	}
	text, err := m.OVPN("bob", "udp")
	if err != nil || !strings.Contains(text, "remote 203.0.113.7") {
		t.Fatalf("ovpn text err = %v\n%s", err, text)
	}
	if _, err := m.OVPN("bob", "icmp"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad ovpn proto: %v", err)
	}
	if _, err := m.Links("ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}
}

func TestAPISettings(t *testing.T) {
	m := newManager(t)
	a, err := m.API()
	if err != nil || a.Enabled || a.Listen != "127.0.0.1" || len(a.Token) != 48 {
		t.Fatalf("defaults = %+v err = %v", a, err)
	}
	on, port := true, 9000
	b, err := m.SetAPI(APIPatch{Enabled: &on, Port: &port, NewToken: true})
	if err != nil || !b.Enabled || b.Port != 9000 || b.Token == a.Token {
		t.Fatalf("updated = %+v err = %v", b, err)
	}
	bad := 0
	if _, err := m.SetAPI(APIPatch{Port: &bad}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad port: %v", err)
	}
}

func TestUsageAndQuotaFlag(t *testing.T) {
	m := newManager(t)
	if _, err := m.AddUser(NewUser{Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	// The running server would write these numbers.
	tr := stats.NewTracker(m.Store.Dir)
	tr.Add("alice", 2<<20, 8<<20)
	tr.SetLive(map[string]stats.Live{"alice": {Connections: 3, UpBps: 5, DownBps: 9}})
	if err := tr.Save(); err != nil {
		t.Fatal(err)
	}

	u, err := m.User("alice")
	if err != nil || u.Usage.MonthBytes != 10<<20 || u.Usage.TotalBytes != 10<<20 || u.Usage.Connections != 3 {
		t.Fatalf("usage = %+v err = %v", u.Usage, err)
	}
	if info, _ := m.Server(); !info.Running || info.Connections != 3 {
		t.Fatalf("server = %+v", info)
	}

	// A quota below the month's usage blocks at once, raising it frees the user.
	low := 5.0 / 1024 // 5 MB
	u, _ = m.UpdateUser("alice", UserPatch{QuotaGB: &low})
	if u.Status != "over-quota" {
		t.Fatalf("status = %s", u.Status)
	}
	high := 1.0
	if u, _ = m.UpdateUser("alice", UserPatch{QuotaGB: &high}); u.Status != "active" {
		t.Fatalf("status after raising = %s", u.Status)
	}

	// Reset asks the server to restart the month and unblocks.
	m.UpdateUser("alice", UserPatch{QuotaGB: &low})
	u, err = m.ResetUsage("alice")
	if err != nil || u.Status != "active" || u.Usage.MonthBytes != 0 || u.Usage.TotalBytes != 10<<20 {
		t.Fatalf("after reset = %+v err = %v", u, err)
	}
	st, _ := m.Store.Load()
	if st.User("alice").UsageResetAt == nil {
		t.Fatal("no reset request stored for the server")
	}
	if _, err := m.ResetUsage("ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}

	// Once the server stops, live numbers disappear.
	tr.Stop()
	if u, _ = m.User("alice"); u.Usage.Connections != 0 {
		t.Fatalf("connections after stop = %d", u.Usage.Connections)
	}
	if info, _ := m.Server(); info.Running {
		t.Fatal("server still reported as running")
	}
}
