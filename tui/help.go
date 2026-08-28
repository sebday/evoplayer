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
		{"←→", "menu"},
		{"↓", "into viewport"},
		{"↑", "back to menu"},
		{"jk", "browse"},
		{"tab", "header / pane"},
		{"⏎", "play"},
		{"+", "add to current"},
		{"l", "like"},
		{"v", "visualizer"},
		{"space", "play / pause"},
		{", .", "previous / next"},
		{"- =", "volume"},
		{"/", "find"},
		{"?", "this screen"},
		{"q", "quit"},
	}
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
