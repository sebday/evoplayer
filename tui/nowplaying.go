package tui

import (
	"fmt"
	"image"
	"math"
	"strings"
	"unicode"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/lipgloss"
)

const (
	browseColW = 27
	panePadX   = 1
	panePadY   = 1
)

type artworkPlacement struct {
	seq      string
	row      int
	col      int
	atCursor bool
}

type playerGeom struct {
	browseW, playlistW, artworkW int
	artworkCols, artworkRows     int
	bodyH, nowPlayingH           int
	nowPlayingInnerW             int
	vizW                         int
}

func (m model) playerGeom() playerGeom {
	total := m.mainWidth()
	footerH := footerBlockHeight(m.renderFooter())
	estNowPlayingH := vizPaintRows + 2
	bodyH := max(5, m.contentHeight()-estNowPlayingH-footerH)

	cw, ch := artCellSize()
	if cw < 1 {
		cw = 8
	}
	if ch < 1 {
		ch = 16
	}
	const (
		minBrowse     = 22
		minPlaylist   = 34
		minNowPlaying = 24
		chromeW       = 4
	)
	maxArtworkW := total / 2
	maxArtworkCols := total - minBrowse - minPlaylist - chromeW
	if cap := max(8, maxArtworkW-chromeW); maxArtworkCols > cap {
		maxArtworkCols = cap
	}
	if maxArtworkCols < 8 {
		maxArtworkCols = 8
	}

	maxRows := max(4, bodyH-2)
	artworkCols, artworkRows := squareArtworkFit(maxArtworkCols, maxRows, cw, ch)
	artworkW := artworkCols + chromeW
	browseW := browseColW
	playlistW := total - browseW - artworkW
	if playlistW < minPlaylist {
		shrink := minPlaylist - playlistW
		artworkW = max(chromeW+8, artworkW-shrink)
		artworkCols = max(8, artworkW-chromeW)
		artworkCols, artworkRows = squareArtworkFit(artworkCols, maxRows, cw, ch)
		artworkW = artworkCols + chromeW
		playlistW = total - browseW - artworkW
		if playlistW < minPlaylist {
			browseW = max(minBrowse, browseW-(minPlaylist-playlistW))
			playlistW = total - browseW - artworkW
		}
	}
	browseW = max(minBrowse, browseW)
	playlistW = max(8, playlistW)

	nowInner := max(24, total-2)
	contentW := max(8, nowInner-2*nowPlayingPadX)
	vizW := max(8, artworkW-2)
	nowPlayingW := contentW - vizW - 1
	if nowPlayingW < minNowPlaying {
		nowPlayingW = minNowPlaying
		vizW = max(8, contentW-nowPlayingW-1)
	}
	if vizW < 8 {
		vizW = 0
		nowPlayingW = max(minNowPlaying, contentW)
	}

	nowPlayingH := lipgloss.Height(m.renderNowPlaying(nowPlayingW)) + 3
	if vizW >= 8 {
		nowPlayingH = max(nowPlayingH, vizPaintRows+2)
	}
	bodyH = max(5, m.contentHeight()-nowPlayingH-footerH)
	maxRows = max(4, bodyH-2)
	if artworkRows > maxRows || artworkW > maxArtworkW {
		colsCap := maxArtworkCols
		if artworkW > maxArtworkW {
			colsCap = min(colsCap, max(8, maxArtworkW-chromeW))
		}
		artworkCols, artworkRows = squareArtworkFit(colsCap, maxRows, cw, ch)
		artworkW = artworkCols + chromeW
		playlistW = max(8, total-browseW-artworkW)
	}

	return playerGeom{
		browseW:          browseW,
		playlistW:        playlistW,
		artworkW:         artworkW,
		artworkCols:      artworkCols,
		artworkRows:      artworkRows,
		bodyH:            bodyH,
		nowPlayingH:      nowPlayingH,
		nowPlayingInnerW: nowPlayingW,
		vizW:             vizW,
	}
}

