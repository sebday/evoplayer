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
		{"↑↓", "focused column"},
		{"tab", "files / playlist"},
		{"←→", "folders"},
		{"⏎", "play"},
		{"d", "add dir"},
		{"a", "art"},
		{"l", "like"},
		{"v", "visualizer on / off"},
		{"space", "play / pause"},
		{", .", "previous / next"},
		{"< >", "skip 10s"},
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
