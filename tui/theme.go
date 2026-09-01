package tui

import (
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	colAccent = lipgloss.Color("4") // purple
	colGood   = lipgloss.Color("2") // green
	colLiked  = lipgloss.Color("1") // red
	colMuted  = lipgloss.Color("8") // bright black
	colText   = lipgloss.Color("7") // white
	colBG     = lipgloss.Color("0") // terminal background (black)
	colWarn   = lipgloss.Color("3") // yellow
)

func panelIdleColor(num int) lipgloss.Color {
	_ = num
	return colAccent
}

func init() {
	lipgloss.SetColorProfile(termenv.ANSI)
}

var (
	logoRendererOnce sync.Once
	logoRenderer     *lipgloss.Renderer
	artRendererOnce  sync.Once
	artRenderer      *lipgloss.Renderer
)

func logoColor() *lipgloss.Renderer {
	logoRendererOnce.Do(func() {
		logoRenderer = lipgloss.NewRenderer(os.Stdout)
		logoRenderer.SetColorProfile(termenv.ANSI)
	})
	return logoRenderer
}

func artColor() *lipgloss.Renderer {
	artRendererOnce.Do(func() {
		artRenderer = lipgloss.NewRenderer(os.Stdout)
		artRenderer.SetColorProfile(termenv.TrueColor)
	})
	return artRenderer
}

func styleMuted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colMuted)
}

func styleText() lipgloss.Style {
	return logoColor().NewStyle().Foreground(colText)
}

func fieldsetPad(legend, legendTopRight, inner string, width, height int, active bool, padY, padX int, bottomLeft, bottomRight string, num int) string {
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
	return fieldsetBox(legend, legendTopRight, lines, innerW, active, bottomLeft, bottomRight, num)
}

func fieldsetArt(legend, inner string, width, height, padX, cols int, bottomRight string, num int) string {
	width = max(8, width)
	innerW := width - 2
	innerH := max(1, height-2)
	cols = max(1, cols)
	padX = max(0, padX)
	src := strings.Split(inner, "\n")
	left := strings.Repeat(" ", padX)
	right := strings.Repeat(" ", max(0, innerW-padX-cols))
	blank := strings.Repeat(" ", innerW)
	lines := make([]string, innerH)
	for i := 0; i < innerH; i++ {
		if i < len(src) && src[i] != "" {
			lines[i] = left + src[i] + right
		} else {
			lines[i] = blank
		}
	}
	return fieldsetBox(legend, "", lines, innerW, false, "", bottomRight, num)
}

func fieldsetBox(legend, legendTopRight string, lines []string, innerW int, active bool, bottomLeft, bottomRight string, num int, borderOverride ...borderChase) string {
	border := newBorderChase(active, num)
	if len(borderOverride) > 0 {
		border = borderOverride[0]
	}
	top := fieldsetTop(legend, legendTopRight, innerW, border, num)
	var mid strings.Builder
	for _, line := range lines {
		mid.WriteString(border.paint("│") + line + border.paint("│") + "\n")
	}
	bottom := fieldsetBottom(bottomLeft, bottomRight, innerW, border)
	return top + "\n" + mid.String() + bottom
}

func browseSectionRule(width int, active bool) string {
	w := max(4, width)
	border := newBorderChase(active, 2)
	return border.paint(strings.Repeat("─", w))
}

func fieldsetInlineFlush(legend, inner string, width int, active bool, num int) string {
	return fieldsetInlineEdge(legend, inner, width, active, num, false)
}

func fieldsetInlineEdge(legend, inner string, width int, active bool, num int, bottom bool) string {
	width = max(8, width)
	innerW := width - 2
	line := padExact(lipgloss.NewStyle().Padding(0, 1).Render(inner), innerW)

	border := newBorderChase(active, num)
	top := fieldsetTop(legend, "", innerW, border, num)
	mid := border.paint("│") + line + border.paint("│")
	if !bottom {
		return top + "\n" + mid
	}
	bottomEdge := fieldsetBottom("", "", innerW, border)
	return top + "\n" + mid + "\n" + bottomEdge
}

func fieldsetTop(left, right string, innerW int, border borderChase, num int) string {
	return fieldsetEdge(left, right, innerW, border, "top", "╭", "╮", "┐", "┌", num)
}

func fieldsetBottom(left, right string, innerW int, border borderChase) string {
	return fieldsetEdge(left, right, innerW, border, "bottom", "╰", "╯", "┘", "└", 0)
}

func fieldsetEdge(left, right string, innerW int, border borderChase, side, start, end, dropOpen, dropClose string, num int) string {
	hex := border.hex()
	branch := side == "top" && right == "─"
	if branch {
		right = ""
	}
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
	b.WriteString(border.paint(start))
	x := 1
	writeCh := func(ch string) {
		if x >= W-1 {
			return
		}
		b.WriteString(border.paint(ch))
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
		if branch {
			writeCh("┬")
		}
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
	b.WriteString(border.paint(end))
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
	st := logoColor().NewStyle().Foreground(lipgloss.Color(hex)).Bold(true)
	return st.Render(label)
}

type borderChase struct {
	active bool
	idle   lipgloss.Color
}

func newBorderChase(active bool, num int) borderChase {
	return borderChase{
		active: active,
		idle:   panelIdleColor(num),
	}
}

func (c borderChase) paint(ch string) string {
	return logoColor().NewStyle().Foreground(lipgloss.Color(c.hex())).Render(ch)
}

func (c borderChase) hex() string {
	if c.active {
		return string(colLiked)
	}
	return string(colAccent)
}

func panelBorderHex(num int, active bool) string {
	return borderChase{active: active, idle: panelIdleColor(num)}.hex()
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

func styleLiked() lipgloss.Style {
	return logoColor().NewStyle().Foreground(colLiked)
}

func stylePlaylistSelected() lipgloss.Style {
	return logoColor().NewStyle().Background(colGood).Foreground(colBG).Bold(true)
}

func styleWarn() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colWarn)
}