func squareArtworkFit(maxCols, maxRows, cw, ch int) (cols, rows int) {
	maxCols = max(8, maxCols)
	maxRows = max(4, maxRows)
	cols = maxCols
	rows = squareArtworkRows(cols, cw, ch)
	if rows > maxRows {
		rows = maxRows
		cols = min(maxCols, squareArtworkCols(rows, cw, ch))
		rows = squareArtworkRows(cols, cw, ch)
		if rows > maxRows {
			rows = maxRows
		}
	}
	return max(8, cols), max(4, rows)
}

func squareArtworkRows(cols, cw, ch int) int {
	if cw < 1 {
		cw = 8
	}
	if ch < 1 {
		ch = 16
	}
	// Floor so the pane is never taller than a pixel-square. Rounding up left a
	// black band under kitty/sixel art when the image used the shorter side.
	return max(4, cols*cw/ch)
}

func squareArtworkCols(rows, cw, ch int) int {
	if cw < 1 {
		cw = 8
	}
	if ch < 1 {
		ch = 16
	}
	cols := (rows*ch + cw/2) / cw
	return max(8, cols)
}

func artPaneHeight(rows int) int {
	return max(6, rows+2)
}

func padNowPlayingGap(chrome string, rows int) string {
	lines := strings.Split(chrome, "\n")
	w := 0
	for _, s := range lines {
		if n := lipgloss.Width(s); n > w {
			w = n
		}
	}
	out := []string{padExact("", w)}
	out = append(out, lines...)
	if len(out) > rows {
		out = out[:rows]
	}
	for len(out) < rows {
		out = append(out, padExact("", w))
	}
	for i := range out {
		out[i] = padExact(out[i], w)
	}
	return strings.Join(out, "\n")
}

func (m model) renderNowPlayingChrome(width int) string {
	pad := nowPlayingPadLeft
	inner := max(24, width-pad)
	chrome := padNowPlayingGap(m.renderNowPlaying(inner), vizPaintRows)
	if pad < 1 {
		return chrome
	}
	prefix := strings.Repeat(" ", pad)
	lines := strings.Split(chrome, "\n")
	for i := range lines {
		lines[i] = padExact(prefix+lines[i], width)
	}
	return strings.Join(lines, "\n")
}

func (m model) renderNowPlayingBar() string {
	g := m.playerGeom()
	chrome := m.renderNowPlayingChrome(g.nowPlayingInnerW)
	inner := chrome
	if g.vizW >= 8 {
		waveW := vizWaveWidth(g.vizW)
		raw := padToHeight(m.renderFooterViz(waveW), vizPaintRows)
		lines := strings.Split(raw, "\n")
		if len(lines) > vizPaintRows {
			lines = lines[:vizPaintRows]
		}
		wave := strings.Join(padVizInner(lines, g.vizW, waveW), "\n")
		inner = lipgloss.JoinHorizontal(lipgloss.Top, chrome, " ", wave)
	}
	h := lipgloss.Height(inner) + 2
	return fieldsetPad(m.nowPlayingLegend(g.nowPlayingInnerW), "", inner, m.mainWidth(), h, false, nowPlayingPadY, nowPlayingPadX, "", m.nowPlayingHintLegend(), 1)
}

func (m model) nowPlayingHintLegend() string {
	return m.spaceHint() + "  " + hint("/", "find", 1, false) + "  " + hint("?", "help", 1, false)
}

func (m model) renderBody(height int) (string, artworkPlacement) {
	g := m.playerGeom()
	g.bodyH = height
	browse := m.renderBrowsePane(g)
	playlist := m.renderPlaylistPane(g)
	artwork, place := m.renderArtworkPane(g)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, browse, playlist, artwork)
	if place.seq != "" && place.atCursor {
		place.col = lipgloss.Width(browse) + lipgloss.Width(playlist) + 2 + panePadX
		place.row = 2
	}
	return joined, place
}

func paneInnerWidth(boxW int) int {
	return max(8, boxW-2-2*panePadX)
}

func paneInnerHeight(bodyH int) int {
	return max(1, bodyH-2-2*panePadY)
}

func (m model) renderBrowsePane(g playerGeom) string {
	innerW := paneInnerWidth(g.browseW)
	innerH := paneInnerHeight(g.bodyH)
	browseFocused := m.focus == focusBrowse || m.focus == focusSearch
	return fieldsetPad(m.browseLegend(innerW), m.browseLegendTick(), m.renderBrowseInner(innerW, innerH), g.browseW, g.bodyH, browseFocused, panePadY, panePadX, m.browseHintLegend(), "", 2)
}

