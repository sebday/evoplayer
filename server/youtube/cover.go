package youtube

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"

	"github.com/sebday/evoplayer/server/tags"
)

// squareCover center-crops a YouTube thumbnail to a square cover.
// Letterbox bars are trimmed first so a 16:9 frame does not keep black strips.
func squareCover(data []byte) ([]byte, string) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, tags.PictureMIME(data)
	}
	squared := squareImage(img)
	if squared == nil {
		return data, tags.PictureMIME(data)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, squared, &jpeg.Options{Quality: 92}); err != nil {
		return data, tags.PictureMIME(data)
	}
	return buf.Bytes(), "image/jpeg"
}

func squareImage(img image.Image) image.Image {
	b := trimBars(img)
	w, h := b.Dx(), b.Dy()
	if w < 8 || h < 8 {
		b = img.Bounds()
		w, h = b.Dx(), b.Dy()
	}
	side := w
	if h < side {
		side = h
	}
	if side < 8 {
		return nil
	}
	if w == h && b.Eq(img.Bounds()) {
		return nil
	}
	x0 := b.Min.X + (w-side)/2
	y0 := b.Min.Y + (h-side)/2
	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(dst, dst.Bounds(), img, image.Point{X: x0, Y: y0}, draw.Src)
	return dst
}

func trimBars(img image.Image) image.Rectangle {
	b := img.Bounds()
	minX, maxX := b.Min.X, b.Max.X
	minY, maxY := b.Min.Y, b.Max.Y
	xLimit := b.Dx() / 4
	yLimit := b.Dy() / 4
	for minY < maxY && minY-b.Min.Y < yLimit && barRow(img, minY, b.Min.X, b.Max.X) {
		minY++
	}
	for maxY > minY && b.Max.Y-maxY < yLimit && barRow(img, maxY-1, b.Min.X, b.Max.X) {
		maxY--
	}
	for minX < maxX && minX-b.Min.X < xLimit && barCol(img, minX, minY, maxY) {
		minX++
	}
	for maxX > minX && b.Max.X-maxX < xLimit && barCol(img, maxX-1, minY, maxY) {
		maxX--
	}
	if maxX-minX < 8 || maxY-minY < 8 {
		return b
	}
	return image.Rect(minX, minY, maxX, maxY)
}

func barRow(img image.Image, y, minX, maxX int) bool {
	return mostlyDark(minX, maxX, func(x int) color.Color { return img.At(x, y) })
}

func barCol(img image.Image, x, minY, maxY int) bool {
	return mostlyDark(minY, maxY, func(y int) color.Color { return img.At(x, y) })
}

func mostlyDark(from, to int, at func(int) color.Color) bool {
	span := to - from
	if span < 4 {
		return false
	}
	step := span / 40
	if step < 1 {
		step = 1
	}
	dark, n := 0, 0
	for i := from; i < to; i += step {
		n++
		if darkPixel(at(i)) {
			dark++
		}
	}
	return n >= 4 && dark*100/n >= 92
}

func darkPixel(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 < 18 && g>>8 < 18 && b>>8 < 18
}
