package tags

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bogem/id3v2"
)

func TestYearFromText(t *testing.T) {
	if got := YearFromText("2024-08-23 Artist - Title"); got != 2024 {
		t.Fatalf("YearFromText date prefix = %d", got)
	}
	if got := YearFromText("Artist - 15.08.2024 Mix"); got != 2024 {
		t.Fatalf("YearFromText dotted date = %d", got)
	}
}

func TestEmbedMP3Picture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp3")
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05", "-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	if st, err := os.Stat(path); err != nil || st.Size() < 100 {
		t.Skip("ffmpeg did not produce mp3")
	}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00}
	if err := EmbedMP3(path, map[string]string{"title": "t", "artist": "a"}, png, "image/png"); err != nil {
		t.Fatal(err)
	}
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tag.Close()
	if tag.Title() != "t" || tag.Artist() != "a" {
		t.Fatalf("tags = %q / %q", tag.Title(), tag.Artist())
	}
	pics := tag.GetFrames("APIC")
	if len(pics) == 0 {
		t.Fatal("expected APIC frame")
	}
	_ = os.Remove(path)
}
