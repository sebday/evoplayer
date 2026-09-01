package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sebday/evoplayer/server/library"
)

func (m model) View() string {
	if !m.ready {
		return "loading…"
	}
	if m.frames != nil && m.frames.freeze && m.frames.view != "" {
		return m.frames.view
	}
	m.search.Width = max(8, m.browseInnerWidth())
	nowPlaying := m.renderNowPlayingBar()
	footer := m.renderFooter()
	nowPlayingH := lipgloss.Height(nowPlaying)
	footerH := footerBlockHeight(footer)
	bodyH := max(5, m.contentHeight()-nowPlayingH-footerH)
	body, place := m.renderBody(bodyH)
	body = padToHeight(body, bodyH)
	var inner string
	if footer != "" {
		inner = lipgloss.JoinVertical(lipgloss.Left, nowPlaying, body, footer)
	} else {
		inner = lipgloss.JoinVertical(lipgloss.Left, nowPlaying, body)
	}
	padX, padY := m.framePad()
	out := lipgloss.NewStyle().Padding(padY, padX).Render(inner)
	g := m.playerGeom()
	if place.seq != "" {
		if m.art != nil {
			m.art.shown = true
		}
		row, col := 0, 0
		if place.atCursor {
			row = padY + nowPlayingH + place.row
			col = padX + place.col
		}
		m.storeArtOverlay(place.seq, row, col, g.artworkCols, g.artworkRows)
	} else if m.art != nil && m.art.shown {
		m.art.shown = false
		m.clearStoredArtOverlay()
		out = clearArtOverlay() + out
	}
	if m.frames != nil {
		m.frames.view = out
		nowPlayingW := g.nowPlayingInnerW
		nowPlayingRow := padY + 2 + nowPlayingPadY
		nowPlayingCol := padX + nowPlayingContentCol()
		vizRow, vizCol, vizW := 0, 0, 0
		if g.vizW >= 8 {
			vizRow = nowPlayingRow
			vizCol = vizPaintCol(nowPlayingCol, nowPlayingW)
			vizW = g.vizW
		}
		browseW := paneInnerWidth(g.browseW)
		browseH := paneInnerHeight(bodyH)
		playlistW := paneInnerWidth(g.playlistW)
		bodyStart := padY + nowPlayingH + 1
		browseRow := bodyStart + 1 + panePadY
		browseCol := padX + paneContentCol()
		playlistCol := padX + g.browseW + paneContentCol()
		lastNowPlaying := m.renderNowPlayingChrome(nowPlayingW)
		lastBrowse := m.renderBrowseInner(browseW, browseH)
		lastPlaylist := m.renderPlaylistInner(playlistW, browseH)
		m.frames.mu.Lock()
		m.frames.vizRow = vizRow
		m.frames.vizCol = vizCol
		m.frames.vizW = vizW
		m.frames.nowPlayingRow = nowPlayingRow
		m.frames.nowPlayingCol = nowPlayingCol
		m.frames.nowPlayingW = nowPlayingW
		m.frames.lastNowPlaying = lastNowPlaying
		m.frames.browseRow = browseRow
		m.frames.browseCol = browseCol
		m.frames.browseW = browseW
		m.frames.browseH = browseH
		m.frames.lastBrowse = lastBrowse
		m.frames.playlistRow = browseRow
		m.frames.playlistCol = playlistCol
		m.frames.playlistW = playlistW
		m.frames.playlistH = browseH
		m.frames.lastPlaylist = lastPlaylist
		m.frames.mu.Unlock()
	}
	return out
}

func (m model) renderSearch(width int) string {
	return padExact(m.search.View(), max(6, width))
}

