package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const headerLogoSearchGap = 4

func (m model) View() string {
	if !m.ready {
		return "loading…"
	}
	logo := m.renderHeader()
	rule := styleMuted().Render(strings.Repeat("─", max(8, m.contentWidth())))
	footer := m.renderFooter()
	var right string
	if m.nowPlayingSelected() {
		right = m.renderNowPlaying(m.paneHeight())
	} else if m.helpSelected() {
		right = m.renderHelp(m.paneHeight())
	} else {
		right = m.renderPane(m.paneHeight())
	}
	nav := m.renderNav(lipgloss.Height(right))
	body := lipgloss.JoinHorizontal(lipgloss.Top, nav, "  ", right)
	inner := lipgloss.JoinVertical(lipgloss.Left, logo, "", rule, "", body, "", footer)
	padX, padY := m.framePad()
	return lipgloss.NewStyle().Padding(padY, padX).Render(inner)
}

func (m model) renderHeader() string {
	contentW := m.contentWidth()
	if m.helpSelected() {
		return renderLogoAt(max(8, contentW), m.logoPhase)
	}
	searchW := m.searchBarWidth()
	logo := renderLogoAt(max(8, contentW-searchW-headerLogoSearchGap), m.logoPhase)
	search := m.renderSearch(searchW, 3)
	gap := contentW - lipgloss.Width(logo) - lipgloss.Width(search)
	if gap < 0 {
		gap = 0
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, logo, strings.Repeat(" ", gap), search)
}

func (m model) renderNav(height int) string {
	innerW := m.navWidth()
	var b strings.Builder
	for i, item := range m.nav {
		if item.Kind == "settings" && i > 0 && m.nav[i-1].Kind != "settings" {
			b.WriteString("\n")
		}
		if i > 0 && m.nav[i-1].Kind == "nowplaying" {
			b.WriteString("\n")
		}
		prefix := "  "
		label := item.Label
		count := ""
		if item.Count > 0 {
			count = fmt.Sprintf("%d", item.Count)
		}
		if i == m.navIdx {
			prefix = "▌ "
			label = styleNavActive().Render(label)
		} else if item.Kind == "settings" || item.Kind == "help" {
			label = styleMuted().Render(label)
		}
		gap := innerW - 2 - lipgloss.Width(item.Label) - lipgloss.Width(count)
		if gap < 1 {
			gap = 1
		}
		fmt.Fprintf(&b, "%s%s%s%s\n", prefix, label, strings.Repeat(" ", gap), styleMuted().Render(count))
	}
	return lipgloss.NewStyle().
		Width(innerW).
		Height(max(6, height)).
		Render(strings.TrimRight(b.String(), "\n"))
}

func (m model) renderSearch(width, height int) string {
	return fieldset("Search", m.search.View(), width, height, m.focus == focusSearch)
}

func (m model) renderPane(height int) string {
	legend := m.paneLegend()
	var body string
	switch {
	case m.settingsSelected():
		body = m.renderSettings()
	case m.loading:
		body = styleMuted().Render("loading…")
	case m.err != "":
		body = styleWarn().Render(m.err)
	case m.listLen() == 0:
		body = styleMuted().Render("no tracks")
	default:
		sub := m.paneSubtitle()
		tracks := m.renderTracks(m.listVisible())
		if sub != "" {
			body = sub + "\n\n" + tracks
		} else {
			body = tracks
		}
	}
	return fieldset(legend, body, m.mainWidth(), height, m.focus == focusList)
}

func (m model) paneLegend() string {
	if m.settingsSelected() {
		return "Settings"
	}
	if m.nowPlayingSelected() {
		return "Now playing"
	}
	if m.helpSelected() {
		return "Help"
	}
	name := "Library"
	if m.navIdx >= 0 && m.navIdx < len(m.nav) {
		name = m.nav[m.navIdx].Label
	}
	return fmt.Sprintf("%s (%d)", name, m.listLen())
}