func (m model) browseLegendTick() string {
	if m.browsePath == "" || m.searching() {
		return ""
	}
	return "─"
}

func (m model) browseHintLegend() string {
	active := m.focus == focusBrowse || m.focus == focusSearch
	return hint("⏎", "play", 2, active) + "  " + hint("d", "add dir", 2, active)
}

func (m model) renderPlaylistPane(g playerGeom) string {
	innerW := paneInnerWidth(g.playlistW)
	innerH := paneInnerHeight(g.bodyH)
	bottomLeft := hint("l", "like", 3, m.focus == focusPlaylist) + "  " + hint("m", "move", 3, m.focus == focusPlaylist) + "  " + hint("e", "edit", 3, m.focus == focusPlaylist)
	if !m.status.Shuffle && m.focus == focusPlaylist {
		bottomLeft += "  " + hint("⇧↑↓", "reorder", 3, true)
	}
	if m.tagEditor {
		bottomLeft = hint("tab", "next", 3, m.focus == focusPlaylist) + "  " + hint("⏎", "save", 3, m.focus == focusPlaylist)
	} else if m.movePicker {
		bottomLeft = hint("⏎", "move", 3, m.focus == focusPlaylist)
	} else if m.artPicker {
		bottomLeft = hint("⏎", "set", 3, m.focus == focusPlaylist) + "  " + hint("s", "track", 3, m.focus == focusPlaylist)
	} else if m.settingsSelected() {
		bottomLeft = hint("⏎", "save", 3, m.focus == focusPlaylist)
	} else if m.helpSelected() {
		bottomLeft = ""
	}
	return fieldsetPad(m.playlistLegend(innerW), "", m.renderPlaylistInner(innerW, innerH), g.playlistW, g.bodyH, m.focus == focusPlaylist, panePadY, panePadX, bottomLeft, "", 3)
}

func (m model) renderArtworkPane(g playerGeom) (string, artworkPlacement) {
	layout, seq, overlay := m.cachedArtwork(g.artworkCols, g.artworkRows)
	place := artworkPlacement{}
	if overlay {
		place.seq = seq
		place.atCursor = !strings.Contains(layout, kittyPlaceholder)
	}
	// Pane height tracks the square art grid exactly; don't stretch to bodyH or
	// blank inner rows show as a black band above the bottom border.
	artH := artPaneHeight(g.artworkRows)
	var pane string
	if overlay {
		pane = fieldsetArt(m.artworkLegend(), layout, g.artworkW, artH, panePadX, g.artworkCols, hint("a", "art", 4, false), 4)
	} else {
		pane = fieldsetPad(m.artworkLegend(), "", layout, g.artworkW, artH, false, 0, panePadX, "", hint("a", "art", 4, false), 4)
	}
	return pane, place
}

func (m model) renderBrowseInner(width, innerH int) string {
	if m.libraryScanning() {
		return clipLines(styleMuted().Render("Please wait for library to finish scanning"), innerH)
	}
	var head []string
	if m.focus == focusSearch {
		m.search.Width = max(6, width-2)
		head = append(head, m.renderSearch(width))
	} else if sub := m.browseSubtitle(); sub != "" {
		head = append(head, sub)
	}
	bodyH := max(1, innerH-len(head))
	body := m.renderBrowseRows(width, bodyH)
	if len(head) == 0 {
		return clipLines(body, innerH)
	}
	return clipLines(strings.Join(append(head, body), "\n"), innerH)
}

