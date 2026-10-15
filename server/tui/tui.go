// Package tui is the interactive manager for omnitun-server: users, their
// links and QR codes, and protocols. It edits the same state file as the CLI;
// a running server picks changes up within seconds.
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/NoobforAl/omnitun/server/qrterm"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/store"
)

type view int

const (
	viewUsers view = iota
	viewUser
	viewProtocols
)

// prompt is a pending single-line input.
type prompt struct {
	label  string
	submit func(m *model, value string) error
}

type tickMsg time.Time

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
)

type model struct {
	s      *store.Store
	st     *store.State
	loaded time.Time

	view    view
	users   table.Model
	user    string // user shown in viewUser
	qrIndex int    // 0 = subscription, then each link
	proto   int    // cursor in viewProtocols

	input   textinput.Model
	prompt  *prompt
	confirm *prompt // yes/no question; submit on "y"

	status string
	err    error
	width  int
	height int
}

// Run starts the TUI.
func Run(s *store.Store) error {
	st, err := s.Load()
	if err != nil {
		return err
	}
	m := newModel(s, st)
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func newModel(s *store.Store, st *store.State) *model {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Name", Width: 18},
			{Title: "Status", Width: 9},
			{Title: "Expires", Width: 11},
			{Title: "Quota", Width: 10},
			{Title: "Note", Width: 20},
		}),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	in := textinput.New()
	in.CharLimit = 64
	m := &model{s: s, st: st, loaded: s.ModTime(), users: t, input: in}
	m.refreshRows()
	return m
}

func (m *model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// reload picks up changes made by the CLI or another TUI.
func (m *model) reload() {
	if mt := m.s.ModTime(); !mt.Equal(m.loaded) {
		if st, err := m.s.Load(); err == nil {
			m.st, m.loaded = st, mt
			m.refreshRows()
		}
	}
}

// update applies f to the state file and reloads.
func (m *model) update(f func(*store.State) error, done string) {
	if err := m.s.Update(f); err != nil {
		m.err, m.status = err, ""
		return
	}
	m.err, m.status = nil, done
	m.loaded = time.Time{}
	m.reload()
}

func (m *model) refreshRows() {
	now := time.Now()
	rows := make([]table.Row, 0, len(m.st.Users))
	for _, u := range m.st.Users {
		exp := "never"
		if u.ExpiresAt != nil {
			exp = u.ExpiresAt.Local().Format("2006-01-02")
		}
		quota := "unlimited"
		if u.QuotaGB > 0 {
			quota = strconv.FormatFloat(u.QuotaGB, 'f', -1, 64) + " GB"
		}
		rows = append(rows, table.Row{u.Name, u.Status(now), exp, quota, u.Note})
	}
	m.users.SetRows(rows)
	// An empty table leaves the cursor at -1; select the first row once there is one.
	if m.users.Cursor() < 0 && len(rows) > 0 {
		m.users.SetCursor(0)
	}
}

func (m *model) selectedUser() *store.User {
	row := m.users.SelectedRow()
	if row == nil {
		return nil
	}
	return m.st.User(row[0])
}

func (m *model) ask(label, initial string, submit func(*model, string) error) {
	m.prompt = &prompt{label: label, submit: submit}
	m.input.SetValue(initial)
	m.input.CursorEnd()
	m.input.Focus()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.users.SetHeight(max(5, msg.Height-12))
		return m, nil
	case tickMsg:
		m.reload()
		return m, tick()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.prompt != nil {
			return m.updatePrompt(msg)
		}
		if m.confirm != nil {
			if msg.String() == "y" {
				if err := m.confirm.submit(m, ""); err != nil {
					m.err = err
				}
			}
			m.confirm = nil
			return m, nil
		}
		switch m.view {
		case viewUsers:
			return m.updateUsers(msg)
		case viewUser:
			return m.updateUser(msg)
		case viewProtocols:
			return m.updateProtocols(msg)
		}
	}
	return m, nil
}

