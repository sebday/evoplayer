package tui

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	vizPaintRows    = 5
	vizTickInterval = 50 * time.Millisecond
	peakHoldFrames  = 6
	peakFall        = 0.035
	waveIdleFloor   = 0.12
)

type waveVizOpts struct {
	Width      int
	Height     int
	Peaks      []int
	PlayedFrac float64
}

func downsamplePeaks(peaks []int, width int) []float64 {
	if width < 1 {
		return nil
	}
	out := make([]float64, width)
	if len(peaks) == 0 {
		return out
	}
	if len(peaks) == width {
		for i, v := range peaks {
			out[i] = float64(v)
		}
		return out
	}
	if len(peaks) > width {
		for i := 0; i < width; i++ {
			start := i * len(peaks) / width
			end := (i + 1) * len(peaks) / width
			if end <= start {
				end = start + 1
			}
			if end > len(peaks) {
				end = len(peaks)
			}
			mx, sum := 0, 0
			for _, v := range peaks[start:end] {
				if v > mx {
					mx = v
				}
				sum += v
			}
			n := float64(end - start)
			out[i] = float64(mx)*0.48 + float64(sum)/n*0.52
		}
		return out
	}
	last := len(peaks) - 1
	span := float64(max(1, width-1))
	for i := 0; i < width; i++ {
		pos := float64(i) * float64(last) / span
		i0 := int(pos)
		i1 := i0 + 1
		if i1 > last {
			i1 = last
		}
		t := pos - float64(i0)
		out[i] = float64(peaks[i0])*(1-t) + float64(peaks[i1])*t
	}
	return out
}

func normalizeAmps(cols []float64) []float64 {
	n := len(cols)
	if n == 0 {
		return cols
	}
	sorted := append([]float64(nil), cols...)
	sort.Float64s(sorted)
	if sorted[n-1] <= 1 {
		return cols
	}
	span := sorted[n-1] - sorted[0]
	if span < 1 {
		out := make([]float64, n)
		for i := range out {
			out[i] = 0.28
		}
		return out
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		if cols[idx[a]] == cols[idx[b]] {
			return idx[a] < idx[b]
		}
		return cols[idx[a]] < cols[idx[b]]
	})
	rank := make([]float64, n)
	den := float64(max(1, n-1))
	for r, i := range idx {
		rank[i] = float64(r) / den
	}
	const halfWin = 2
	out := make([]float64, n)
	for i, v := range cols {
		lo, hi := v, v
		for k := max(0, i-halfWin); k <= min(n-1, i+halfWin); k++ {
			if cols[k] < lo {
				lo = cols[k]
			}
			if cols[k] > hi {
				hi = cols[k]
			}
		}
		local := 0.5
		if hi-lo > 0.5 {
			local = (v - lo) / (hi - lo)
		}
		out[i] = math.Pow(clamp01(0.55*rank[i]+0.45*local), 1.35)
	}
	return out
}

const brailleBase = 0x2800