func (m model) paneSubtitle() string {
	if m.search.Value() != "" {
		return styleMuted().Render("filter: " + m.search.Value())
	}
	if m.filetreeSelected() {
		if m.browsePath == "" {
			return styleMuted().Render("folders")
		}
		return styleMuted().Render(m.browsePath)
	}
	return styleMuted().Render("tracks in this playlist")
}

func (m model) renderTracks(h int) string {
	if m.filetreeSelected() {
		return m.renderBrowse(h)
	}
	vis := max(1, h)
	start := m.listOffset
	end := min(len(m.filtered), start+vis)
	var b strings.Builder
	for i := start; i < end; i++ {
		t := m.filtered[i]
		label := trackLabel(t)
		meta := dur(t.Duration)
		heart := "  "
		if t.Liked {
			heart = styleGood().Render("♥ ")
		}
		cursor := "  "
		if t.Path != "" && t.Path == m.status.Path {
			cursor = styleGood().Render("▶ ")
		}
		if i == m.listIdx {
			cursor = styleSelected().Render("> ")
			label = styleSelected().Render(label)
		}
		fmt.Fprintf(&b, "%s%s%s  %s\n", cursor, heart, label, styleMuted().Render(meta))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) renderBrowse(h int) string {
	vis := max(1, h)
	start := m.listOffset
	end := min(len(m.browse), start+vis)
	var b strings.Builder
	for i := start; i < end; i++ {
		e := m.browse[i]
		label := browseLabel(e)
		meta := ""
		heart := "  "
		switch e.Type {
		case "dir":
			if e.Count > 0 {
				meta = fmt.Sprintf("%d", e.Count)
			}
		case "track":
			meta = dur(e.Duration)
			if e.Liked {
				heart = styleGood().Render("♥ ")
			}
		}
		cursor := "  "
		if e.Type == "track" && e.Path != "" && e.Path == m.status.Path {
			cursor = styleGood().Render("▶ ")
		}
		if i == m.listIdx {
			cursor = styleSelected().Render("> ")
			label = styleSelected().Render(label)
		} else if e.Type == "dir" || e.Type == "parent" {
			label = styleMuted().Render(label)
		}
		fmt.Fprintf(&b, "%s%s%s  %s\n", cursor, heart, label, styleMuted().Render(meta))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) renderSettings() string {
	return strings.Join([]string{
		styleMuted().Render("player"),
		"  evoplayer",
		"",
		styleMuted().Render("music root"),
		"  " + m.env.MusicRoot,
		"",
		styleMuted().Render("volume"),
		fmt.Sprintf("  %d", m.status.Volume),
		"",
		styleMuted().Render("socket"),
		"  " + m.env.SocketPath,
	}, "\n")
}

func (m model) renderFooter() string {
	hints := []string{
		hint("←→", "focus"),
		hint("↑↓", "browse"),
	}
	if m.helpSelected() {
		hints = append(hints,
			hint("l", "like"),
			hint("space", "play / pause"),
			hint(",/.", "skip"),
			hint("?", "help"),
			hint("q", "quit"),
		)
		return strings.Join(hints, "   ")
	}
	if m.nowPlayingSelected() {
		hints = append(hints,
			hint("l", "like"),
			hint("space", "play / pause"),
			hint(",/.", "skip"),
			hint("/", "find"),
			hint("?", "help"),
			hint("q", "quit"),
		)
		return strings.Join(hints, "   ")
	}
	hints = append(hints,
		hint("tab", "focus"),
		hint("⏎", "add"),
		hint("l", "like"),
		hint("space", "play / pause"),
		hint(",/.", "skip"),
		hint("/", "find"),
		hint("?", "help"),
		hint("q", "quit"),
	)
	return strings.Join(hints, "   ")
}

func hint(key, label string) string {
	return styleSelected().Render(key) + " " + styleMuted().Render(label)
}

func dur(sec float64) string {
	if sec <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d", int(sec)/60, int(sec)%60)
}
