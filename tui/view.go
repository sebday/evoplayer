package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/server/library"
)

func (m model) View() string {
	if !m.ready {
		return "loading…"
	}
	if m.frames != nil && m.frames.freeze && m.frames.view != "" {
		return m.frames.view
	}
	m.search.Width = max(8, m.searchBarWidth())
	header := m.renderHeader()
	footer := m.renderFooter()
	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	bodyRowH := m.bodyRowHeight(headerH, footerH)
	body, place := m.renderListWithArt(bodyRowH)
	body = padToHeight(body, bodyRowH)
	inner := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	padX, padY := m.framePad()
	out := lipgloss.NewStyle().Padding(padY, padX).Render(inner)
	if place.seq != "" {
		frame := 0
		if m.art != nil {
			m.art.shown = true
			m.art.frame++
			frame = m.art.frame
		}
		row := padY + headerH + place.row
		col := padX + place.col
		out = overlayArt(out, place.seq, row, col, frame)
	} else if m.art != nil && m.art.shown {
		m.art.shown = false
		out = clearArtOverlay() + out
	}
	if m.frames != nil {
		m.frames.view = out
		m.frames.mu.Lock()
		m.frames.vizRow = padY + headerH + bodyRowH + 2
		m.frames.vizCol = padX + 2
		m.frames.vizW = max(8, m.contentWidth()-2)
		m.frames.mu.Unlock()
	}
	return out
}

func (m model) renderHeader() string {
	contentW := m.contentWidth()
	innerW := max(8, contentW-4)
	logo := renderLogoText()
	active := m.focus == focusNav || m.focus == focusSearch
	searchW := m.searchBarWidth()
	search := m.renderSearch(searchW)
	icons := m.renderHeaderIcons()
	gap := 2
	used := lipgloss.Width(icons) + lipgloss.Width(search) + gap*2
	tabsW := innerW - used
	if tabsW < 8 {
		tabsW = 8
	}
	tabs := m.renderHeaderTabs(tabsW)
	rightPad := innerW - lipgloss.Width(icons) - lipgloss.Width(tabs) - lipgloss.Width(search) - gap
	if rightPad < 1 {
		rightPad = 1
	}
	leftGap := strings.Repeat(" ", gap)
	row := lipgloss.JoinHorizontal(lipgloss.Top, icons, leftGap, tabs, strings.Repeat(" ", rightPad), search)
	return fieldsetInline(logo, row, contentW, active, m.pulsePhase, 1)
}

func (m model) renderHeaderIcons() string {
	var parts []string
	for i, item := range m.nav {
		g := navGlyph(item)
		if g == "" {
			continue
		}
		if i == m.navIdx {
			parts = append(parts, styleSelected().Render(g))
		} else {
			parts = append(parts, styleMuted().Render(g))
		}
	}
	return strings.Join(parts, "  ")
}

func (m model) renderHeaderTabs(width int) string {
	var parts []string
	for i, item := range m.nav {
		if navIsIcon(item) {
			continue
		}
		label := item.Label
		if i == m.navIdx {
			label = styleSelected().Render(label)
		} else {
			label = styleMuted().Render(label)
		}
		if item.Count > 0 {
			label += " " + styleMuted().Render(fmt.Sprintf("%d", item.Count))
		}
		parts = append(parts, label)
	}
	row := strings.Join(parts, "   ")
	return lipgloss.NewStyle().MaxWidth(width).Render(row)
}

func (m model) renderSearch(width int) string {
	return padExact(m.search.View(), max(6, width))
}

func (m model) renderPaneInner(innerH int) string {
	var body string
	switch {
	case m.helpSelected():
		body = m.renderHelpBody()
	case m.downloadSelected():
		body = m.renderDownload()
	case m.loading:
		body = styleMuted().Render("loading…")
	case m.err != "":
		body = styleWarn().Render(m.err)
	case m.listLen() == 0:
		if m.filetreeSelected() {
			body = styleMuted().Render("no folders")
		} else {
			body = styleMuted().Render("no tracks")
		}
	default:
		sub := m.paneSubtitle()
		tracks := m.renderTracks(m.listVisible())
		if sub != "" {
			body = sub + "\n\n" + tracks
		} else {
			body = tracks
		}
	}
	if m.currentSelected() && !m.staticSelected() {
		title := m.nowPlayingTitleRow(m.listInnerWidth())
		head := m.renderNowPlayingHead(m.listInnerWidth())
		prefix := title
		if head != "" {
			prefix = title + "\n" + head
		}
		if body == "" {
			body = prefix
		} else {
			body = prefix + "\n\n" + body
		}
	}
	return clipLines(body, innerH)
}