var brailleDots = [4][2]int{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

func renderWaveformBars(opts waveVizOpts) string {
	width := max(1, opts.Width)
	height := max(vizPaintRows, opts.Height)
	playX := wavePlayCol(opts.PlayedFrac, width)
	rows := waveformGlyphRows(opts.Peaks, width, height)
	return strings.Join(colorizeWaveRows(rows, playX), "\n")
}

func renderWaveformOverlay(opts waveVizOpts, levels, peaks []float64, playing bool) string {
	width := max(1, opts.Width)
	height := max(vizPaintRows, opts.Height)
	playX := wavePlayCol(opts.PlayedFrac, width)
	fill := waveformFill(opts.Peaks, width, height)
	var live []float64
	if playing {
		live = liveOverlayFill(fill, levels, peaks, waveMaxHalf(height), len(opts.Peaks) > 0)
	}
	if !hasLiveFill(live) {
		return strings.Join(colorizeWaveRows(waveformGlyphRowsFromFill(fill, height), playX), "\n")
	}
	return strings.Join(renderOverlayRows(fill, live, width, height, playX), "\n")
}

func waveMaxHalf(height int) float64 {
	height = max(vizPaintRows, height)
	center := float64(height*4-1) / 2
	if center < 1 {
		return 1
	}
	return center
}

func waveformFill(peaks []int, width, height int) []float64 {
	width = max(1, width)
	height = max(vizPaintRows, height)
	cols := normalizeAmps(downsamplePeaks(peaks, width))
	maxHalf := waveMaxHalf(height)
	fill := make([]float64, width)
	for x := 0; x < width; x++ {
		lv := 0.0
		if x < len(cols) {
			lv = cols[x]
		}
		fill[x] = (waveIdleFloor + clamp01(lv)*(1-waveIdleFloor)) * maxHalf
	}
	return fill
}

func waveformGlyphRows(peaks []int, width, height int) []string {
	height = max(vizPaintRows, height)
	return waveformGlyphRowsFromFill(waveformFill(peaks, width, height), height)
}

func waveformGlyphRowsFromFill(fill []float64, height int) []string {
	width := max(1, len(fill))
	height = max(vizPaintRows, height)
	center := float64(height*4-1) / 2
	rows := make([]string, height)
	for r := 0; r < height; r++ {
		cells := make([]rune, width)
		for x := 0; x < width; x++ {
			v := 0.0
			if x < len(fill) {
				v = fill[x]
			}
			cells[x] = waveBrailleRune(v, r, center)
		}
		rows[r] = string(cells)
	}
	return rows
}

func downsampleLevels(levels []float64, width int) []float64 {
	if width < 1 || len(levels) == 0 {
		return nil
	}
	out := make([]float64, width)
	for i := 0; i < width; i++ {
		start := i * len(levels) / width
		end := (i + 1) * len(levels) / width
		if end <= start {
			end = start + 1
		}
		if end > len(levels) {
			end = len(levels)
		}
		mx := 0.0
		for _, v := range levels[start:end] {
			if v > mx {
				mx = v
			}
		}
		out[i] = mx
	}
	return out
}

func updateVizPeaks(peaks []float64, hold []int, levels []float64) ([]float64, []int) {
	n := len(levels)
	if len(peaks) != n {
		peaks = make([]float64, n)
		hold = make([]int, n)
	}
	for i, v := range levels {
		if v >= peaks[i] {
			peaks[i] = v
			hold[i] = peakHoldFrames
			continue
		}
		if hold[i] > 0 {
			hold[i]--
			continue
		}
		peaks[i] -= peakFall
		if peaks[i] < v {
			peaks[i] = v
		}
	}
	return peaks, hold
}

func liveOverlayFill(waveFill, levels, peaks []float64, maxHalf float64, clip bool) []float64 {
	width := len(waveFill)
	if width < 1 {
		return nil
	}
	cols := downsampleLevels(levels, width)
	peakCols := downsampleLevels(peaks, width)
	out := make([]float64, width)
	for x := 0; x < width; x++ {
		lv := 0.0
		if x < len(cols) {
			lv = cols[x]
		}
		if x < len(peakCols) && peakCols[x] > lv {
			lv = peakCols[x]
		}
		lv = math.Pow(clamp01(lv), 0.85)
		if lv < 0.04 {
			continue
		}
		env := maxHalf
		if clip && x < len(waveFill) {
			env = waveFill[x]
		}
		out[x] = env * lv
	}
	return out
}

func hasLiveFill(fill []float64) bool {
	for _, v := range fill {
		if v > 0 {
			return true
		}
	}
	return false
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func paletteFG(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 15 {
		n = 15
	}
	if n >= 8 {
		return fmt.Sprintf("\x1b[%dm", 90+n-8)
	}
	return fmt.Sprintf("\x1b[%dm", 30+n)
}

func vuPalette(t float64) int {
	t = clamp01(t)
	switch {
	case t < 0.22:
		return 2
	case t < 0.40:
		return 10
	case t < 0.58:
		return 3
	case t < 0.78:
		return 11
	case t < 0.92:
		return 1
	default:
		return 9
	}
}

func overlayRowPrefixes(height int) []string {
	height = max(vizPaintRows, height)
	center := float64(height*4-1) / 2
	out := make([]string, height)
	for r := 0; r < height; r++ {
		mid := float64(r*4) + 1.5
		t := 0.0
		if center > 0 {
			t = math.Abs(mid-center) / center
		}
		out[r] = paletteFG(vuPalette(t))
	}
	return out
}

var overlayPrefixesCached = overlayRowPrefixes(vizPaintRows)

func renderOverlayRows(waveFill, liveFill []float64, width, height, playX int) []string {
	width = max(1, width)
	height = max(vizPaintRows, height)
	center := float64(height*4-1) / 2
	prefixes := overlayPrefixesCached
	if height != vizPaintRows {
		prefixes = overlayRowPrefixes(height)
	}
	rows := make([]string, height)
	for r := 0; r < height; r++ {
		var b strings.Builder
		last := -2
		for x := 0; x < width; x++ {
			tag := -1
			prefix := ""
			ch := ' '
			if x == playX {
				prefix = waveWhitePrefix
				tag = 1
				ch = wavePlayhead
			} else {
				wf := 0.0
				lf := 0.0
				if x < len(waveFill) {
					wf = waveFill[x]
				}
				if x < len(liveFill) {
					lf = liveFill[x]
				}
				waveBits := waveBrailleBits(wf, r, center)
				liveBits := waveBrailleBits(lf, r, center)
				if liveBits != 0 {
					prefix = prefixes[r]
					tag = 2
					ch = rune(brailleBase + (waveBits | liveBits))
				} else if waveBits != 0 {
					prefix = waveMutedPrefix
					tag = 0
					ch = rune(brailleBase + waveBits)
				}
			}
			if tag != last {
				if last >= 0 {
					b.WriteString(waveReset)
				}
				if prefix != "" {
					b.WriteString(prefix)
				}
				last = tag
			}
			b.WriteRune(ch)
		}
		if last >= 0 {
			b.WriteString(waveReset)
		}
		rows[r] = b.String()
	}
	return rows
}

func wavePlayCol(frac float64, width int) int {
	if frac <= 0 || width <= 0 {
		return -1
	}
	if frac >= 1 {
		return width - 1
	}
	n := int(math.Round(frac * float64(width-1)))
	if n < 0 {
		return 0
	}
	if n >= width {
		return width - 1
	}
	return n
}

const wavePlayhead = '│'

var (
	waveWhitePrefix = paletteFG(15)
	waveMutedPrefix = paletteFG(8)
	waveReset       = "\x1b[0m"
	vizOutMu        sync.Mutex
)

func colorizeWaveRows(rows []string, playX int) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = colorizeWaveRow(row, playX)
	}
	return out
}

