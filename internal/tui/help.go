package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpRow struct {
	Key   string
	Label string
}

func helpKeys() []helpRow {
	return []helpRow{
		{"←→", "nav / pane"},
		{"↑↓←→ jk", "browse"},
		{"tab", "nav / list"},
		{"⏎", "add to queue"},
		{"l", "like"},
		{"space", "play / pause"},
		{", .", "previous / next"},
		{"/", "find"},
		{"?", "this screen"},
		{"q", "quit"},
	}
}

func (m model) renderHelp(height int) string {
	return fieldset("Help", m.renderHelpBody(), m.mainWidth(), height, m.focus == focusList)
}

func (m model) renderHelpBody() string {
	var b strings.Builder
	b.WriteString(styleMuted().Render("keys"))
	b.WriteByte('\n')
	keyW := 8
	for _, row := range helpKeys() {
		if w := lipgloss.Width(row.Key); w > keyW {
			keyW = w
		}
	}
	for _, row := range helpKeys() {
		pad := keyW - lipgloss.Width(row.Key)
		b.WriteString("  ")
		b.WriteString(styleSelected().Render(row.Key))
		b.WriteString(strings.Repeat(" ", pad+2))
		b.WriteString(styleMuted().Render(row.Label))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
