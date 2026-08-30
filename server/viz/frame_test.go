package viz

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSpectrumFrameRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spectrum")
	in := []float32{0, 0.25, 0.5, 1}
	if err := WriteFrame(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := ReadFrame(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("len = %d", len(out))
	}
	for i := range in {
		if out[i] != float64(in[i]) {
			t.Fatalf("out[%d] = %v want %v", i, out[i], in[i])
		}
	}
	if _, err := ReadFrame(path + ".missing"); !os.IsNotExist(err) {
		t.Fatalf("missing = %v", err)
	}

	w := NewFrameWriter(path + ".reuse")
	defer w.Close()
	if err := w.Write(in); err != nil {
		t.Fatal(err)
	}
	second := []float32{0, 0.25, 0.75, 1}
	if err := w.Write(second); err != nil {
		t.Fatal(err)
	}
	var r FrameReader
	got, err := r.Read(path + ".reuse")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(second) || got[2] != float64(second[2]) {
		t.Fatalf("reuse read = %v", got)
	}
}

func TestFramePath(t *testing.T) {
	if got := FramePath("/run/user/1000/evoplayer.sock"); got != "/run/user/1000/evoplayer.sock.spectrum" {
		t.Fatalf("got %q", got)
	}
}
