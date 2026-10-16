package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

// loadLogs reads as many log lines as fit on the screen.
func (m *model) loadLogs() {
	n := 15
	if m.height > 12 {
		n = m.height - 10
	}
	if lines, err := m.mgr.Logs(n); err == nil {
		m.logs = lines
	}
}

func (m *model) updateLogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "r":
		m.loadLogs()
	}
	return m, nil
}

func (m *model) logsView() string {
	var b strings.Builder
	b.WriteString("Server log (refreshes every 2 s)\n\n")
	if len(m.logs) == 0 {
		b.WriteString(dimStyle.Render("Nothing logged yet. The log is written while `omnitun-server run` is running."))
	}
	width := m.width - 2
	for _, l := range m.logs {
		if r := []rune(l); width > 20 && len(r) > width {
			l = string(r[:width-1]) + "…"
		}
		style := dimStyle
		switch {
		case strings.Contains(l, " ERROR "):
			style = errStyle
		case strings.Contains(l, " WARN "):
			style = warnStyle
		}
		b.WriteString(style.Render(l) + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("r refresh · tab next · q quit"))
	return b.String()
}
