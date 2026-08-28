package tui

import (
	"fmt"
	"math"
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
	colText   = lipgloss.Color("#FFFFFF")
	colBorder = lipgloss.Color("#4B5563")
	colWarn   = lipgloss.Color("#FBBF24")
	pulseLo   = "#00B4E6"
	pulseHi   = "#9AEDFE" // terminal bright_blue
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

func styleText() lipgloss.Style {
	return logoColor().NewStyle().Foreground(colText)
}

func fieldset(legend, inner string, width, height int, active bool, pulse float64, num int) string {
	return fieldsetPad(legend, inner, width, height, active, pulse, 1, 1, "", "", num)
}

func fieldsetPad(legend, inner string, width, height int, active bool, pulse float64, padY, padX int, bottomLeft, bottomRight string, num int) string {
	width = max(8, width)
	innerW := width - 2
	innerH := max(1, height-2)
	body := lipgloss.NewStyle().Padding(padY, padX).Width(innerW).Height(innerH).Render(inner)
	lines := strings.Split(body, "\n")
	if len(lines) > innerH {
		lines = lines[:innerH]
	}
	for i := range lines {
		lines[i] = padExact(lines[i], innerW)
	}

	border := newBorderChase(active, pulse, innerW, len(lines))
	top := fieldsetTop(legend, innerW, border, num)
	var mid strings.Builder
	for r, line := range lines {
		mid.WriteString(border.cell("left", r, "│") + line + border.cell("right", r, "│") + "\n")
	}
	bottom := fieldsetBottom(bottomLeft, bottomRight, innerW, border)
	return top + "\n" + mid.String() + bottom
}

// fieldsetInline is a single-line fieldset without vertical inner padding (e.g. search).
func fieldsetInline(legend, inner string, width int, active bool, pulse float64, num int) string {
	width = max(8, width)
	innerW := width - 2
	line := padExact(lipgloss.NewStyle().Padding(0, 1).Render(inner), innerW)

	border := newBorderChase(active, pulse, innerW, 1)
	top := fieldsetTop(legend, innerW, border, num)
	mid := border.cell("left", 0, "│") + line + border.cell("right", 0, "│")
	bottom := fieldsetBottom("", "", innerW, border)
	return top + "\n" + mid + "\n" + bottom
}

func fieldsetTop(legend string, innerW int, border borderChase, num int) string {
	return fieldsetEdge(legend, "", innerW, border, "top", "╭", "╮", "┐", "┌", num)
}

func fieldsetBottom(left, right string, innerW int, border borderChase) string {
	return fieldsetEdge(left, right, innerW, border, "bottom", "╰", "╯", "┘", "└", 0)
}

func fieldsetEdge(left, right string, innerW int, border borderChase, side, start, end, dropOpen, dropClose string, num int) string {
	hex := border.hex()
	leftLabel := styleFieldsetLegend(left, hex, border.active)
	rightLabel := styleFieldsetLegend(right, hex, border.active)
	numStr := stylePanelNum(num, hex)
	W := innerW + 2

	leftW := lipgloss.Width(numStr) + lipgloss.Width(leftLabel)
	rightW := lipgloss.Width(rightLabel)
	leftExtra, rightExtra := 0, 0
	if leftW > 0 {
		leftExtra = 3
	}
	if rightW > 0 {
		rightExtra = 3
	}
	budget := max(0, W-2-leftExtra-rightExtra)
	if leftW+rightW > budget {
		maxRight := max(0, budget-leftW)
		if rightW > maxRight {
			rightLabel = lipgloss.NewStyle().MaxWidth(maxRight).Render(rightLabel)
			rightW = lipgloss.Width(rightLabel)
		}
		maxLeft := max(0, budget-rightW)
		if leftW > maxLeft {
			maxLab := max(0, maxLeft-lipgloss.Width(numStr))
			leftLabel = lipgloss.NewStyle().MaxWidth(maxLab).Render(leftLabel)
			leftW = lipgloss.Width(numStr) + lipgloss.Width(leftLabel)
		}
	}

	var b strings.Builder
	b.WriteString(border.cell(side, 0, start))
	x := 1
	writeCh := func(ch string) {
		if x >= W-1 {
			return
		}
		b.WriteString(border.cell(side, x, ch))
		x++
	}
	writeRaw := func(s string) {
		if s == "" {
			return
		}
		w := lipgloss.Width(s)
		if x+w > W-1 {
			s = lipgloss.NewStyle().MaxWidth(max(0, W-1-x)).Render(s)
			w = lipgloss.Width(s)
		}
		b.WriteString(s)
		x += w
	}

	if leftW > 0 {
		writeCh("─")
		writeCh(dropOpen)
		writeRaw(numStr)
		writeRaw(leftLabel)
		writeCh(dropClose)
	}

	rightNeed := 1
	if rightW > 0 {
		rightNeed = 1 + rightW + 1 + 1
		if x+rightNeed < W {
			rightNeed++
		}
	}
	for x < W-rightNeed {
		writeCh("─")
	}
	if rightW > 0 {
		writeCh(dropOpen)
		writeRaw(rightLabel)
		writeCh(dropClose)
	}
	for x < W-1 {
		writeCh("─")
	}
	b.WriteString(border.cell(side, W-1, end))
	return b.String()
}

var panelNums = []string{"¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"}

func stylePanelNum(n int, hex string) string {
	if n < 1 || n > len(panelNums) {
		return ""
	}
	return logoColor().NewStyle().Foreground(lipgloss.Color(hex)).Render(panelNums[n-1])
}

func styleFieldsetLegend(legend, hex string, active bool) string {
	label := strings.TrimSpace(legend)
	if label == "" {
		return ""
	}
	if strings.Contains(label, "\x1b") {
		return label
	}
	st := logoColor().NewStyle().Foreground(lipgloss.Color(hex))
	if active {
		st = st.Bold(true)
	}
	return st.Render(label)
}

type borderChase struct {
	active bool
	phase  float64
	innerW int
	bodyH  int
}

func newBorderChase(active bool, phase float64, innerW, bodyH int) borderChase {
	return borderChase{active: active, phase: phase, innerW: max(1, innerW), bodyH: max(1, bodyH)}
}

func (c borderChase) cell(side string, pos int, ch string) string {
	return c.paint(ch)
}

func (c borderChase) paint(ch string) string {
	return logoColor().NewStyle().Foreground(lipgloss.Color(c.hex())).Render(ch)
}

func (c borderChase) hex() string {
	if !c.active {
		return string(colAccent)
	}
	t := 0.45
	if c.phase >= 0 {
		t = (math.Sin(c.phase*2*math.Pi) + 1) / 2
	}
	return mixHex(pulseLo, pulseHi, t)
}

func padExact(s string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Width(width).Render(s)
}

func padToHeight(s string, height int) string {
	if height < 1 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		return strings.Join(lines[:height], "\n")
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func clipLines(s string, maxLines int) string {
	if maxLines < 1 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func styleSelected() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colAccent).Bold(true)
}

func styleGood() lipgloss.Style {
	return logoColor().NewStyle().Foreground(colGood)
}

func stylePill() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colMuted).
		Background(lipgloss.Color("#2A3038")).
		Padding(0, 1)
}

func styleWarn() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colWarn)
}