func (m model) renderBrowseRows(width, vis int) string {
	playlists := m.sidebarPlaylists()
	tools := m.sidebarTools()
	pending := m.playlistsPending()
	nBrowse := len(m.browse)
	nPlaylists := m.playlistSlotCount()
	nTools := len(tools)
	total := nBrowse + nPlaylists + nTools
	if total == 0 {
		if m.searching() {
			return styleMuted().Render("no matches")
		}
		if m.leftLoading() {
			return styleMuted().Render("loading…")
		}
		return styleMuted().Render("no folders")
	}
	vis = max(1, vis)
	start := m.browseOffset
	end := min(total, start+vis)
	focused := m.focus == focusBrowse || m.focus == focusSearch
	browseFocused := m.focus == focusBrowse || m.focus == focusSearch
	var browseLines, playlistLines, toolLines []string
	for i := start; i < end; i++ {
		if i < nBrowse {
			browseLines = append(browseLines, m.renderBrowseRow(i, width, focused))
			continue
		}
		rel := i - nBrowse
		if pending && rel == 0 {
			playlistLines = append(playlistLines, styleMuted().Render("loading…"))
			continue
		}
		if rel < nPlaylists {
			playlistLines = append(playlistLines, m.renderPlaylistRow(playlists[rel], i, width, focused))
			continue
		}
		ti := rel - nPlaylists
		if ti >= 0 && ti < nTools {
			toolLines = append(toolLines, m.renderPlaylistRow(tools[ti], i, width, focused))
		}
	}
	var parts []string
	if len(browseLines) == 0 && m.leftLoading() && nBrowse == 0 && !m.searching() {
		parts = append(parts, styleMuted().Render("loading…"))
	}
	if len(browseLines) > 0 {
		parts = append(parts, strings.Join(browseLines, "\n"))
	}
	if len(playlistLines) > 0 || pending {
		var block strings.Builder
		block.WriteString(browseSectionRule(width, browseFocused))
		block.WriteByte('\n')
		if len(playlistLines) > 0 {
			block.WriteString(strings.Join(playlistLines, "\n"))
		} else if pending {
			block.WriteString(styleMuted().Render("loading…"))
		}
		parts = append(parts, block.String())
	}
	if len(toolLines) > 0 {
		block := strings.Join(toolLines, "\n")
		if len(parts) > 0 {
			parts = append(parts, "", block)
		} else {
			parts = append(parts, block)
		}
	}
	return strings.TrimRight(strings.Join(parts, "\n"), "\n")
}

func (m model) renderBrowseRow(i, width int, focused bool) string {
	e := m.browse[i]
	plain := browseLabel(e)
	if m.searching() && e.Type == "track" {
		plain = trackLabel(e.Track)
	}
	selected := focused && i == m.browseIdx
	playing := e.Type == "track" && e.Path != "" && e.Path == m.status.Path
	cur := " "
	if playing {
		cur = "▶"
	} else if selected {
		cur = ">"
	}
	count := ""
	if e.Type == "dir" && e.Count > 0 {
		count = fmt.Sprintf("%d", e.Count)
	}
	countW := lipgloss.Width(count)
	nameW := max(6, width-lipgloss.Width(cur)-countW)
	if countW > 0 {
		nameW = max(6, width-lipgloss.Width(cur)-countW-1)
	}
	name := clipEllipsis(plain, nameW)
	if playing {
		cur = styleGood().Render(cur)
		name = styleGood().Render(name)
	} else if selected {
		cur = styleSelected().Render(cur)
		name = styleSelected().Render(name)
	} else if e.Type == "dir" {
		name = styleMuted().Render(name)
	} else {
		name = styleText().Render(name)
	}
	row := cur + name
	if count != "" {
		pad := max(1, width-lipgloss.Width(row)-countW)
		row += strings.Repeat(" ", pad) + styleMuted().Render(count)
	}
	return row
}

func (m model) renderPlaylistRow(item navItem, i, width int, focused bool) string {
	selected := focused && i == m.browseIdx
	label := item.Label
	count := ""
	if item.Count > 0 {
		count = fmt.Sprintf("%d", item.Count)
	}
	const lead = 1
	cursor, label := m.listCursorLead(selected, false, label, lead)
	return renderTrackColumns(trackColWidthsBrowse(width, lead), cursor, "", label, count, "", "", selected, false, 0)
}

