package tui

import (
	"strings"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/lipgloss"
)

type artPlacement struct {
	seq string
	row int
	col int
}

func (m model) renderListWithArt(height int) (string, artPlacement) {
	g := m.splitArtGeom(height)
	innerH := max(1, g.artPanelH-4)
	legendW := max(8, g.leftW-6)
	legend := m.paneLegend()
	if m.currentSelected() {
		legend = m.currentLegend(legendW)
	}
	return m.renderSplitArt(height, legend, m.renderPaneInner(innerH))
}

type splitArtGeom struct {
	totalW               int
	artCols, artRows     int
	artPanelW, artPanelH int
	leftW                int
}

func (m model) splitArtGeom(height int) splitArtGeom {
	totalW := m.mainWidth()
	artCols, artRows := squareArtSize(totalW, height)
	artPanelW := artCols + 4
	leftW := max(24, totalW-artPanelW-1)
	return splitArtGeom{
		totalW:    totalW,
		artCols:   artCols,
		artRows:   artRows,
		artPanelW: artPanelW,
		artPanelH: height,
		leftW:     leftW,
	}
}

func (m model) renderSplitArt(height int, legend, leftBody string) (string, artPlacement) {
	g := m.splitArtGeom(height)
	layout, seq, overlay := m.cachedArt(g.artCols, g.artRows)
	pulse := -1.0
	if !overlay {
		pulse = m.pulsePhase
	}
	leftPadY := 1
	left := fieldsetPad(legend, leftBody, g.leftW, g.artPanelH, m.focus == focusList, pulse, leftPadY, 1, "", "", 2)
	right := fieldsetPad(m.artLegend(), layout, g.artPanelW, g.artPanelH, false, 0, 0, 1, "", "", 3)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)

	place := artPlacement{}
	if overlay {
		place.seq = seq
		place.row = 2
		place.col = lipgloss.Width(left) + 4
	}
	return joined, place
}

// squareArtSize returns cell cols/rows for a pixel-square cover.
func squareArtSize(totalW, paneH int) (cols, rows int) {
	cw, ch := artCellSize()
	return squareArtSizeForCells(totalW, paneH, cw, ch)
}

func squareArtSizeForCells(totalW, paneH, cw, ch int) (cols, rows int) {
	const (
		minLeft = 36
		gap     = 1
		chromeW = 4
		minCols = 8
		minRows = 4
	)
	if cw < 1 {
		cw = 8
	}
	if ch < 1 {
		ch = 16
	}
	availH := max(minRows, paneH-2)
	maxCols := totalW - minLeft - gap - chromeW
	if maxCols < minCols {
		maxCols = minCols
	}
	rows = availH
	cols = (rows*ch + cw/2) / cw
	if cols < minCols {
		cols = minCols
	}
	if cols > maxCols {
		cols = maxCols
	}
	return cols, rows
}

func (m model) listInnerWidth() int {
	return max(8, m.splitArtGeom(m.paneHeight()).leftW-4)
}

func (m model) renderNowPlayingHead(width int) string {
	if m.status.Path == "" {
		return ""
	}
	sub := m.nowPlayingSubtitle()
	if sub == "" {
		return ""
	}
	return styleMuted().Render(clipWidth(sub, width))
}

func (m model) nowPlayingHeadRows() int {
	rows := 1 // nowPlayingTitleRow sits under the fieldset legend
	head := m.renderNowPlayingHead(m.listInnerWidth())
	if head == "" {
		return rows
	}
	return rows + lipgloss.Height(head) + 1
}

func (m model) artLegend() string {
	if y := strings.TrimSpace(m.status.Year); y != "" {
		return y
	}
	return "Artwork"
}

func (m model) currentLegend(maxW int) string {
	return clipWidth("Now playing", maxW)
}

func (m model) nowPlayingTitleRow(width int) string {
	if m.status.Path == "" {
		return styleMuted().Render(clipWidth("nothing playing", width))
	}
	title := strings.TrimSpace(m.status.Title)
	if title == "" {
		title = m.status.Path
	}
	pills := m.renderNowPills(width)
	if pills == "" {
		return styleText().Render(clipWidth(title, width))
	}
	gap := 1
	pillW := lipgloss.Width(pills)
	titleMax := width - pillW - gap
	if titleMax < 8 {
		pills = m.renderNowPills(max(8, width/2))
		pillW = lipgloss.Width(pills)
		titleMax = width - pillW - gap
		if titleMax < 4 {
			titleMax = 4
		}
	}
	left := padRight(styleText().Render(clipWidth(title, titleMax)), titleMax)
	return padExact(left+" "+pills, width)
}

func (m model) nowPlayingSubtitle() string {
	artist := strings.TrimSpace(m.status.Artist)
	album := strings.TrimSpace(m.status.Album)
	switch {
	case artist != "" && album != "":
		return artist + " — " + album
	case artist != "":
		return artist
	default:
		return album
	}
}

func (m model) renderNowPills(width int) string {
	vals := make([]string, 0, 4)
	if l := strings.TrimSpace(m.status.Label); l != "" {
		vals = append(vals, l)
	}
	if g := strings.TrimSpace(m.status.Genre); g != "" {
		vals = append(vals, g)
	}
	if y := strings.TrimSpace(m.status.Year); y != "" {
		vals = append(vals, y)
	}
	if d := playbackLabel(m.status.DurationLabel, m.status.Duration); d != "" && d != "0:00" {
		vals = append(vals, d)
	}
	if len(vals) == 0 {
		return ""
	}
	pills := make([]string, 0, len(vals))
	used := 0
	for _, v := range vals {
		p := stylePill().Render(v)
		w := lipgloss.Width(p)
		if used > 0 {
			w++
		}
		if used+w > width {
			break
		}
		pills = append(pills, p)
		used += w
	}
	return strings.Join(pills, " ")
}

func (m model) cachedArt(cols, rows int) (layout, seq string, overlay bool) {
	if m.art == nil {
		return encodeArt(m.artImg, cols, rows)
	}
	cw, ch := artCellSize()
	if m.art.path == m.artPath && m.art.cols == cols && m.art.rows == rows && m.art.cellW == cw && m.art.cellH == ch && m.art.layout != "" {
		return m.art.layout, m.art.seq, m.art.overlay
	}
	layout, seq, overlay = encodeArt(m.artImg, cols, rows)
	m.art.path = m.artPath
	m.art.cols = cols
	m.art.rows = rows
	m.art.cellW = cw
	m.art.cellH = ch
	m.art.layout = layout
	m.art.seq = seq
	m.art.overlay = overlay
	return layout, seq, overlay
}

func playbackLabel(label string, sec float64) string {
	if label != "" {
		return label
	}
	return dur(sec)
}

func clipWidth(s string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

func clearArtOverlay() string {
	return termimg.ClearAllString()
}
