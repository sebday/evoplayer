package tui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-sixel"
	"golang.org/x/sys/unix"
)

type artCache struct {
	path    string
	cols    int
	rows    int
	cellW   int
	cellH   int
	layout  string
	seq     string
	overlay bool
	shown   bool
	frame   int
}

var (
	artProtoMu  sync.Mutex
	artProto    termimg.Protocol
	artProtoSet bool
	artCellW    int
	artCellH    int
	artQueryW   int
	artQueryH   int
)

func probeArtProtocol() {
	_ = artProtocol()
}

func resetArtProtocol() {
	artProtoMu.Lock()
	artProtoSet = false
	artProto = 0
	artQueryW, artQueryH = 0, 0
	artCellW, artCellH = 0, 0
	artProtoMu.Unlock()
	termimg.ClearFeatureCache()
}

func artProtocol() termimg.Protocol {
	artProtoMu.Lock()
	defer artProtoMu.Unlock()
	if artProtoSet {
		return artProto
	}
	override := os.Getenv("EVOPLAYER_ART_PROTOCOL")
	tty := term.IsTerminal(os.Stdout.Fd())
	emu := detectEmulator()
	detected := termimg.Unsupported
	if tty {
		tuneArtFeatures(emu)
		detected = termimg.DetectProtocol()
	}
	artProto = pickArtProtocol(override, tty, emu, detected)
	artProtoSet = true
	return artProto
}

func pickArtProtocol(override string, tty bool, emu string, detected termimg.Protocol) termimg.Protocol {
	if p, ok := parseArtProtocol(override); ok {
		return p
	}
	if !tty {
		return termimg.Halfblocks
	}
	switch emu {
	case "foot":
		// Foot has full Sixel. It does not implement Kitty graphics
		// (only the Kitty keyboard protocol). Omarchy sets TERM=xterm-256color
		// so go-termimg's env check never sees "foot".
		return termimg.Sixel
	case "kitty", "ghostty", "wezterm":
		return termimg.Kitty
	}
	if graphicsProtocol(detected) {
		return detected
	}
	return termimg.Halfblocks
}

func parseArtProtocol(s string) (termimg.Protocol, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "kitty":
		return termimg.Kitty, true
	case "sixel":
		return termimg.Sixel, true
	case "iterm2", "iterm":
		return termimg.ITerm2, true
	case "halfblocks", "blocks", "unicode":
		return termimg.Halfblocks, true
	default:
		return termimg.Unsupported, false
	}
}

func graphicsProtocol(p termimg.Protocol) bool {
	return p == termimg.Sixel || p == termimg.Kitty || p == termimg.ITerm2
}

func detectEmulator() string {
	termName := strings.ToLower(os.Getenv("TERM"))
	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(termName, "kitty"):
		return "kitty"
	case os.Getenv("GHOSTTY_RESOURCES_DIR") != "" || strings.Contains(termName, "ghostty"):
		return "ghostty"
	case os.Getenv("WEZTERM_EXECUTABLE") != "" || strings.Contains(termName, "wezterm"):
		return "wezterm"
	case strings.Contains(termName, "foot"):
		return "foot"
	}
	pid := os.Getppid()
	for i := 0; i < 12 && pid > 1; i++ {
		switch procComm(pid) {
		case "foot", "footclient":
			return "foot"
		case "kitty":
			return "kitty"
		case "ghostty":
			return "ghostty"
		case "wezterm", "wezterm-gui":
			return "wezterm"
		}
		pid = parentPID(pid)
	}
	return ""
}

func procComm(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func parentPID(pid int) int {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "PPid:") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "PPid:")))
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

func tuneArtFeatures(emu string) {
	feat := termimg.QueryTerminalFeatures()
	artQueryW, artQueryH = feat.FontWidth, feat.FontHeight
	iw, ih := artCellPixels()
	cw, ch := pickArtCellSize(artQueryW, artQueryH, iw, ih)
	feat.FontWidth = cw
	feat.FontHeight = ch
	artCellW, artCellH = cw, ch
	switch emu {
	case "foot":
		feat.SixelGraphics = true
		feat.KittyGraphics = false
	case "kitty", "ghostty", "wezterm":
		feat.KittyGraphics = true
	}
}

func pickArtCellSize(queryW, queryH, ioctlW, ioctlH int) (cw, ch int) {
	if ioctlW >= 4 && ioctlH >= 8 {
		return ioctlW, ioctlH
	}
	if queryW >= 4 && queryH >= 8 {
		return queryW, queryH
	}
	return 8, 16
}

func artCellSize() (cw, ch int) {
	iw, ih := artCellPixels()
	cw, ch = pickArtCellSize(artQueryW, artQueryH, iw, ih)
	if cw >= 4 && ch >= 8 {
		artCellW, artCellH = cw, ch
		return cw, ch
	}
	if artCellW >= 4 && artCellH >= 8 {
		return artCellW, artCellH
	}
	return 8, 16
}

