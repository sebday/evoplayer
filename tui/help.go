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
		{"↑↓", "move selection"},
		{"shift+↑↓", "reorder playlist"},
		{"pgup dn", "page scroll"},
		{"tab", "files / playlist"},
		{"shift+tab", "playlist / files"},
		{"← →", "parent / enter folder"},
		{"backspace", "parent folder"},
		{"⏎", "play / run"},
		{"space", "play / pause"},
		{", .", "previous / next"},
		{"< >", "skip 10s"},
		{"- =", "volume"},
		{"/", "find"},
		{"d", "add dir"},
		{"f", "folder"},
		{"l", "like"},
		{"m", "move"},
		{"e", "edit tags"},
		{"tab", "next tag field (in editor)"},
		{"L", "like playing"},
		{"a", "art"},
		{"s", "art (track)"},
		{"v", "visualizer on / off"},
		{"V", "visualizer prev"},
		{"esc", "back"},
		{"?", "help"},
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