func (m model) renderPlaylistInner(width, innerH int) string {
	switch {
	case m.tagEditor:
		return clipLines(m.renderTagEditor(width), innerH)
	case m.movePicker:
		return clipLines(m.renderMovePicker(width), innerH)
	case m.artPicker:
		return clipLines(m.renderArtPicker(width), innerH)
	case m.helpSelected():
		return clipLines(m.renderHelpBody(), innerH)
	case m.settingsSelected():
		return clipLines(m.renderSettings(width), innerH)
	case m.err != "" && m.focus == focusPlaylist:
		return clipLines(styleWarn().Render(m.err), innerH)
	case len(m.queueFiltered) == 0:
		return clipLines(styleMuted().Render("no tracks"), innerH)
	}
	return clipLines(m.renderPlaylistTrackList(m.queueFiltered, m.playlistIdx, m.playlistOffset, width, max(1, innerH), m.focus == focusPlaylist), innerH)
}

func (m model) renderMovePicker(width int) string {
	if m.movePickBusy {
		return styleMuted().Render("moving…")
	}
	var b strings.Builder
	if m.err != "" {
		b.WriteString(styleWarn().Render(clipWidth(m.err, width)))
		b.WriteByte('\n')
	}
	if len(m.moveFolders) == 0 {
		b.WriteString(styleMuted().Render("no folders"))
		return strings.TrimRight(b.String(), "\n")
	}
	vis := max(1, 16)
	start := m.playlistOffset
	end := min(len(m.moveFolders), start+vis)
	for i := start; i < end; i++ {
		label := m.moveFolders[i]
		selected := m.focus == focusPlaylist && i == m.playlistIdx
		cursor, name := m.listCursorLead(selected, false, clipWidth(label, max(4, width-2)), 1)
		if selected {
			name = styleSelected().Render(name)
		} else {
			name = styleText().Render(name)
		}
		b.WriteString(cursor + name)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) renderArtPicker(width int) string {
	if m.artPickBusy {
		if len(m.artHits) == 0 {
			return styleMuted().Render("searching discogs…")
		}
		return styleMuted().Render("loading covers…")
	}
	var b strings.Builder
	if m.artPickQuery != "" {
		b.WriteString(styleMuted().Render(clipWidth(m.artPickQuery, width)))
		b.WriteByte('\n')
	}
	if len(m.artHits) == 0 {
		b.WriteString(styleMuted().Render("no covers"))
		return strings.TrimRight(b.String(), "\n")
	}
	vis := max(1, 16)
	start := m.playlistOffset
	end := min(len(m.artHits), start+vis)
	for i := start; i < end; i++ {
		row := m.artHits[i]
		label := row.Label
		if y := strings.TrimSpace(row.Year); y != "" {
			label += "  " + y
		}
		selected := m.focus == focusPlaylist && i == m.playlistIdx
		cursor, name := m.listCursorLead(selected, false, clipWidth(label, max(4, width-2)), 1)
		if selected {
			name = styleSelected().Render(name)
		} else {
			name = styleText().Render(name)
		}
		b.WriteString(cursor + name)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) browseLegend(maxW int) string {
	return clipWidth("browse", maxW)
}

func (m model) playlistLegend(maxW int) string {
	if m.tagEditor {
		return clipWidth("edit tags", maxW)
	}
	if m.movePicker {
		return clipWidth("move", maxW)
	}
	if m.artPicker {
		return clipWidth("cover", maxW)
	}
	if m.settingsSelected() {
		return clipWidth("settings", maxW)
	}
	if m.helpSelected() {
		return clipWidth("Help", maxW)
	}
	return clipWidth(fmt.Sprintf("playlist (%d)", len(m.queueFiltered)), maxW)
}

func (m model) browseSubtitle() string {
	if m.search.Value() != "" && m.focus != focusSearch {
		return styleMuted().Render("search: " + m.search.Value())
	}
	if m.browsePath != "" {
		return styleMuted().Render(m.browsePath)
	}
	return ""
}

func (m model) browseVisible(innerH int) int {
	vis := max(1, innerH)
	if m.browseSubtitle() != "" {
		vis = max(1, vis-1)
	}
	return vis
}

func (m model) browseInnerWidth() int {
	return paneInnerWidth(m.playerGeom().browseW)
}

func (m model) artworkLegend() string {
	if y := strings.TrimSpace(m.status.Year); y != "" {
		return y
	}
	return "artwork"
}

func (m model) nowPlayingLegend(maxW int) string {
	return clipWidth("now playing", maxW)
}

func (m model) artDisplayImage() (image.Image, string) {
	if m.artPicker && m.artPreviewImg != nil {
		return m.artPreviewImg, "preview:" + m.artPickPreviewURL
	}
	return m.artImg, m.artPath
}

func (m model) cachedArtwork(cols, rows int) (layout, seq string, overlay bool) {
	img, cacheKey := m.artDisplayImage()
	if m.art == nil {
		return encodeArt(img, cols, rows)
	}
	cw, ch := artCellSize()
	if m.art.path == cacheKey && m.art.cols == cols && m.art.rows == rows && m.art.cellW == cw && m.art.cellH == ch && m.art.layout != "" {
		return m.art.layout, m.art.seq, m.art.overlay
	}
	layout, seq, overlay = encodeArt(img, cols, rows)
	m.art.path = cacheKey
	m.art.cols = cols
	m.art.rows = rows
	m.art.cellW = cw
	m.art.cellH = ch
	m.art.layout = layout
	m.art.seq = seq
	m.art.overlay = overlay
	return layout, seq, overlay
}

func clipWidth(s string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

func clearArtOverlay() string {
	return termimg.ClearAllString()
}

const (
	nowPlayingKHZ     = "48 KHZ"
	nowPlayingLCDMinW = 48
)

var lcdGlyphs = map[rune][3]string{
	'0': {"█▀█", "█ █", "▀▀▀"},
	'1': {"▄█ ", " █ ", " ▀ "},
	'2': {"▀▀█", "█▀▀", "▀▀▀"},
	'3': {"▀▀█", " ▀█", "▀▀▀"},
	'4': {"█ █", "▀▀█", "  ▀"},
	'5': {"█▀▀", "▀▀█", "▀▀▀"},
	'6': {"█▀▀", "█▀█", "▀▀▀"},
	'7': {"▀▀█", "  █", "  ▀"},
	'8': {"█▀█", "█▀█", "▀▀▀"},
	'9': {"█▀█", "▄▀█", " ▀▀"},
	'-': {"   ", "▀▀▀", "   "},
	':': {" ", "▄", "▀"},
}

func (m *model) patchNowPlaying() {
	if m.frames == nil {
		return
	}
	m.frames.mu.Lock()
	row, col, w := m.frames.nowPlayingRow, m.frames.nowPlayingCol, m.frames.nowPlayingW
	m.frames.mu.Unlock()
	if row < 1 || col < 1 || w < 8 {
		return
	}
	out := m.renderNowPlayingChrome(w)
	lines := strings.Split(out, "\n")
	for i := range lines {
		lines[i] = padExact(lines[i], w)
	}
	m.frames.mu.Lock()
	same := strings.Join(lines, "\n") == m.frames.lastNowPlaying
	if !same {
		m.frames.lastNowPlaying = strings.Join(lines, "\n")
	}
	m.frames.mu.Unlock()
	if same {
		return
	}
	paintVizAt(row, col, lines)
}

func patchLines(s string, width, height int) []string {
	width = max(1, width)
	height = max(1, height)
	raw := strings.Split(s, "\n")
	if s == "" {
		raw = nil
	}
	lines := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(raw) {
			lines[i] = padExact(raw[i], width)
		} else {
			lines[i] = strings.Repeat(" ", width)
		}
	}
	return lines
}

func (m *model) patchPane(row, col, width, height int, inner string, last *string) {
	if row < 1 || col < 1 || width < 1 || height < 1 {
		return
	}
	lines := patchLines(inner, width, height)
	out := strings.Join(lines, "\n")
	m.frames.mu.Lock()
	same := out == *last
	if !same {
		*last = out
	}
	m.frames.mu.Unlock()
	if same {
		return
	}
	paintVizAt(row, col, lines)
}

func (m *model) patchBrowse() {
	if m.frames == nil {
		return
	}
	m.frames.mu.Lock()
	row, col, w, h := m.frames.browseRow, m.frames.browseCol, m.frames.browseW, m.frames.browseH
	m.frames.mu.Unlock()
	m.patchPane(row, col, w, h, m.renderBrowseInner(w, h), &m.frames.lastBrowse)
}

func (m *model) freezeAndPatchPlaylist() {
	if m.frames == nil || m.frames.view == "" {
		return
	}
	m.freezeFrame()
	m.patchPlaylist()
}

func (m *model) patchPlaylist() {
	if m.frames == nil {
		return
	}
	m.frames.mu.Lock()
	row, col, w, h := m.frames.playlistRow, m.frames.playlistCol, m.frames.playlistW, m.frames.playlistH
	m.frames.mu.Unlock()
	m.patchPane(row, col, w, h, m.renderPlaylistInner(w, h), &m.frames.lastPlaylist)
}

func (m model) renderNowPlaying(width int) string {
	width = max(24, width)
	rows := m.nowPlayingDisplay(width)
	rows = append(rows, m.nowPlayingButtons(width))
	for i := range rows {
		rows[i] = padExact(rows[i], width)
	}
	return strings.Join(rows, "\n")
}

func (m model) nowPlayingDisplay(innerW int) []string {
	clock := lcdRows(nowPlayingClock(m.status.Path, m.status.Position))

	left := make([]string, 3)
	if innerW >= nowPlayingLCDMinW {
		left[0] = styleGood().Render(clock[0])
		left[1] = styleGood().Render(clock[1])
		left[2] = styleGood().Render(clock[2])
	} else {
		left[0] = styleGood().Render(nowPlayingClock(m.status.Path, m.status.Position))
	}

	leftW := 0
	for _, s := range left {
		if w := lipgloss.Width(s); w > leftW {
			leftW = w
		}
	}
	leftW = min(leftW, max(8, innerW/2))
	status := m.nowPlayingStatusLines()
	statusW := 0
	for _, s := range status {
		if w := lipgloss.Width(s); w > statusW {
			statusW = w
		}
	}
	const colGaps = 4
	midW := max(8, innerW-leftW-statusW-colGaps)

	right := []string{
		m.nowPlayingMarqueeRow(midW),
		m.nowPlayingRelease(midW),
		m.nowPlayingMeters(midW),
	}

	out := make([]string, 3)
	for i := 0; i < 3; i++ {
		statusCell := ""
		if statusW > 0 {
			statusCell = "  " + padExact(status[i], statusW)
		}
		out[i] = padRight(left[i], leftW) + "  " + padExact(right[i], midW) + statusCell
	}
	return out
}

func (m model) nowPlayingMarquee() string {
	if m.status.Path == "" {
		return "NOTHING PLAYING   ***"
	}
	title := strings.TrimSpace(m.status.Title)
	if title == "" {
		title = m.status.Path
	}
	artist := strings.TrimSpace(m.status.Artist)
	msg := title
	if artist != "" {
		msg = artist + " - " + title
	}
	return nowPlayingUpper(msg) + "   ***"
}

func (m model) nowPlayingMarqueeRow(width int) string {
	return padExact(styleText().Render(clipEllipsis(m.nowPlayingMarquee(), width)), width)
}

func (m model) nowPlayingRelease(width int) string {
	release := nowPlayingUpper(m.nowPlayingReleaseLine())
	if release == "" {
		return padExact("", width)
	}
	return padExact(styleText().Render(clipEllipsis(release, width)), width)
}

func (m model) nowPlayingReleaseLine() string {
	year := strings.TrimSpace(m.status.Year)
	label := strings.TrimSpace(m.status.Label)
	cat := strings.TrimSpace(m.status.Album)
	if cat != "" && !releaseLooksLikeCatno(cat) {
		cat = ""
	}
	var parts []string
	if year != "" {
		parts = append(parts, year)
	}
	if label != "" {
		parts = append(parts, label)
	}
	if cat != "" && !strings.EqualFold(cat, label) {
		parts = append(parts, cat)
	}
	return strings.Join(parts, "  ")
}

func releaseLooksLikeCatno(s string) bool {
	if len(s) < 3 || strings.ContainsAny(s, " \t") {
		return false
	}
	hasDigit := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			hasDigit = true
			break
		}
	}
	return hasDigit
}