func (m *model) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.prompt = nil
		return m, nil
	case "enter":
		p := m.prompt
		m.prompt = nil
		if err := p.submit(m, strings.TrimSpace(m.input.Value())); err != nil {
			m.err = err
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) updateUsers(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	u := m.selectedUser()
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "a":
		m.ask("New user name", "", func(m *model, name string) error {
			m.update(func(st *store.State) error { _, err := store.AddUser(st, name); return err }, "added "+name)
			if m.err == nil {
				m.ask("Days until it expires (empty = never)", "", func(m *model, v string) error {
					return m.setExpiry(name, v)
				})
			}
			return nil
		})
	case "enter":
		if u != nil {
			m.view, m.user, m.qrIndex = viewUser, u.Name, 0
		}
	case "e", " ":
		if u != nil {
			name, enable := u.Name, !u.Enabled
			m.update(func(st *store.State) error { st.User(name).Enabled = enable; return nil },
				fmt.Sprintf("%s %s", name, map[bool]string{true: "enabled", false: "disabled"}[enable]))
		}
	case "x":
		if u != nil {
			name := u.Name
			m.ask("Days until "+name+" expires (empty/never = no expiry)", "", func(m *model, v string) error {
				return m.setExpiry(name, v)
			})
		}
	case "g":
		if u != nil {
			name := u.Name
			m.ask("Data cap for "+name+" in GB (0 = unlimited)", strconv.FormatFloat(u.QuotaGB, 'f', -1, 64), func(m *model, v string) error {
				gb, err := strconv.ParseFloat(v, 64)
				if err != nil || gb < 0 {
					return fmt.Errorf("invalid quota %q", v)
				}
				m.update(func(st *store.State) error { st.User(name).QuotaGB = gb; return nil }, "quota updated")
				return nil
			})
		}
	case "n":
		if u != nil {
			name := u.Name
			m.ask("Note for "+name, u.Note, func(m *model, v string) error {
				m.update(func(st *store.State) error { st.User(name).Note = v; return nil }, "note updated")
				return nil
			})
		}
	case "d":
		if u != nil {
			name := u.Name
			m.confirm = &prompt{label: "Delete " + name + "? (y/N)", submit: func(m *model, _ string) error {
				m.update(func(st *store.State) error { return store.DeleteUser(st, name) }, "deleted "+name)
				return nil
			}}
		}
	case "r":
		if u != nil {
			m.confirmReset(u.Name)
		}
	case "p":
		m.view, m.proto = viewProtocols, 0
	default:
		var cmd tea.Cmd
		m.users, cmd = m.users.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) confirmReset(name string) {
	m.confirm = &prompt{label: "Issue new credentials for " + name + "? Old links stop working. (y/N)", submit: func(m *model, _ string) error {
		m.update(func(st *store.State) error { store.ResetCredentials(st.User(name)); return nil }, "new credentials for "+name)
		return nil
	}}
}

func (m *model) setExpiry(name, v string) error {
	if v == "" || v == "never" {
		m.update(func(st *store.State) error { st.User(name).ExpiresAt = nil; return nil }, name+": never expires")
		return nil
	}
	days, err := strconv.Atoi(v)
	if err != nil || days < 0 {
		return fmt.Errorf("invalid number of days %q", v)
	}
	t := time.Now().AddDate(0, 0, days).UTC()
	m.update(func(st *store.State) error { st.User(name).ExpiresAt = &t; return nil },
		fmt.Sprintf("%s expires %s", name, t.Local().Format("2006-01-02")))
	return nil
}

// qrTargets are the subscription URL and every link for the shown user.
func (m *model) qrTargets(u *store.User) []serverconf.Link {
	var out []serverconf.Link
	if sub := serverconf.SubscriptionURL(m.st, u); sub != "" {
		out = append(out, serverconf.Link{Protocol: "subscription (all protocols)", URL: sub})
	}
	return append(out, serverconf.Links(m.st, u)...)
}

func (m *model) updateUser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	u := m.st.User(m.user)
	if u == nil {
		m.view = viewUsers
		return m, nil
	}
	n := len(m.qrTargets(u))
	switch msg.String() {
	case "esc", "q", "backspace":
		m.view = viewUsers
	case "right", "l", "tab":
		if n > 0 {
			m.qrIndex = (m.qrIndex + 1) % n
		}
	case "left", "h", "shift+tab":
		if n > 0 {
			m.qrIndex = (m.qrIndex - 1 + n) % n
		}
	case "r":
		m.confirmReset(u.Name)
	}
	return m, nil
}

