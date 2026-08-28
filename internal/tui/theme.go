package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	colAccent = lipgloss.Color("#C4B5FD")
	colGood   = lipgloss.Color("#86EFAC")
	colMuted  = lipgloss.Color("#6B7280")
	colBorder = lipgloss.Color("#4B5563")
	colWarn   = lipgloss.Color("#FBBF24")
)

var (
	logoRendererOnce sync.Once
	logoRenderer     *lipgloss.Renderer
)

func logoColor() *lipgloss.Renderer {
	logoRendererOnce.Do(func() {
		logoRenderer = lipgloss.NewRenderer(os.Stdout)
		logoRenderer.SetColorProfile(termenv.TrueColor)
	})
	return logoRenderer
}

func styleTitle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colAccent).Bold(true)
}

func styleLogoCell(t float64) lipgloss.Style {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	accent := string(colAccent)
	hi := mixHex(accent, "#FFFFFF", 0.72)
	lo := mixHex(accent, "#000000", 0.48)
	hex := mixHex(hi, accent, t/0.4)
	if t >= 0.4 {
		hex = mixHex(accent, lo, (t-0.4)/0.6)
	}
	return logoColor().NewStyle().Foreground(lipgloss.Color(hex)).Bold(true)
}

func mixHex(a, b string, t float64) string {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	ar, ag, ab := rgb(a)
	br, bg, bb := rgb(b)
	r := ar + int(t*float64(br-ar))
	g := ag + int(t*float64(bg-ag))
	bl := ab + int(t*float64(bb-ab))
	return fmt.Sprintf("#%02X%02X%02X", r, g, bl)
}

func rgb(hex string) (r, g, b int) {
	s := strings.TrimPrefix(hex, "#")
	if len(s) != 6 {
		return 196, 181, 253
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 196, 181, 253
	}
	return int(n >> 16), int((n >> 8) & 0xff), int(n & 0xff)
}

func styleMuted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colMuted)
}

func fieldset(legend, inner string, width, height int, active bool) string {
	width = max(8, width)
	innerW := width - 2
	innerH := max(1, height-2)
	body := lipgloss.NewStyle().Padding(1, 1).Width(innerW).Height(innerH).Render(inner)
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = padExact(lines[i], innerW)
	}

	fg := colBorder
	if active {
		fg = colAccent
	}
	border := lipgloss.NewStyle().Foreground(fg)
	label := strings.TrimSpace(legend)
	if label != "" {
		label = " " + label + " "
	}
	lw := lipgloss.Width(label)
	dash := innerW - lw
	if dash < 0 {
		label = padExact(label, innerW)
		lw = innerW
		dash = 0
	}
	top := border.Render("╭" + label + strings.Repeat("─", dash) + "╮")
	var mid strings.Builder
	for _, line := range lines {
		mid.WriteString(border.Render("│") + line + border.Render("│") + "\n")
	}
	bottom := border.Render("╰" + strings.Repeat("─", innerW) + "╯")
	return top + "\n" + mid.String() + bottom
}

func padExact(s string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Width(width).Render(s)
}

func styleNavActive() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colAccent).Bold(true)
}

func styleSelected() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colAccent).Bold(true)
}

func styleGood() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colGood)
}

func styleSprout() lipgloss.Style {
	return logoColor().NewStyle().Foreground(colGood)
}

func styleWarn() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colWarn)
}
