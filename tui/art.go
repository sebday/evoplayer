package tui

import (
	"bytes"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/sebday/evoplayer/server/art"
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

func artProtocol() termimg.Protocol {
	artProtoMu.Lock()
	defer artProtoMu.Unlock()
	if artProtoSet {
		return artProto
	}
	override := os.Getenv("EVOPLAYER_ART_PROTOCOL")
	tty := term.IsTerminal(os.Stdout.Fd())
	emu := detectEmulator()
	if tty {
		tuneArtFeatures(emu)
	}
	// Skip DetectProtocol(): it CSI-queries stdin and leftover bytes eat the
	// first keypress once bubbletea starts reading the TTY.
	artProto = pickArtProtocol(override, tty, emu, termimg.Unsupported)
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
	bypass := os.Getenv("TERMIMG_BYPASS_DETECTION")
	if bypass == "" {
		switch emu {
		case "foot":
			bypass = "sixel"
		case "kitty", "ghostty", "wezterm":
			bypass = "kitty"
		default:
			bypass = "halfblocks"
		}
		_ = os.Setenv("TERMIMG_BYPASS_DETECTION", bypass)
	}
	feat := termimg.QueryTerminalFeatures()
	iw, ih := artCellPixels()
	cw, ch := pickArtCellSize(feat.FontWidth, feat.FontHeight, iw, ih)
	feat.FontWidth = cw
	feat.FontHeight = ch
	artQueryW, artQueryH = cw, ch
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
	fd := int(os.Stdout.Fd())
	if f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0); err == nil {
		fd = int(f.Fd())
		cw, ch = winsizeCellPixels(fd)
		_ = f.Close()
		if cw >= 4 && ch >= 8 {
			if fcw, fch, ok := footCellPixels(fd); ok {
				return fcw, fch
			}
			return cw, ch
		}
	}
	cw, ch = winsizeCellPixels(fd)
	if cw >= 4 && ch >= 8 {
		if fcw, fch, ok := footCellPixels(fd); ok {
			return fcw, fch
		}
	}
	return cw, ch
}

func cellPixelsCeil(px, cells int) int {
	if cells <= 0 || px <= 0 {
		return 0
	}
	return (px + cells - 1) / cells
}

func footCellPixels(fd int) (cw, ch int, ok bool) {
	if detectEmulator() != "foot" {
		return 0, 0, false
	}
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 0, 0, false
	}
	cw = cellPixelsCeil(int(ws.Xpixel), int(ws.Col))
	ch = cellPixelsCeil(int(ws.Ypixel), int(ws.Row))
	if cw < 4 || ch < 8 {
		return 0, 0, false
	}
	return cw, ch, true
}

func terminalWinsize() (*unix.Winsize, error) {
	if f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0); err == nil {
		ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
		_ = f.Close()
		if err == nil {
			return ws, nil
		}
	}
	return unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
}

func winsizeCellPixels(fd int) (cw, ch int) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
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

func winsizeGridPixels(cols, rows int) (pxW, pxH int, ok bool) {
	ws, err := terminalWinsize()
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 0, 0, false
	}
	pxW = (int(ws.Xpixel)*cols + int(ws.Col) - 1) / int(ws.Col)
	pxH = (int(ws.Ypixel)*rows + int(ws.Row) - 1) / int(ws.Row)
	if pxW < 4 || pxH < 8 {
		return 0, 0, false
	}
	return pxW, pxH, true
}

func artBlank(cols, rows int) string {
	cols = max(2, cols)
	rows = max(1, rows)
	line := strings.Repeat(" ", cols)
	return strings.Repeat(line+"\n", rows-1) + line
}

const kittyArtID = 1
const kittyPlaceholder = "\U0010EEEE"

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
	if p == termimg.Kitty {
		layout, seq, err := renderKittyArt(img, cols, rows)
		if err != nil || seq == "" {
			return renderArt(img, cols, rows), "", false
		}
		return layout, seq, true
	}
	out, err := renderGraphicsArt(img, cols, rows, p)
	if err != nil || out == "" {
		return renderArt(img, cols, rows), "", false
	}
	return artBlank(cols, rows), out, true
}

func artGridPixels(cols, rows int) (pxW, pxH int) {
	if w, h, ok := winsizeGridPixels(cols, rows); ok {
		return w, h
	}
	cw, ch := artCellSize()
	return max(1, cols*cw), max(1, rows*ch)
}

func renderKittyArt(img image.Image, cols, rows int) (layout, seq string, err error) {
	cols = max(2, cols)
	rows = max(1, rows)
	if img == nil {
		return "", "", fmt.Errorf("empty art image")
	}
	ti := termimg.New(img).Protocol(termimg.Kitty).
		Width(cols).Height(rows).
		Scale(termimg.ScaleFill).
		Compression(true).ZIndex(1).ImageNum(kittyArtID)
	seq, err = ti.Render()
	if err != nil || seq == "" {
		return "", "", err
	}
	seq = stripKittyPlaceholders(seq)
	if seq == "" {
		return "", "", fmt.Errorf("empty kitty transmit")
	}
	seq = forceKittyPlacementCells(seq, cols, rows)
	return artBlank(cols, rows), seq, nil
}

func stripKittyPlaceholders(seq string) string {
	if i := strings.Index(seq, kittyPlaceholder); i >= 0 {
		seq = seq[:i]
	}
	return strings.TrimRight(seq, "\r\n")
}

