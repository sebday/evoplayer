package tui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const logoSprout = '♪'

// miniwi FIGlet (https://patorjk.com/software/taag/#f=miniwi)
var logoArt = []string{
	"▄▖▖▖▄▖▄▖▖ ▄▖▖▖▄▖▄▖" + string(logoSprout),
	"▙▖▌▌▌▌▙▌▌ ▌▌▌▌▙▖▙▘",
	"▙▖▚▘▙▌▌ ▙▖▛▌▐ ▙▖▌▌",
}

func renderLogo(maxWidth int) string {
	return renderLogoAt(maxWidth, 0)
}

func renderLogoAt(maxWidth int, phase float64) string {
	if maxWidth < 8 {
		return styleTitle().Render("EVOPLAYER")
	}
	w := logoArtWidth()
	if maxWidth < w {
		if maxWidth >= 8 {
			return styleTitle().Render("EVO")
		}
		return styleTitle().Render("EVOPLAYER")
	}
	return colorizeLogo(logoArt, phase)
}

func logoArtWidth() int {
	if len(logoArt) == 0 {
		return 0
	}
	return lipgloss.Width(logoArt[0])
}

func colorizeLogo(lines []string, phase float64) string {
	if len(lines) == 0 {
		return styleTitle().Render("EVOPLAYER")
	}
	w := lipgloss.Width(lines[0])
	lastX := max(1, w-1)
	lastY := max(1, len(lines)-1)
	out := make([]string, len(lines))
	for y, line := range lines {
		var b strings.Builder
		x := 0
		for _, r := range line {
			if r == ' ' {
				b.WriteByte(' ')
				x++
				continue
			}
			if r == logoSprout {
				b.WriteString(styleSprout().Render(string(r)))
				x++
				continue
			}
			u := (float64(x)/float64(lastX) + float64(lastY-y)/float64(lastY)) / 2
			d := math.Abs(wrapUnit(u - phase))
			t := d / 0.22
			if t > 1 {
				t = 1
			}
			b.WriteString(styleLogoCell(t).Render(string(r)))
			x++
		}
		out[y] = b.String()
	}
	return strings.Join(out, "\n")
}

func wrapUnit(v float64) float64 {
	v = v - math.Floor(v)
	if v > 0.5 {
		return v - 1
	}
	return v
}