func colorizeWaveRow(row string, playX int) string {
	cols := []rune(row)
	n := len(cols)
	if n == 0 {
		return ""
	}
	var b strings.Builder
	if playX < 0 || playX >= n {
		b.WriteString(waveMutedPrefix)
		b.WriteString(row)
		b.WriteString(waveReset)
		return b.String()
	}
	if playX > 0 {
		b.WriteString(waveMutedPrefix)
		b.WriteString(string(cols[:playX]))
		b.WriteString(waveReset)
	}
	b.WriteString(waveWhitePrefix)
	b.WriteRune(wavePlayhead)
	b.WriteString(waveReset)
	if playX+1 < n {
		b.WriteString(waveMutedPrefix)
		b.WriteString(string(cols[playX+1:]))
		b.WriteString(waveReset)
	}
	return b.String()
}

func waveformPlayedFrac(position, duration float64) float64 {
	if duration <= 0 {
		return 0
	}
	f := position / duration
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func waveBrailleBits(fill float64, row int, center float64) int {
	bits := 0
	for i := 0; i < 4; i++ {
		dr := float64(row*4 + i)
		if math.Abs(dr-center) <= fill {
			bits |= brailleDots[i][0]
			bits |= brailleDots[i][1]
		}
	}
	return bits
}

func waveBrailleRune(fill float64, row int, center float64) rune {
	bits := waveBrailleBits(fill, row, center)
	if bits == 0 {
		return ' '
	}
	return rune(brailleBase + bits)
}

func (m model) ensureWaveFill(width, height int) []float64 {
	width = max(1, width)
	height = max(vizPaintRows, height)
	if m.frames != nil {
		m.frames.mu.Lock()
		hit := m.frames.waveW == width &&
			m.frames.waveH == height &&
			m.frames.waveN == len(m.wavePeaks) &&
			m.frames.wavePath == m.wavePath &&
			len(m.frames.waveFill) == width
		if hit {
			fill := m.frames.waveFill
			m.frames.mu.Unlock()
			return fill
		}
		m.frames.mu.Unlock()
	}
	fill := waveformFill(m.wavePeaks, width, height)
	rows := waveformGlyphRowsFromFill(fill, height)
	if m.frames != nil {
		m.frames.mu.Lock()
		m.frames.waveW = width
		m.frames.waveH = height
		m.frames.waveN = len(m.wavePeaks)
		m.frames.wavePath = m.wavePath
		m.frames.waveFill = fill
		m.frames.waveRows = rows
		m.frames.playedTo = -1
		m.frames.mu.Unlock()
	}
	return fill
}

func (m model) ensureWaveRows(width, height int) []string {
	width = max(1, width)
	height = max(vizPaintRows, height)
	m.ensureWaveFill(width, height)
	if m.frames != nil && len(m.frames.waveRows) == height {
		return m.frames.waveRows
	}
	return waveformGlyphRows(m.wavePeaks, width, height)
}

func (m model) overlayWaveRows(width, height, playedTo int) []string {
	fill := m.ensureWaveFill(width, height)
	var live []float64
	if m.playing() {
		live = liveOverlayFill(fill, m.vizLevels, m.vizPeaks, waveMaxHalf(height), len(m.wavePeaks) > 0)
	}
	if !hasLiveFill(live) {
		return colorizeWaveRows(m.ensureWaveRows(width, height), playedTo)
	}
	return renderOverlayRows(fill, live, width, height, playedTo)
}

func (m model) vizInnerLines() []string {
	if m.frames == nil || m.frames.vizW < 8 {
		return nil
	}
	innerW := m.frames.vizW
	waveW := max(8, innerW-2)
	playX := wavePlayCol(waveformPlayedFrac(m.status.Position, m.status.Duration), waveW)
	m.ensureWaveFill(waveW, vizPaintRows)
	return composeVizLines(m.frames, m.vizLevels, m.vizPeaks, playX, len(m.wavePeaks) > 0)
}

func composeVizLines(frames *frameCache, levels, livePeaks []float64, playX int, hasWave bool) []string {
	if frames == nil {
		return nil
	}
	frames.mu.Lock()
	innerW := frames.vizW
	fill := frames.waveFill
	rows := frames.waveRows
	frames.playedTo = playX
	frames.mu.Unlock()
	if innerW < 8 {
		return nil
	}
	waveW := max(8, innerW-2)
	live := liveOverlayFill(fill, levels, livePeaks, waveMaxHalf(vizPaintRows), hasWave)
	var glyphs []string
	if !hasLiveFill(live) {
		glyphs = colorizeWaveRows(rows, playX)
	} else {
		glyphs = renderOverlayRows(fill, live, waveW, vizPaintRows, playX)
	}
	return padVizInner(glyphs, innerW)
}

func padVizInner(rows []string, innerW int) []string {
	lines := make([]string, vizPaintRows)
	for i := 0; i < vizPaintRows; i++ {
		s := ""
		if i < len(rows) {
			s = rows[i]
		}
		lines[i] = padExact(" "+s, innerW)
	}
	return lines
}

func paintVizAt(row, col int, lines []string) {
	if row < 1 || col < 1 || len(lines) == 0 {
		return
	}
	var b strings.Builder
	b.WriteString("\x1b[s")
	for i, line := range lines {
		fmt.Fprintf(&b, "\x1b[%d;%dH%s", row+i, col, line)
	}
	b.WriteString("\x1b[u")
	vizOutMu.Lock()
	_, _ = os.Stdout.WriteString(b.String())
	vizOutMu.Unlock()
}
