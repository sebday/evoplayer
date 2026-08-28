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
	if err := os.MkdirAll(filepath.Join(root, "misc"), 0o755); err != nil {
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

func TestIncomingDestYouTubeMixGoesToMixes(t *testing.T) {
	root := t.TempDir()
	env := Env{MusicRoot: root}
	incoming := filepath.Join(root, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "Drum & Bass"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(incoming, "DJ Hype - FABRICLIVE Mix.mp3")
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05", "-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	if err := tags.EmbedMP3(path, map[string]string{
		"artist":  "DJ Hype",
		"title":   "FABRICLIVE Mix",
		"genre":   "Drum & Bass",
		"comment": "source:youtube",
		"year":    "2026",
	}, nil, ""); err != nil {
		t.Fatal(err)
	}
	probed, _ := tags.Probe(path)
	probed.Duration = MixMinDurationSec
	dest, err := incomingDest(env, path, probed)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "Drum & Bass", "mixes", "2026")
	if filepath.Dir(dest) != want {
		t.Fatalf("dir = %q, want %q", filepath.Dir(dest), want)
	}
}

func TestIncomingDestShortYouTubeStaysYouTube(t *testing.T) {
	root := t.TempDir()
	env := Env{MusicRoot: root}
	incoming := filepath.Join(root, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "misc"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(incoming, "Channel - Tune.mp3")
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05", "-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	if err := tags.EmbedMP3(path, map[string]string{
		"artist":  "Channel",
		"title":   "Tune",
		"genre":   "misc",
		"comment": "source:youtube",
		"year":    "2024",
	}, nil, ""); err != nil {
		t.Fatal(err)
	}
	probed, _ := tags.Probe(path)
	probed.Duration = float64(MixMinDurationSec - 1)
	dest, err := incomingDest(env, path, probed)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "misc", "youtube", "2024")
	if filepath.Dir(dest) != want {
		t.Fatalf("dir = %q, want %q", filepath.Dir(dest), want)
	}
}
