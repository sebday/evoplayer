package tui

import (
	"strings"
	"testing"
)

func TestDownsamplePeaksWidth(t *testing.T) {
	peaks := make([]int, 100)
	for i := range peaks {
		peaks[i] = i
	}
	got := downsamplePeaks(peaks, 10)
	if len(got) != 10 {
		t.Fatalf("len = %d", len(got))
	}
	if got[9] <= got[0] {
		t.Fatalf("expected rising mix pool, got %v", got)
	}
}

func TestWaveformPlayheadLine(t *testing.T) {
	peaks := make([]int, 8)
	for i := range peaks {
		peaks[i] = 255
	}
	got := renderWaveformBars(waveVizOpts{
		Width:      8,
		Height:     5,
		Peaks:      peaks,
		PlayedFrac: 0.5,
	})
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	if strings.Contains(got, "38;2;") {
		t.Fatalf("waveform should use terminal palette, got %q", got)
	}
	if !strings.Contains(got, "[90") {
		t.Fatalf("waveform should stay muted, got %q", got)
	}
	if strings.Count(got, "[97") != 5 {
		t.Fatalf("playhead should be one white run per row, got %d", strings.Count(got, "[97"))
	}
	plain := strings.Split(lipglossStrip(got), "\n")
	for i, line := range plain {
		cols := []rune(line)
		if len(cols) != 8 {
			t.Fatalf("row %d width = %d", i, len(cols))
		}
		if cols[4] != wavePlayhead {
			t.Fatalf("row %d playhead at %d want │, got %q", i, 4, line)
		}
	}
}

func TestWaveformBarsIdleNotEmpty(t *testing.T) {
	got := renderWaveformBars(waveVizOpts{
		Width:  8,
		Height: 5,
	})
	lines := strings.Split(lipglossStrip(got), "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	if !hasBraille(lines[2]) {
		t.Fatalf("idle should keep a flat muted row of dots, got %q", lines[2])
	}
	if strings.Contains(lipglossStrip(got), string(wavePlayhead)) {
		t.Fatalf("idle should not draw a playhead, got %q", got)
	}
	if strings.Contains(got, "[97") {
		t.Fatalf("idle bars should stay muted, got %q", got)
	}
	if strings.Contains(got, "38;2;") {
		t.Fatalf("waveform should use terminal palette, got %q", got)
	}
}

func TestDownsampleLevelsWidth(t *testing.T) {
	levels := make([]float64, 100)
	for i := range levels {
		levels[i] = float64(i) / 99
	}
	got := downsampleLevels(levels, 10)
	if len(got) != 10 {
		t.Fatalf("len = %d", len(got))
	}
	if got[9] <= got[0] {
		t.Fatalf("expected rising max pool, got %v", got)
	}
}

func TestWaveformOverlayLightsLiveDots(t *testing.T) {
	peaks := make([]int, 8)
	for i := range peaks {
		peaks[i] = 255
	}
	levels := []float64{1, 1, 1, 1, 0, 0, 0, 0}
	got := renderWaveformOverlay(waveVizOpts{
		Width:      8,
		Height:     5,
		Peaks:      peaks,
		PlayedFrac: 0.5,
	}, levels, levels, true)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	plain := lipglossStrip(got)
	if !hasBraille(plain) {
		t.Fatalf("overlay should keep waveform dots, got %q", plain)
	}
	if !strings.Contains(got, "[90") {
		t.Fatalf("quiet remaining columns should stay muted waveform, got %q", got)
	}
	if !strings.Contains(got, "[32") {
		t.Fatalf("live overlay should use palette green, got %q", got)
	}
	if strings.Contains(got, "38;2;") {
		t.Fatalf("vis should use terminal palette, got %q", got)
	}
	plainLines := strings.Split(plain, "\n")
	if len(plainLines) > 2 {
		cols := []rune(plainLines[2])
		if len(cols) < 5 || cols[4] != wavePlayhead {
			t.Fatalf("overlay should keep a playhead, got %q", plainLines[2])
		}
	}
}