func (m model) nowPlayingMeters(width int) string {
	if width < 8 {
		return ""
	}
	const heartW = 2 // space + heart
	volW := min(12, max(6, width-8-heartW))
	panW := min(7, max(0, width-volW-1-heartW))
	vol := nowPlayingVolume(m.status.Volume, volW)
	if panW >= 5 {
		return padExact(vol+" "+nowPlayingPan(panW)+" "+nowPlayingHeart(m.status.Liked), width)
	}
	return padExact(vol+" "+nowPlayingHeart(m.status.Liked), width)
}

func (m model) nowPlayingStatusLines() [3]string {
	line1 := styleMuted().Render("MONO") + " " + styleGood().Render("STEREO")
	line2 := styleText().Render(nowPlayingKHZ)
	line3 := nowPlayingToggle("EQ", m.vizMode != vizModeNone) + " " + nowPlayingToggle("PL", true)
	w := max(lipgloss.Width(line1), lipgloss.Width(line2), lipgloss.Width(line3))
	left := lipgloss.NewStyle().Width(w).Align(lipgloss.Left)
	right := lipgloss.NewStyle().Width(w).Align(lipgloss.Right)
	return [3]string{
		left.Render(line1),
		right.Render(line2),
		right.Render(line3),
	}
}

func nowPlayingTransportBtn(glyph string, active bool) string {
	if active {
		return styleGood().Render(glyph)
	}
	return styleMuted().Render(glyph)
}

