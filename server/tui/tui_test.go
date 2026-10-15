package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/NoobforAl/omnitun/server/store"
)

func keys(m *model, ks ...string) {
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
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		m.Update(msg)
	}
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

func newTestModel(t *testing.T) (*model, *store.Store) {
	t.Helper()
	s := store.Open(tempDir(t))
	st, err := store.Init(s, store.InitOptions{PublicHost: "203.0.113.7", Name: "Lab"})
	if err != nil {
		t.Fatal(err)
	}
	return newModel(s, st), s
}

func TestAddToggleDeleteUser(t *testing.T) {
	m, s := newTestModel(t)
	keys(m, "a", "b", "o", "b", "enter", "3", "0", "enter")
	st, _ := s.Load()
	bob := st.User("bob")
	if bob == nil || bob.ExpiresAt == nil {
		t.Fatalf("bob not added with expiry: %+v (tui error: %v)", bob, m.err)
	}
	if !strings.Contains(m.View(), "bob") {
		t.Fatal("table does not show bob")
	}

	keys(m, "e") // disable
	st, _ = s.Load()
	if st.User("bob").Enabled {
		t.Fatalf("bob still enabled (tui error: %v)", m.err)
	}

	keys(m, "d", "n") // cancelled
	st, _ = s.Load()
	if st.User("bob") == nil {
		t.Fatal("deleted without confirmation")
	}
	keys(m, "d", "y")
	st, _ = s.Load()
	if st.User("bob") != nil {
		t.Fatalf("bob not deleted (tui error: %v)", m.err)
	}
}

func TestUserViewShowsQRAndLinks(t *testing.T) {
	m, s := newTestModel(t)
	if err := s.Update(func(st *store.State) error { _, err := store.AddUser(st, "alice"); return err }); err != nil {
		t.Fatal(err)
	}
	m.loaded = m.loaded.Add(-1) // force reload
	m.reload()
	keys(m, "enter")
	view := m.View()
	if !strings.Contains(view, "subscription") || !strings.Contains(view, "http://203.0.113.7:2096/") {
		t.Fatalf("subscription not shown:\n%s", view)
	}
	if !strings.Contains(view, "█") {
		t.Fatal("no QR code rendered")
	}
	keys(m, "right")
	if !strings.Contains(m.View(), "vless://") {
		t.Fatal("second target is not the VLESS link")
	}
}

func TestProtocolToggleAndPort(t *testing.T) {
	m, s := newTestModel(t)
	keys(m, "p", " ")
	st, _ := s.Load()
	if st.Server.Protocols[store.VLESSReality].Enabled {
		t.Fatal("vless-reality not disabled")
	}
	keys(m, "down", "enter")
	m.input.SetValue("9999")
	keys(m, "enter")
	st, _ = s.Load()
	if st.Server.Protocols[store.Shadowsocks].Port != 9999 {
		t.Fatalf("port = %d", st.Server.Protocols[store.Shadowsocks].Port)
	}
}