func kittyPlacementCells(seq string) (cols, rows int, ok bool) {
	c, cOK := kittyDim(seq, ",c=")
	r, rOK := kittyDim(seq, ",r=")
	if !cOK || !rOK {
		return 0, 0, false
	}
	return c, r, true
}

func kittyDim(seq, key string) (int, bool) {
	i := strings.Index(seq, key)
	if i < 0 {
		return 0, false
	}
	start := i + len(key)
	end := start
	for end < len(seq) && seq[end] >= '0' && seq[end] <= '9' {
		end++
	}
	if end == start {
		return 0, false
	}
	n, err := strconv.Atoi(seq[start:end])
	if err != nil {
		return 0, false
	}
	return n, true
}

func forceKittyPlacementCells(seq string, cols, rows int) string {
	seq = replaceKittyDim(seq, ",c=", max(1, cols))
	seq = replaceKittyDim(seq, ",r=", max(1, rows))
	return seq
}

func replaceKittyDim(seq, key string, n int) string {
	i := strings.Index(seq, key)
	if i < 0 {
		return seq
	}
	start := i + len(key)
	end := start
	for end < len(seq) && seq[end] >= '0' && seq[end] <= '9' {
		end++
	}
	if end == start {
		return seq
	}
	return seq[:start] + strconv.Itoa(n) + seq[end:]
}

func renderGraphicsArt(img image.Image, cols, rows int, p termimg.Protocol) (string, error) {
	if p == termimg.Sixel && detectEmulator() == "foot" {
		pxW, pxH := artGridPixels(cols, rows)
		ti := termimg.New(img).Protocol(p).WidthPixels(pxW).HeightPixels(pxH).Scale(termimg.ScaleFill)
		return ti.Render()
	}
	ti := termimg.New(img).Protocol(p).Width(cols).Height(rows).Scale(termimg.ScaleFill)
	if p == termimg.Kitty {
		ti = ti.Compression(true)
	}
	return ti.Render()
}

func overlayArt(view, seq string, row, col int) string {
	if seq == "" {
		return view
	}
	if row < 1 || col < 1 {
		return view + seq
	}
	return view + fmt.Sprintf("\x1b[s\x1b[%d;%dH%s\x1b[u", row, col, seq)
}

func kittyPlaceSeq(cols, rows int) string {
	cols = max(2, cols)
	rows = max(1, rows)
	return fmt.Sprintf("\x1b_Ga=p,I=%d,c=%d,r=%d,z=1,q=2\x1b\\", kittyArtID, cols, rows)
}

func (m *model) storeArtOverlay(seq string, row, col, cols, rows int) {
	if m.frames == nil {
		return
	}
	m.frames.mu.Lock()
	if m.frames.quitting {
		m.frames.mu.Unlock()
		return
	}
	place := seq
	if strings.Contains(seq, "\x1b_G") {
		place = kittyPlaceSeq(cols, rows)
	}
	m.frames.artDirty = m.frames.artSeq != seq || m.frames.artRow != row || m.frames.artCol != col || m.frames.artSeq == ""
	m.frames.artSeq = seq
	m.frames.artPlace = place
	m.frames.artRow = row
	m.frames.artCol = col
	m.frames.mu.Unlock()
}

func (m *model) clearStoredArtOverlay() {
	if m.frames == nil {
		return
	}
	m.frames.mu.Lock()
	m.frames.artSeq = ""
	m.frames.artPlace = ""
	m.frames.artRow = 0
	m.frames.artCol = 0
	m.frames.artDirty = false
	m.frames.mu.Unlock()
}

type artRestorer struct {
	w         io.Writer
	fd        uintptr
	frames    *frameCache
	restoring bool
}

func newArtRestorer(f *os.File, frames *frameCache) *artRestorer {
	return &artRestorer{w: f, fd: f.Fd(), frames: frames}
}

func (w *artRestorer) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if err != nil {
		return n, err
	}
	w.restore()
	return n, nil
}

func (w *artRestorer) Read(p []byte) (int, error) {
	if r, ok := w.w.(io.Reader); ok {
		return r.Read(p)
	}
	return 0, io.EOF
}

func (w *artRestorer) Close() error {
	return nil
}

func (w *artRestorer) Fd() uintptr {
	if w.fd != 0 {
		return w.fd
	}
	return os.Stdout.Fd()
}

func (w *artRestorer) restore() {
	if w == nil || w.frames == nil || w.restoring {
		return
	}
	w.frames.mu.Lock()
	if w.frames.quitting {
		w.frames.mu.Unlock()
		return
	}
	row, col := w.frames.artRow, w.frames.artCol
	seq := w.frames.artPlace
	if w.frames.artDirty {
		seq = w.frames.artSeq
		w.frames.artDirty = false
	}
	w.frames.mu.Unlock()
	if seq == "" || row < 1 || col < 1 {
		return
	}
	w.restoring = true
	vizOutMu.Lock()
	_, _ = w.w.Write([]byte(overlayArt("", seq, row, col)))
	vizOutMu.Unlock()
	w.restoring = false
}

func fetchArtURL(url string) (image.Image, error) {
	body, err := art.FetchImage(url)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	return img, err
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

func boundImage(src image.Image, maxPx int) image.Image {
	if src == nil || maxPx < 1 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxPx && h <= maxPx {
		return src
	}
	if w >= h {
		return scaleImage(src, maxPx, max(1, h*maxPx/w))
	}
	return scaleImage(src, max(1, w*maxPx/h), maxPx)
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
