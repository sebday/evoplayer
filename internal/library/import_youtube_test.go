package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/tags"
)

func TestIncomingDestYouTube(t *testing.T) {
	root := t.TempDir()
	env := Env{MusicRoot: root}
	incoming := filepath.Join(root, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(incoming, "Channel - 2024-08-23 Title.mp3")
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05", "-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	if err := tags.EmbedMP3(path, map[string]string{
		"artist":  "Channel",
		"title":   "2024-08-23 Title",
		"genre":   "misc",
		"comment": "source:youtube",
		"year":    "2024",
	}, nil, ""); err != nil {
		t.Fatal(err)
	}
	probed, _ := tags.Probe(path)
	dest, err := incomingDest(env, path, probed)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "misc", "youtube", "2024")
	if filepath.Dir(dest) != want {
		t.Fatalf("dir = %q, want %q", filepath.Dir(dest), want)
	}
}
