package youtube

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestSquareImageCropsWidescreenCenter(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			if x < 100 {
				src.Set(x, y, color.RGBA{R: 200, A: 255})
			} else {
				src.Set(x, y, color.RGBA{B: 200, A: 255})
			}
		}
	}
	out := squareImage(src)
	if out == nil {
		t.Fatal("expected a crop")
	}
	b := out.Bounds()
	if b.Dx() != 100 || b.Dy() != 100 {
		t.Fatalf("size = %dx%d", b.Dx(), b.Dy())
	}
	if r, _, _, _ := out.At(0, 50).RGBA(); r>>8 < 100 {
		t.Fatal("left edge should stay in the red half")
	}
	if _, _, bl, _ := out.At(99, 50).RGBA(); bl>>8 < 100 {
		t.Fatal("right edge should stay in the blue half")
	}
}

func TestSquareImageDropsLetterbox(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 160, 120))
	white := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	for y := 30; y < 90; y++ {
		for x := 0; x < 160; x++ {
			src.Set(x, y, white)
		}
	}
	out := squareImage(src)
	if out == nil {
		t.Fatal("expected a crop")
	}
	b := out.Bounds()
	if b.Dx() != 60 || b.Dy() != 60 {
		t.Fatalf("size = %dx%d", b.Dx(), b.Dy())
	}
	for y := 0; y < 60; y += 10 {
		r, g, b, _ := out.At(30, y).RGBA()
		if r>>8 < 200 || g>>8 < 200 || b>>8 < 200 {
			t.Fatalf("pixel y=%d is not the frame", y)
		}
	}
}

func TestYtDownloadDetail(t *testing.T) {
	line := "[download]  45.2% of 12.34MiB at 1.23MiB/s ETA 00:08"
	detail, pct, ok := ytDownloadDetail(line, "DJ Hype — Rinse FM")
	if !ok || pct != 45 {
		t.Fatalf("ok=%v pct=%d", ok, pct)
	}
	want := "downloading 45% of 12.34MiB at 1.23MiB/s, 00:08 left · DJ Hype — Rinse FM"
	if detail != want {
		t.Fatalf("detail = %q", detail)
	}
	if _, _, ok := ytDownloadDetail("ERROR: no", "x"); ok {
		t.Fatal("error line reported as progress")
	}
}

func TestSquareCoverKeepsSquareJPEG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			src.Set(x, y, color.RGBA{G: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	orig := append([]byte(nil), buf.Bytes()...)
	got, mime := squareCover(orig)
	if mime != "image/jpeg" {
		t.Fatalf("mime = %q", mime)
	}
	if !bytes.Equal(got, orig) {
		t.Fatal("already-square cover was re-encoded")
	}
}
