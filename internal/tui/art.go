package tui

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
		Foreground(colBorder).
		Background(lipgloss.Color("#1F2937")).
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
			sb.WriteString(logoColor().NewStyle().
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