func TestWaveformOverlayClipsToEnvelope(t *testing.T) {
	peaks := []int{12, 12, 12, 12, 12, 12, 12, 12}
	levels := []float64{1, 1, 1, 1, 1, 1, 1, 1}
	got := renderWaveformOverlay(waveVizOpts{
		Width:  8,
		Height: 5,
		Peaks:  peaks,
	}, levels, levels, true)
	lines := strings.Split(lipglossStrip(got), "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	top := []rune(lines[0])
	if len(top) < 8 {
		t.Fatalf("top width = %d", len(top))
	}
	for i, r := range top {
		if isBraille(r) {
			t.Fatalf("live overlay should stay inside a quiet envelope, col %d of %q", i, lines[0])
		}
	}
	if !hasBraille(lines[2]) {
		t.Fatalf("live overlay should still light the envelope, got %q", lines[2])
	}
}

func TestWaveformOverlayKeepsWaveformInGaps(t *testing.T) {
	peaks := make([]int, 8)
	for i := range peaks {
		peaks[i] = 255
	}
	levels := []float64{1, 1, 1, 1, 1, 1, 1, 1}
	got := renderWaveformOverlay(waveVizOpts{
		Width:  8,
		Height: 5,
		Peaks:  peaks,
	}, levels, levels, true)
	plain := strings.Split(lipglossStrip(got), "\n")
	if len(plain) < 3 {
		t.Fatalf("rows = %d", len(plain))
	}
	mid := []rune(plain[2])
	if len(mid) < 8 {
		t.Fatalf("mid width = %d", len(mid))
	}
	if !isBraille(mid[1]) {
		t.Fatalf("gap columns should keep the muted waveform, got %q", plain[2])
	}
	if !strings.Contains(got, "[90") {
		t.Fatalf("gap columns should stay muted, got %q", got)
	}
	if !strings.Contains(got, "[32") {
		t.Fatalf("live bars should use palette green, got %q", got)
	}
}

func TestWaveformOverlayIdleWaveUsesFullVis(t *testing.T) {
	levels := make([]float64, 8)
	for i := range levels {
		levels[i] = 1
	}
	got := renderWaveformOverlay(waveVizOpts{
		Width:  8,
		Height: 5,
	}, levels, levels, true)
	lines := strings.Split(lipglossStrip(got), "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	top := []rune(lines[0])
	if len(top) < 8 || !isBraille(top[0]) {
		t.Fatalf("without a waveform, live vis should fill the grid, got %q", lines[0])
	}
}

func TestWaveformOverlayKeepsWaveformUnderLive(t *testing.T) {
	peaks := []int{1, 255, 255, 255, 255, 255, 255, 255}
	levels := []float64{0.2, 0.2, 0.2, 0.2, 0.2, 0.2, 0.2, 0.2}
	got := renderWaveformOverlay(waveVizOpts{
		Width:  8,
		Height: 5,
		Peaks:  peaks,
	}, levels, levels, true)
	lines := strings.Split(lipglossStrip(got), "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	row := []rune(lines[2])
	if len(row) < 3 || row[2] != '⣿' {
		t.Fatalf("live bars should keep the waveform dots underneath, got %q", lines[2])
	}
	if strings.Contains(got, "48;") {
		t.Fatalf("live overlay should not paint a background, got %q", got)
	}
}

func TestWaveformBarsVaryHeight(t *testing.T) {
	peaks := []int{12, 12, 255, 255, 12, 12, 255, 255}
	got := renderWaveformBars(waveVizOpts{
		Width:  8,
		Height: 5,
		Peaks:  peaks,
	})
	lines := strings.Split(lipglossStrip(got), "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	top := []rune(lines[0])
	if len(top) < 8 {
		t.Fatalf("top width = %d", len(top))
	}
	if isBraille(top[0]) {
		t.Fatalf("quiet bins should stay off the outer row, got %q", lines[0])
	}
	if !isBraille(top[2]) {
		t.Fatalf("loud bins should reach the outer row, got %q", lines[0])
	}
	mid := []rune(lines[2])
	if !isBraille(mid[0]) || !isBraille(mid[1]) {
		t.Fatalf("dots should fill a dense grid, got %q", lines[2])
	}
}

func TestWaveformBarsHighPeaksVary(t *testing.T) {
	peaks := []int{200, 255, 210, 248, 205, 252, 218, 240}
	got := renderWaveformBars(waveVizOpts{
		Width:  8,
		Height: 5,
		Peaks:  peaks,
	})
	lines := strings.Split(lipglossStrip(got), "\n")
	if len(lines) != 5 {
		t.Fatalf("rows = %d", len(lines))
	}
	top := []rune(lines[0])
	if len(top) < 8 {
		t.Fatalf("top width = %d", len(top))
	}
	filled := 0
	for _, r := range top {
		if isBraille(r) {
			filled++
		}
	}
	if filled == 0 {
		t.Fatalf("loud bins should still reach the outer row, got %q", lines[0])
	}
	if filled == len(top) {
		t.Fatalf("high peaks should not fill a solid outer row, got %q", lines[0])
	}
}

func hasBraille(s string) bool {
	for _, r := range s {
		if isBraille(r) {
			return true
		}
	}
	return false
}

func isBraille(r rune) bool {
	return r >= 0x2800 && r <= 0x28FF
}

func TestVizModeCycle(t *testing.T) {
	if vizModeBars.next() != vizModeFill {
		t.Fatalf("bars should cycle to fill, got %s", vizModeBars.next())
	}
	if vizModeNone.next() != vizModeBars {
		t.Fatalf("none should wrap to bars, got %s", vizModeNone.next())
	}
	if vizModeBars.prev() != vizModeNone {
		t.Fatalf("bars should wrap back to none, got %s", vizModeBars.prev())
	}
}

func TestVizNoneHasNoLiveOverlay(t *testing.T) {
	fill := waveformFill([]int{255, 255, 255, 255, 255, 255, 255, 255}, 8, 5)
	levels := []float64{1, 1, 1, 1, 1, 1, 1, 1}
	live, peak := liveOverlayLayers(fill, levels, levels, waveMaxHalf(5), true, vizModeNone)
	if hasLiveFill(live) || hasLiveFill(peak) {
		t.Fatalf("none should not paint live bars")
	}
}
