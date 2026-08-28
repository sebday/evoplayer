package tui

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	liveVizEnabled  = true
	vizPaintRows    = 5
	vizTickInterval = 50 * time.Millisecond
	peakHoldFrames  = 6
	peakFall        = 0.035
	waveIdleFloor   = 0.12
	liveBarGap      = 1
	liveBarWidth    = 1
)

type vizMode int

const (
	vizModeBars vizMode = iota
	vizModeFill
	vizModeScatter
	vizModeWave
	vizModeNone
	vizModeCount
)

func (m vizMode) String() string {
	switch m {
	case vizModeFill:
		return "fill"
	case vizModeScatter:
		return "scatter"
	case vizModeWave:
		return "wave"
	case vizModeNone:
		return "none"
	default:
		return "bars"
	}
}

func (m vizMode) next() vizMode {
	return (m + 1) % vizModeCount
}

func (m vizMode) prev() vizMode {
	return (m + vizModeCount - 1) % vizModeCount
}

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
	var live, peak []float64
	if playing {
		live, peak = liveOverlayLayers(fill, levels, peaks, waveMaxHalf(height), len(opts.Peaks) > 0, vizModeBars)
	}
	if !hasLiveFill(live) && !hasLiveFill(peak) {
		return strings.Join(colorizeWaveRows(waveformGlyphRowsFromFill(fill, height), playX), "\n")
	}
	return strings.Join(renderOverlayRows(fill, live, peak, width, height, playX, vizModeBars, 0), "\n")
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

func liveBarColumn(x int, mode vizMode) bool {
	if mode == vizModeFill || mode == vizModeScatter || mode == vizModeWave {
		return true
	}
	stride := liveBarWidth + liveBarGap
	if stride < 1 {
		return true
	}
	return x%stride < liveBarWidth
}

func liveOverlayLayers(waveFill, levels, peaks []float64, maxHalf float64, clip bool, mode vizMode) (bar, peak []float64) {
	width := len(waveFill)
	if width < 1 || mode == vizModeNone {
		return nil, nil
	}
	cols := downsampleLevels(levels, width)
	peakCols := downsampleLevels(peaks, width)
	bar = make([]float64, width)
	peak = make([]float64, width)
	for x := 0; x < width; x++ {
		if !liveBarColumn(x, mode) {
			continue
		}
		lv := 0.0
		if x < len(cols) {
			lv = cols[x]
		}
		pk := lv
		if x < len(peakCols) {
			pk = peakCols[x]
		}
		lv = math.Pow(clamp01(lv), 0.85)
		pk = math.Pow(clamp01(pk), 0.85)
		env := maxHalf
		if clip && x < len(waveFill) {
			env = waveFill[x]
		}
		if lv >= 0.04 {
			bar[x] = env * lv
		}
		if pk >= 0.04 && mode != vizModeWave && mode != vizModeScatter {
			peak[x] = env * pk
		}
	}
	return bar, peak
}

func liveOverlayFill(waveFill, levels, peaks []float64, maxHalf float64, clip bool) []float64 {
	bar, _ := liveOverlayLayers(waveFill, levels, peaks, maxHalf, clip, vizModeBars)
	return bar
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
		return fmt.Sprintf("\x1b[%d;49m", 90+n-8)
	}
	return fmt.Sprintf("\x1b[%d;49m", 30+n)
}

func vuPalette(t float64) int {
	switch specTag(clamp01(t)) {
	case 2:
		return 1
	case 1:
		return 3
	default:
		return 2
	}
}