func (m *model) updateProtocols(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	name := store.AllProtocols[m.proto]
	switch msg.String() {
	case "esc", "q":
		m.view = viewUsers
	case "up", "k":
		m.proto = (m.proto - 1 + len(store.AllProtocols)) % len(store.AllProtocols)
	case "down", "j":
		m.proto = (m.proto + 1) % len(store.AllProtocols)
	case " ", "e":
		m.update(func(st *store.State) error {
			p := st.Server.Protocols[name]
			p.Enabled = !p.Enabled
			return nil
		}, name+" toggled")
	case "enter":
		m.ask("Port for "+name, strconv.Itoa(m.st.Server.Protocols[name].Port), func(m *model, v string) error {
			port, err := strconv.Atoi(v)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("invalid port %q", v)
			}
			m.update(func(st *store.State) error { st.Server.Protocols[name].Port = port; return nil }, name+" port updated")
			return nil
		})
	}
	return m, nil
}

func (m *model) View() string {
	var b strings.Builder
	srv := m.st.Server
	b.WriteString(titleStyle.Render("Omnitun server") + dimStyle.Render(fmt.Sprintf("  %s · %s · %s", srv.Name, srv.PublicHost, m.s.Dir)) + "\n\n")

	switch m.view {
	case viewUsers:
		active := len(m.st.ActiveUsers(time.Now()))
		b.WriteString(fmt.Sprintf("Users (%d, %d active)\n", len(m.st.Users), active))
		if len(m.st.Users) == 0 {
			b.WriteString(dimStyle.Render("No users yet — press a to add one.") + "\n")
		} else {
			b.WriteString(m.users.View() + "\n")
		}
		b.WriteString(dimStyle.Render("enter links & QR · a add · e enable/disable · x expiry · g quota · n note · r new credentials · d delete · p protocols · q quit"))
	case viewUser:
		b.WriteString(m.userView())
	case viewProtocols:
		b.WriteString(m.protocolsView())
	}

	b.WriteString("\n\n")
	switch {
	case m.prompt != nil:
		b.WriteString(m.prompt.label + ": " + m.input.View() + dimStyle.Render("  (enter to save, esc to cancel)"))
	case m.confirm != nil:
		b.WriteString(errStyle.Render(m.confirm.label))
	case m.err != nil:
		b.WriteString(errStyle.Render("Error: " + m.err.Error()))
	case m.status != "":
		b.WriteString(okStyle.Render(m.status))
	}
	return b.String()
}

func (m *model) userView() string {
	u := m.st.User(m.user)
	if u == nil {
		return "user not found"
	}
	var b strings.Builder
	exp := "never"
	if u.ExpiresAt != nil {
		exp = u.ExpiresAt.Local().Format("2006-01-02")
	}
	b.WriteString(fmt.Sprintf("%s  %s  expires %s\n\n", titleStyle.Render(u.Name), u.Status(time.Now()), exp))
	targets := m.qrTargets(u)
	if len(targets) == 0 {
		return b.String() + "No protocols enabled."
	}
	t := targets[m.qrIndex%len(targets)]
	b.WriteString(fmt.Sprintf("%s %s\n", okStyle.Render(fmt.Sprintf("[%d/%d]", m.qrIndex%len(targets)+1, len(targets))), t.Protocol))
	if code, err := qrterm.Render(t.URL); err == nil {
		b.WriteString(code)
	}
	b.WriteString(boxStyle.Width(max(40, min(m.width-4, 110))).Render(t.URL) + "\n")
	b.WriteString(dimStyle.Render("←/→ switch link · r new credentials · esc back"))
	return b.String()
}

func (m *model) protocolsView() string {
	var b strings.Builder
	b.WriteString("Protocols\n\n")
	for i, name := range store.AllProtocols {
		p := m.st.Server.Protocols[name]
		cursor := "  "
		if i == m.proto {
			cursor = okStyle.Render("▸ ")
		}
		state := errStyle.Render("off")
		if p.Enabled {
			state = okStyle.Render("on ")
		}
		b.WriteString(fmt.Sprintf("%s%s  %-14s port %d\n", cursor, state, name, p.Port))
	}
	b.WriteString("\n" + dimStyle.Render("space toggle · enter change port · esc back"))
	return b.String()
}
