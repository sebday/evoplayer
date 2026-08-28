package tui

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/internal/waveform"
)

func loadWaveformPeaks(path string) []int {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var p waveform.Payload
	if json.Unmarshal(b, &p) != nil || len(p.Data) == 0 {
		return nil
	}
	return p.Data
}

func downsamplePeaks(data []int, width int) []int {
	if width < 1 || len(data) == 0 {
		return nil
	}
	out := make([]int, width)
	for i := 0; i < width; i++ {
		start := i * len(data) / width
		end := (i + 1) * len(data) / width
		if end <= start {
			end = start + 1
		}
		if end > len(data) {
			end = len(data)
		}
		mx := 0
		for _, v := range data[start:end] {
			if v > mx {
				mx = v
			}
		}
		out[i] = mx
	}
	return out
}

func playheadIndex(position, duration float64, width int) int {
	if width < 1 || duration <= 0 {
		return 0
	}
	n := int(position / duration * float64(width))
	if n < 0 {
		return 0
	}
	if n >= width {
		return width - 1
	}
	return n
}

func renderWaveform(peaks []int, playAt, height int) string {
	width := len(peaks)
	if width < 1 {
		return ""
	}
	height = max(1, height)
	rows := make([][]string, height)
	for r := 0; r < height; r++ {
		rows[r] = make([]string, width)
	}
	for i, v := range peaks {
		n := ampLevelForHeight(v, height)
		played := i <= playAt
		for r := 0; r < height; r++ {
			fromBottom := height - r
			rows[r][i] = waveGlyph(fromBottom <= n, played)
		}
	}
	lines := make([]string, height)
	for r := 0; r < height; r++ {
		lines[r] = strings.Join(rows[r], "")
	}
	return strings.Join(lines, "\n")
}

func ampLevelForHeight(v, height int) int {
	if v <= 0 || height <= 0 {
		return 0
	}
	n := (v*height + 127) / 255
	if n < 0 {
		return 0
	}
	if n > height {
		return height
	}
	return n
}

func waveGlyph(filled bool, played bool) string {
	ch := " "
	if filled {
		ch = "█"
	}
	fg := colMuted
	if played {
		fg = colAccent
	}
	return lipgloss.NewStyle().Foreground(fg).Render(ch)
}