func (m model) paneLegend() string {
	if m.downloadSelected() {
		return "Downloads"
	}
	if m.helpSelected() {
		return "Help"
	}
	name := "Library"
	if m.navIdx >= 0 && m.navIdx < len(m.nav) {
		name = m.nav[m.navIdx].Label
	}
	n := m.listLen()
	if m.filetreeSelected() {
		return fmt.Sprintf("%s (%d folders)", name, m.browseFolderCount())
	}
	if m.currentSelected() {
		return m.currentLegend(max(8, m.mainWidth()-6))
	}
	return fmt.Sprintf("%s (%d tracks)", name, n)
}

func (m model) browseFolderCount() int {
	n := 0
	for _, e := range m.browse {
		if e.Type == "dir" {
			n++
		}
	}
	return n
}

func (m model) paneSubtitle() string {
	if m.search.Value() != "" {
		return styleMuted().Render("filter: " + m.search.Value())
	}
	if m.filetreeSelected() {
		if m.browsePath != "" {
			return styleMuted().Render(m.browsePath)
		}
		return ""
	}
	return ""
}

func (m model) renderTracks(h int) string {
	if m.filetreeSelected() {
		return m.renderBrowse(h)
	}
	width := m.listInnerWidth()
	cols := trackColWidths(width)
	vis := max(1, h)
	start := m.listOffset
	end := min(len(m.filtered), start+vis)
	var b strings.Builder
	for i := start; i < end; i++ {
		t := m.filtered[i]
		label := trackLabel(t)
		heart := heartPrefix(t.Liked)
		playing := t.Path != "" && t.Path == m.status.Path
		cursor, label := m.listCursor(i == m.listIdx, playing, label)
		fmt.Fprintf(&b, "%s\n", renderTrackColumns(cols, cursor, heart, label, trackTime(t), strings.TrimSpace(t.Year), strings.TrimSpace(t.Genre), i == m.listIdx, playing))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) renderBrowse(h int) string {
	width := m.listInnerWidth()
	cols := trackColWidths(width)
	vis := max(1, h)
	start := m.listOffset
	end := min(len(m.browse), start+vis)
	var b strings.Builder
	for i := start; i < end; i++ {
		e := m.browse[i]
		label := browseLabel(e)
		heart := heartPrefix(false)
		timeStr, year, genre := "", "", ""
		switch e.Type {
		case "dir":
			if e.Count > 0 {
				timeStr = fmt.Sprintf("%d", e.Count)
			}
		case "track":
			timeStr = trackTime(e.Track)
			year = strings.TrimSpace(e.Year)
			genre = strings.TrimSpace(e.Genre)
			heart = heartPrefix(e.Liked)
		}
		selected := i == m.listIdx
		playing := e.Type == "track" && e.Path != "" && e.Path == m.status.Path
		cursor, label := m.listCursor(selected, playing, label)
		if !selected && !playing && (e.Type == "dir" || e.Type == "parent") {
			label = styleMuted().Render(label)
		}
		fmt.Fprintf(&b, "%s\n", renderTrackColumns(cols, cursor, heart, label, timeStr, year, genre, selected, playing))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) listCursor(selected, playing bool, label string) (cursor, out string) {
	if playing {
		return styleGood().Render("▶ "), styleGood().Render(label)
	}
	if selected {
		return styleSelected().Render("> "), styleSelected().Render(label)
	}
	return "  ", label
}

type trackCols struct {
	name, time, year, genre int
}

func trackColWidths(total int) trackCols {
	const (
		lead  = 5 // cursor + heart
		timeW = 6
		yearW = 4
		gaps  = 6 // three 2-space gutters
	)
	genreW := 12
	nameW := total - lead - timeW - yearW - genreW - gaps
	if nameW < 10 {
		shrink := 10 - nameW
		genreW = max(6, genreW-shrink)
		nameW = total - lead - timeW - yearW - genreW - gaps
		if nameW < 8 {
			nameW = 8
		}
	}
	return trackCols{name: nameW, time: timeW, year: yearW, genre: genreW}
}

func renderTrackColumns(cols trackCols, cursor, heart, name, timeStr, year, genre string, selected, playing bool) string {
	nameCell := padRight(clipEllipsis(name, cols.name), cols.name)
	if playing {
		if !strings.Contains(name, "\x1b") {
			nameCell = styleGood().Render(nameCell)
		}
	} else if !selected && !strings.Contains(name, "\x1b") {
		nameCell = styleText().Render(nameCell)
	}
	meta := styleMuted()
	if playing {
		meta = styleGood()
		if !strings.Contains(heart, "\x1b") {
			heart = meta.Render(heart)
		}
	}
	return cursor + heart + nameCell + "  " +
		meta.Render(padLeft(clipEllipsis(timeStr, cols.time), cols.time)) +
		"  " +
		meta.Render(padLeft(clipEllipsis(year, cols.year), cols.year)) +
		"  " +
		meta.Render(padRight(clipEllipsis(genre, cols.genre), cols.genre))
}

func trackTime(t library.Track) string {
	if d := strings.TrimSpace(t.DurationLabel); d != "" && d != "0:00" {
		return d
	}
	return dur(t.Duration)
}

func clipEllipsis(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 3 {
		return lipgloss.NewStyle().MaxWidth(width).Render(s)
	}
	return lipgloss.NewStyle().MaxWidth(width-3).Render(s) + "..."
}

func padLeft(s string, width int) string {
	n := width - lipgloss.Width(s)
	if n <= 0 {
		return s
	}
	return strings.Repeat(" ", n) + s
}

func padRight(s string, width int) string {
	n := width - lipgloss.Width(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

type frameCache struct {
	mu       sync.Mutex
	view     string
	freeze   bool
	vizRow   int
	vizCol   int
	vizW     int
	waveW    int
	waveH    int
	waveN    int
	wavePath string
	waveRows []string
	waveFill []float64
	playedTo int
}

func (m model) renderFooter() string {
	width := m.contentWidth()
	innerW := max(8, width-4)
	viz := m.renderFooterViz(innerW)
	h := lipgloss.Height(viz) + 2
	if viz == "" {
		h = 3
	}
	return fieldsetPad(m.renderPlayTime(), viz, width, h, false, 0, 0, 1, "", m.renderFooterHints(), 4)
}

func (m model) renderFooterViz(width int) string {
	if width < 8 {
		return ""
	}
	playX := wavePlayCol(waveformPlayedFrac(m.status.Position, m.status.Duration), width)
	return strings.Join(m.overlayWaveRows(width, vizPaintRows, playX), "\n")
}

func (m model) renderPlayTime() string {
	pos := playbackLabel(m.status.PositionLabel, m.status.Position)
	dur := playbackLabel(m.status.DurationLabel, m.status.Duration)
	if pos == "" {
		pos = "0:00"
	}
	if dur == "" {
		dur = "0:00"
	}
	return pos + "/" + dur
}

func (m model) renderFooterHints() string {
	hints := []string{
		hint("←→↑↓", "browse"),
	}
	space := m.spaceHint()
	if m.helpSelected() {
		hints = append(hints,
			hint("l", "like"),
			space,
			hint("/", "find"),
			hint("?", "help"),
			hint("q", "quit"),
		)
		return joinHints(hints)
	}
	if m.downloadSelected() {
		hints = append(hints,
			hint("⏎", "run"),
			space,
			hint("?", "help"),
			hint("q", "quit"),
		)
		return joinHints(hints)
	}
	hints = append(hints,
		hint("⏎", "play"),
		hint("+", "add"),
		hint("l", "like"),
		space,
		hint("/", "find"),
		hint("?", "help"),
		hint("q", "quit"),
	)
	return joinHints(hints)
}

func joinHints(hints []string) string {
	return strings.Join(hints, "   ")
}

func (m model) spaceHint() string {
	label := "play"
	if m.status.State == "playing" {
		label = "pause"
	}
	return hint("space", label)
}

func hint(key, label string) string {
	return styleSelected().Render(key) + " " + styleMuted().Render(label)
}

func heartPrefix(liked bool) string {
	if liked {
		return styleGood().Render("♥") + "  "
	}
	return "   "
}

func dur(sec float64) string {
	if sec <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d", int(sec)/60, int(sec)%60)
}