func (m model) nowPlayingButtons(width int) string {
	prev := nowPlayingTransportBtn("⏮", false)
	play := nowPlayingTransportBtn("▶", false)
	pause := nowPlayingTransportBtn("⏸", false)
	stop := nowPlayingTransportBtn("⏹", false)
	next := nowPlayingTransportBtn("⏭", false)
	eject := nowPlayingTransportBtn("⏏", false)
	switch m.status.State {
	case "playing":
		play = styleGood().Render("▶")
	case "paused":
		pause = styleGood().Render("⏸")
	}
	keys := prev + " " + play + " " + pause + " " + stop + " " + next + " " + eject
	shuffle := nowPlayingToggle("SHUFFLE", m.status.Shuffle)
	rep := nowPlayingToggle("REP", false)
	toggles := shuffle + "  " + rep
	return padExact(keys+" "+toggles, width)
}

func nowPlayingClock(path string, pos float64) string {
	if path == "" {
		return "00:00"
	}
	total := int(math.Max(0, math.Floor(pos)))
	min := total / 60
	sec := total % 60
	if min > 99 {
		return fmt.Sprintf("%d:%02d", min, sec)
	}
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func lcdRows(s string) [3]string {
	var rows [3]strings.Builder
	for i, r := range s {
		g, ok := lcdGlyphs[r]
		if !ok {
			g = [3]string{" ", " ", " "}
		}
		if i > 0 {
			for j := 0; j < 3; j++ {
				rows[j].WriteByte(' ')
			}
		}
		for j := 0; j < 3; j++ {
			rows[j].WriteString(g[j])
		}
	}
	return [3]string{rows[0].String(), rows[1].String(), rows[2].String()}
}

func nowPlayingVolume(pct, width int) string {
	if width < 2 {
		return ""
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	fill := int(math.Round(float64(pct) / 100 * float64(width)))
	if fill > width {
		fill = width
	}
	return styleGood().Render(strings.Repeat("━", fill)) + styleMuted().Render(strings.Repeat("━", width-fill))
}

func nowPlayingPan(width int) string {
	if width < 5 {
		return ""
	}
	return nowPlayingDotSlider(width/2, width)
}

func nowPlayingHeart(liked bool) string {
	if liked {
		return styleLiked().Render("♥")
	}
	return " "
}

func nowPlayingDotSlider(x, width int) string {
	if width < 1 {
		return ""
	}
	if x < 0 {
		x = 0
	}
	if x > width-1 {
		x = width - 1
	}
	left := styleMuted().Render(strings.Repeat("─", x))
	thumb := styleText().Render("•")
	right := styleMuted().Render(strings.Repeat("─", width-1-x))
	return padExact(left+thumb+right, width)
}

func nowPlayingToggle(label string, on bool) string {
	if on {
		return styleGood().Render(label)
	}
	return styleMuted().Render(label)
}

func nowPlayingUpper(s string) string {
	return strings.Map(func(r rune) rune {
		return unicode.ToUpper(r)
	}, s)
}
