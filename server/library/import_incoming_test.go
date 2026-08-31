package library

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/tags"
)

func TestIncomingDestSlugMixMarker(t *testing.T) {
	root := t.TempDir()
	env := Env{MusicRoot: root}
	incoming := filepath.Join(root, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "drum&bass"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(incoming, "Amoss - Spring Mix 4 U.mp3")
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05", "-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	if err := tags.EmbedMP3(path, map[string]string{
		"artist":  "Amoss",
		"title":   "Spring Mix 4 U",
		"genre":   "drum&bass",
		"comment": "source:soundcloud",
		"year":    "2018",
	}, nil, ""); err != nil {
		t.Fatal(err)
	}
	probed, _ := tags.ProbeImport(path)
	dest, err := incomingDest(env, path, probed)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "drum&bass", "mixes", "2018")
	if filepath.Dir(dest) != want {
		t.Fatalf("dir = %q, want %q", filepath.Dir(dest), want)
	}
	if filepath.Base(dest) != "amoss-spring_mix_4_u.mp3" {
		t.Fatalf("base = %q", filepath.Base(dest))
	}
}

func TestIncomingDestLongTrackUsesTLEN(t *testing.T) {
	root := t.TempDir()
	env := Env{MusicRoot: root}
	incoming := filepath.Join(root, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "drum&bass"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(incoming, "Atmoteka - Blu Mar Ten.mp3")
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05", "-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	if err := tags.EmbedMP3(path, map[string]string{
		"artist":      "Atmoteka",
		"title":       "Blu Mar Ten",
		"genre":       "drum&bass",
		"comment":     "source:soundcloud",
		"year":        "2017",
		"duration_ms": "5617000",
	}, nil, ""); err != nil {
		t.Fatal(err)
	}
	probed, _ := tags.ProbeImport(path)
	if probed.Duration < MixMinDurationSec {
		t.Fatalf("duration = %v, want >= %d", probed.Duration, MixMinDurationSec)
	}
	dest, err := incomingDest(env, path, probed)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "drum&bass", "mixes", "2017")
	if filepath.Dir(dest) != want {
		t.Fatalf("dir = %q, want %q", filepath.Dir(dest), want)
	}
}