func artCellPixels() (cw, ch int) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 0, 0
	}
	cw = int(ws.Xpixel) / int(ws.Col)
	ch = int(ws.Ypixel) / int(ws.Row)
	if cw < 4 || ch < 8 {
		return 0, 0
	}
	return cw, ch
}

func artBlank(cols, rows int) string {
	cols = max(2, cols)
	rows = max(1, rows)
	line := strings.Repeat(" ", cols)
	return strings.Repeat(line+"\n", rows-1) + line
}

func encodeArt(img image.Image, cols, rows int) (layout, seq string, overlay bool) {
	cols = max(2, cols)
	rows = max(1, rows)
	if img == nil {
		return placeholderArt(cols, rows), "", false
	}
	p := artProtocol()
	if !graphicsProtocol(p) {
		return renderArt(img, cols, rows), "", false
	}
	out, err := renderGraphicsArt(img, cols, rows, p)
	if err != nil || out == "" {
		return renderArt(img, cols, rows), "", false
	}
	return artBlank(cols, rows), out, true
}

func renderGraphicsArt(img image.Image, cols, rows int, p termimg.Protocol) (string, error) {
	cw, ch := artCellSize()
	pxW := cols * cw
	pxH := rows * ch
	if pxW > 0 && pxH > 0 && pxW != pxH {
		side := pxW
		if pxH < side {
			side = pxH
		}
		pxW, pxH = side, side
	}
	if p == termimg.Sixel {
		return renderSixel(img, pxW, pxH)
	}
	ti := termimg.New(img).Protocol(p).SizePixels(pxW, pxH)
	if p == termimg.Kitty {
		ti = ti.Compression(true)
	}
	return ti.Render()
}

func renderSixel(img image.Image, pxW, pxH int) (string, error) {
	pxW = max(6, pxW)
	pxH = max(6, pxH)
	src := scaleImage(img, pxW, pxH)
	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	enc.Colors = 256
	if err := enc.Encode(src); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func overlayArt(view, seq string, row, col, frame int) string {
	if seq == "" || row < 1 || col < 1 {
		return view
	}
	if frame <= 0 {
		return view + fmt.Sprintf("\x1b[s\x1b[%d;%dH%s\x1b[u", row, col, seq)
	}
	n := frame%2 + 1
	return view + fmt.Sprintf("\x1b[s\x1b[%d;%dH%s\x1b[u\x1b[%dC\x1b[%dD", row, col, seq, n, n)
}

func loadArt(path string) image.Image {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func renderArt(img image.Image, cols, rows int) string {
	cols = max(2, cols)
	rows = max(1, rows)
	if img == nil {
		return placeholderArt(cols, rows)
	}
	px := scaleImage(img, cols, rows*2)
	return paintHalfBlocks(px)
}

func placeholderArt(cols, rows int) string {
	cell := logoColor().NewStyle().
		Foreground(colMuted).
		Background(lipgloss.Color("0")).
		Render("▀")
	line := strings.Repeat(cell, max(2, cols))
	return strings.Repeat(line+"\n", max(1, rows)-1) + line
}

func scaleImage(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	if sw < 1 || sh < 1 {
		return dst
	}
	for y := 0; y < h; y++ {
		y0 := y * sh / h
		y1 := (y + 1) * sh / h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0 := x * sw / w
			x1 := (x + 1) * sw / w
			if x1 <= x0 {
				x1 = x0 + 1
			}
			dst.Set(x, y, boxSample(src, sb, x0, y0, x1, y1))
		}
	}
	return dst
}

func boxSample(src image.Image, sb image.Rectangle, x0, y0, x1, y1 int) color.Color {
	var r, g, b, n uint32
	for sy := y0; sy < y1; sy++ {
		for sx := x0; sx < x1; sx++ {
			cr, cg, cb, _ := src.At(sb.Min.X+sx, sb.Min.Y+sy).RGBA()
			r += cr
			g += cg
			b += cb
			n++
		}
	}
	if n == 0 {
		return color.Black
	}
	return color.RGBA{
		R: uint8(r / n >> 8),
		G: uint8(g / n >> 8),
		B: uint8(b / n >> 8),
		A: 255,
	}
}

func paintHalfBlocks(img *image.RGBA) string {
	b := img.Bounds()
	cols := b.Dx()
	rows := (b.Dy() + 1) / 2
	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		var sb strings.Builder
		for x := 0; x < cols; x++ {
			top := img.RGBAAt(x, y*2)
			bot := top
			if y*2+1 < b.Dy() {
				bot = img.RGBAAt(x, y*2+1)
			}
			sb.WriteString(artColor().NewStyle().
				Foreground(lipgloss.Color(colorHex(top))).
				Background(lipgloss.Color(colorHex(bot))).
				Render("▀"))
		}
		lines[y] = sb.String()
	}
	return strings.Join(lines, "\n")
}

func colorHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}
