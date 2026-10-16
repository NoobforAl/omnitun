package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/NoobforAl/omnitun/server/logs"
	"github.com/NoobforAl/omnitun/server/manager"
	"github.com/NoobforAl/omnitun/server/store"
)

// send feeds key presses to the model and returns the last command.
func send(m *model, ks ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range ks {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "right":
			msg = tea.KeyMsg{Type: tea.KeyRight}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "shift+tab":
			msg = tea.KeyMsg{Type: tea.KeyShiftTab}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		_, cmd = m.Update(msg)
	}
	return cmd
}

// tempDir is t.TempDir with a cleanup that retries: on Windows a virus
// scanner can briefly hold a just-written file, failing t.TempDir's cleanup.
func tempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "omnitun-tui-")
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

func newTestModel(t *testing.T) (*model, *manager.Manager) {
	t.Helper()
	mgr := manager.New(store.Open(tempDir(t)))
	if _, err := mgr.Init(manager.InitOptions{PublicHost: "203.0.113.7", Name: "Lab"}); err != nil {
		t.Fatal(err)
	}
	m, err := newModel(mgr)
	if err != nil {
		t.Fatal(err)
	}
	return m, mgr
}

func TestToggleExpiryDeleteUser(t *testing.T) {
	m, mgr := newTestModel(t)
	if _, err := mgr.AddUser(manager.NewUser{Name: "bob"}); err != nil {
		t.Fatal(err)
	}
	m.reload()
	if !strings.Contains(m.View(), "bob") {
		t.Fatal("table does not show bob")
	}

	send(m, "x", "3", "0", "enter")
	if u, _ := mgr.User("bob"); u.ExpiresAt == nil {
		t.Fatalf("expiry not set (tui error: %v)", m.err)
	}

	send(m, "e") // disable
	if u, _ := mgr.User("bob"); u.Enabled {
		t.Fatalf("bob still enabled (tui error: %v)", m.err)
	}

	send(m, "d", "n") // cancelled
	if _, err := mgr.User("bob"); err != nil {
		t.Fatal("deleted without confirmation")
	}
	send(m, "d", "y")
	if _, err := mgr.User("bob"); err == nil {
		t.Fatalf("bob not deleted (tui error: %v)", m.err)
	}
}

func TestAddUserForm(t *testing.T) {
	m, mgr := newTestModel(t)
	send(m, "a")
	if m.form == nil || !strings.Contains(m.View(), "New user") {
		t.Fatal("the add-user form did not open")
	}
	// Esc closes it without creating anything.
	send(m, "esc")
	if m.form != nil {
		t.Fatal("esc did not close the form")
	}
	if users, _ := mgr.Users(); len(users) != 0 {
		t.Fatalf("users = %v", users)
	}

	// Completing the form adds the user with the typed values.
	m.newUser = manager.NewUser{Name: "carol", Note: "laptop"}
	m.days, m.quota = "30", "5"
	m.form = nil
	m.newUser.Days, m.newUser.QuotaGB = 30, 5
	m.do("added carol", func() error { _, err := m.mgr.AddUser(m.newUser); return err })
	u, err := mgr.User("carol")
	if err != nil || u.ExpiresAt == nil || u.QuotaGB != 5 || u.Note != "laptop" {
		t.Fatalf("carol = %+v err = %v", u, err)
	}
}

func TestUserViewShowsQRAndLinks(t *testing.T) {
	m, mgr := newTestModel(t)
	if _, err := mgr.AddUser(manager.NewUser{Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	m.reload()
	send(m, "enter")
	view := m.View()
	if !strings.Contains(view, "subscription") || !strings.Contains(view, "http://203.0.113.7:2096/") {
		t.Fatalf("subscription not shown:\n%s", view)
	}
	if !strings.Contains(view, "█") {
		t.Fatal("no QR code rendered")
	}
	send(m, "right")
	if !strings.Contains(m.View(), "vless://") {
		t.Fatal("second target is not the VLESS link")
	}
}

func TestProtocolToggleAndPort(t *testing.T) {
	m, mgr := newTestModel(t)
	send(m, "2", " ") // protocols tab, toggle the first
	protos, _ := mgr.Protocols()
	if protos[0].Enabled {
		t.Fatal("vless-reality not disabled")
	}
	send(m, "down", "enter")
	m.input.SetValue("9999")
	send(m, "enter")
	protos, _ = mgr.Protocols()
	if protos[1].Port != 9999 {
		t.Fatalf("port = %d (tui error: %v)", protos[1].Port, m.err)
	}
}

func TestServerTabAPISettings(t *testing.T) {
	m, mgr := newTestModel(t)
	send(m, "3")
	view := m.View()
	if !strings.Contains(view, "Web API") || !strings.Contains(view, "•••") {
		t.Fatalf("server tab:\n%s", view)
	}
	a, _ := mgr.API()
	if strings.Contains(view, a.Token) {
		t.Fatal("token shown before asking for it")
	}
	send(m, "t")
	if !strings.Contains(m.View(), a.Token) {
		t.Fatal("token not shown after t")
	}
	send(m, "e")
	if a, _ = mgr.API(); !a.Enabled {
		t.Fatalf("API not enabled (tui error: %v)", m.err)
	}
	old := a.Token
	send(m, "n", "y")
	if a, _ = mgr.API(); a.Token == old {
		t.Fatal("token not renewed")
	}
	send(m, "p")
	m.input.SetValue("9100")
	send(m, "enter")
	if a, _ = mgr.API(); a.Port != 9100 {
		t.Fatalf("port = %d", a.Port)
	}
}

func TestTabsCycle(t *testing.T) {
	m, _ := newTestModel(t)
	for i, want := range []view{viewProtocols, viewServer, viewLogs, viewUsers} {
		send(m, "tab")
		if m.view != want {
			t.Fatalf("after %d tab(s) the view is %v, want %v", i+1, m.view, want)
		}
	}
	send(m, "shift+tab")
	if m.view != viewLogs {
		t.Fatalf("shift+tab from the first tab went to %v", m.view)
	}
}

func TestLogsTab(t *testing.T) {
	m, mgr := newTestModel(t)
	lg := logs.Open(mgr.Store.Dir)
	lg.Write("warn", "disk is almost full")
	lg.Write("error", "something failed")
	lg.Close()
	send(m, "4")
	view := m.View()
	if m.view != viewLogs || !strings.Contains(view, "disk is almost full") || !strings.Contains(view, "something failed") {
		t.Fatalf("logs tab:\n%s", view)
	}
	send(m, "tab")
	if m.view != viewUsers {
		t.Fatalf("tab from the last tab went to %v", m.view)
	}
}