func specTag(norm float64) int {
	if norm >= 0.6 {
		return 2
	}
	if norm >= 0.3 {
		return 1
	}
	return 0
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

func renderOverlayRows(waveFill, liveFill, peakFill []float64, width, height, playX int, mode vizMode, salt uint64) []string {
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
				wf, lf, pf := 0.0, 0.0, 0.0
				if x < len(waveFill) {
					wf = waveFill[x]
				}
				if x < len(liveFill) {
					lf = liveFill[x]
				}
				if x < len(peakFill) {
					pf = peakFill[x]
				}
				waveBits := waveBrailleBits(wf, r, center)
				liveBits := overlayLiveBits(mode, lf, x, r, center, salt)
				peakBits := 0
				if mode == vizModeBars && pf > lf+0.45 {
					peakBits = waveBraillePeakBits(pf, r, center)
				}
				if liveBits != 0 || peakBits != 0 {
					prefix = prefixes[r]
					tag = 2
					ch = rune(brailleBase + (waveBits | liveBits | peakBits))
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

func overlayLiveBits(mode vizMode, fill float64, x, row int, center float64, salt uint64) int {
	switch mode {
	case vizModeNone:
		return 0
	case vizModeWave:
		return waveBraillePeakBits(fill, row, center)
	case vizModeScatter:
		return scatterBrailleBits(fill, x, row, center, salt)
	default:
		return waveBrailleBits(fill, row, center)
	}
}

func scatterBrailleBits(fill float64, x, row int, center float64, salt uint64) int {
	bits := 0
	for i := 0; i < 4; i++ {
		dr := float64(row*4 + i)
		if math.Abs(dr-center) > fill {
			continue
		}
		if scatterHash(x, row, i, salt) < 0.38 {
			bits |= brailleDots[i][0]
			bits |= brailleDots[i][1]
		}
	}
	return bits
}

func scatterHash(x, row, dot int, salt uint64) float64 {
	h := uint64(x)*7919 + uint64(row)*6271 + uint64(dot)*3037 + salt*104729
	h ^= h >> 16
	h *= 0x45d9f3b37197344b
	h ^= h >> 16
	return float64(h%10000) / 10000.0
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
	waveReset       = "\x1b[39m"
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

func waveBraillePeakBits(fill float64, row int, center float64) int {
	bits := 0
	for i := 0; i < 4; i++ {
		dr := float64(row*4 + i)
		if math.Abs(math.Abs(dr-center)-fill) <= 0.51 {
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
	mode := m.vizMode
	if mode == vizModeNone || !m.playing() {
		return colorizeWaveRows(m.ensureWaveRows(width, height), playedTo)
	}
	live, peak := liveOverlayLayers(fill, m.vizLevels, m.vizPeaks, waveMaxHalf(height), len(m.wavePeaks) > 0, mode)
	if !hasLiveFill(live) && !hasLiveFill(peak) {
		return colorizeWaveRows(m.ensureWaveRows(width, height), playedTo)
	}
	return renderOverlayRows(fill, live, peak, width, height, playedTo, mode, 0)
}

func (m model) vizInnerLines() []string {
	if m.frames == nil || m.frames.vizW < 8 {
		return nil
	}
	innerW := m.frames.vizW
	waveW := max(8, innerW-2)
	playX := wavePlayCol(waveformPlayedFrac(m.status.Position, m.status.Duration), waveW)
	m.ensureWaveFill(waveW, vizPaintRows)
	return composeVizLines(m.frames, m.vizLevels, m.vizPeaks, playX, len(m.wavePeaks) > 0, m.vizMode, 0)
}

func composeVizLines(frames *frameCache, levels, livePeaks []float64, playX int, hasWave bool, mode vizMode, salt uint64) []string {
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
	if mode == vizModeNone {
		return padVizInner(colorizeWaveRows(rows, playX), innerW)
	}
	live, peak := liveOverlayLayers(fill, levels, livePeaks, waveMaxHalf(vizPaintRows), hasWave, mode)
	var glyphs []string
	if !hasLiveFill(live) && !hasLiveFill(peak) {
		glyphs = colorizeWaveRows(rows, playX)
	} else {
		glyphs = renderOverlayRows(fill, live, peak, waveW, vizPaintRows, playX, mode, salt)
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
		if s == "" || s[0] != ' ' {
			s = " " + s
		}
		w := lipgloss.Width(s)
		if w < innerW {
			s += strings.Repeat(" ", innerW-w)
		}
		lines[i] = s
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