func (m model) renderPlaylistTrackList(tracks []library.Track, idx, offset, width, h int, focused bool) string {
	cols := trackColWidthsPlaylist(width)
	vis := max(1, h)
	start := offset
	end := min(len(tracks), start+vis)
	var b strings.Builder
	for i := start; i < end; i++ {
		t := tracks[i]
		playing := t.Path != "" && t.Path == m.status.Path
		selected := focused && i == idx
		cursor, label := m.listCursorLead(selected, playing, trackLabel(t), 1)
		heart := playlistRowHeart(t.Liked, selected)
		fullWidth := 0
		if playing {
			fullWidth = width
		}
		fmt.Fprintf(&b, "%s\n", renderTrackColumns(cols, cursor, heart, label, trackTime(t), strings.TrimSpace(t.Year), "", selected, playing, fullWidth))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) listCursorLead(selected, playing bool, label string, width int) (cursor, out string) {
	if playing {
		if width <= 1 {
			return "▶", label
		}
		return "▶ ", label
	}
	if selected {
		if width <= 1 {
			return ">", label
		}
		return "> ", label
	}
	if width <= 1 {
		return " ", label
	}
	return "  ", label
}

type trackCols struct {
	name, time, year, genre int
}

// trackColWidthsBrowse lays out name + optional count for narrow tree rows.
func trackColWidthsBrowse(total, lead int) trackCols {
	const (
		timeW = 5
		gap   = 2
	)
	nameW := total - lead - timeW - gap
	if nameW < 6 {
		nameW = 6
	}
	return trackCols{name: nameW, time: timeW}
}

func playlistRowHeart(liked, selected bool) string {
	if liked {
		if selected {
			return "♥ "
		}
		return styleGood().Render("♥") + " "
	}
	return "  "
}

func trackColWidthsPlaylist(total int) trackCols {
	const (
		lead   = 1 // compact cursor
		heartW = 2 // ♥ before time
		timeW  = 6
		yearW  = 4
		gaps   = 2 // name-time and time-year gutters
	)
	nameW := total - lead - heartW - timeW - yearW - gaps
	if nameW < 8 {
		nameW = 8
	}
	return trackCols{name: nameW, time: timeW, year: yearW}
}

func renderTrackColumns(cols trackCols, cursor, heart, name, timeStr, year, genre string, selected, playing bool, fullWidth int) string {
	nameCell := padRight(clipEllipsis(name, cols.name), cols.name)
	if playing && !selected && !strings.Contains(name, "\x1b") {
		nameCell = styleGood().Render(nameCell)
	} else if !selected && !playing && !strings.Contains(name, "\x1b") {
		nameCell = styleText().Render(nameCell)
	}
	meta := styleMuted()
	if playing && !selected {
		meta = styleGood()
		if !strings.Contains(heart, "\x1b") {
			heart = meta.Render(heart)
		}
	}
	var row string
	if cols.time == 0 && cols.year == 0 && cols.genre == 0 {
		row = cursor + heart + nameCell
	} else if cols.year == 0 && cols.genre == 0 {
		row = cursor + heart + nameCell + "  " +
			meta.Render(padLeft(clipTruncate(timeStr, cols.time), cols.time))
	} else if cols.genre == 0 {
		timeCell := heart + meta.Render(padLeft(clipTruncate(timeStr, cols.time), cols.time))
		row = cursor + nameCell + " " +
			timeCell +
			" " +
			meta.Render(padLeft(clipEllipsis(year, cols.year), cols.year))
	} else {
		row = cursor + heart + nameCell + "  " +
			meta.Render(padLeft(clipTruncate(timeStr, cols.time), cols.time)) +
			"  " +
			meta.Render(padLeft(clipEllipsis(year, cols.year), cols.year)) +
			"  " +
			meta.Render(padRight(clipEllipsis(genre, cols.genre), cols.genre))
	}
	if fullWidth > 0 {
		row = padExact(row, fullWidth)
	}
	if selected {
		return stylePlaylistSelected().Render(row)
	}
	if playing {
		return styleGood().Render(row)
	}
	return row
}

func trackTime(t library.Track) string {
	if d := strings.TrimSpace(t.DurationLabel); d != "" && d != "0:00" {
		return d
	}
	return dur(t.Duration)
}

func clipTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "")
}

func clipEllipsis(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return ansi.Truncate(s, width, "")
	}
	return ansi.Truncate(s, width, ".")
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
	mu             sync.Mutex
	view           string
	freeze         bool
	vizRow         int
	vizCol         int
	vizW           int
	waveW          int
	waveH          int
	waveN          int
	wavePath       string
	waveRows       []string
	waveFill       []float64
	playedTo       int
	nowPlayingRow  int
	nowPlayingCol  int
	nowPlayingW    int
	lastNowPlaying string
	browseRow      int
	browseCol      int
	browseW        int
	browseH        int
	lastBrowse     string
	playlistRow    int
	playlistCol    int
	playlistW      int
	playlistH      int
	lastPlaylist   string
	artRow         int
	artCol         int
	artSeq         string
	artPlace       string
	artDirty       bool
	quitting       bool
}

func (m model) renderFooter() string {
	hints := m.renderFooterHints()
	if hints == "" {
		return ""
	}
	width := m.contentWidth()
	return fieldsetInlineFlush("", hints, width, false, 0)
}

func footerBlockHeight(footer string) int {
	if footer == "" {
		return 0
	}
	return lipgloss.Height(footer)
}

func (m model) renderFooterViz(width int) string {
	if width < 8 {
		return ""
	}
	playX := wavePlayCol(waveformPlayedFrac(m.status.Position, m.status.Duration), width)
	return strings.Join(m.overlayWaveRows(width, vizPaintRows, playX), "\n")
}

func (m model) renderFooterHints() string {
	return ""
}

func (m model) spaceHint() string {
	label := "play"
	if m.status.State == "playing" {
		label = "pause"
	}
	return hint("space", label, 1, false)
}

func hint(key, label string, panelNum int, active bool) string {
	hex := panelBorderHex(panelNum, active)
	keyStyle := logoColor().NewStyle().Foreground(lipgloss.Color(hex)).Bold(true)
	return keyStyle.Render(key) + " " + styleMuted().Render(label)
}

func dur(sec float64) string {
	if sec <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d", int(sec)/60, int(sec)%60)
}
