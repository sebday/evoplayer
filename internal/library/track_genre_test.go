package library

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTagGenreForPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "track.mp3")
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05",
		"-metadata", "genre=Hip-Hop", "-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	got := tagGenreForPath(path, "hiphop")
	if got != "Hip-Hop" {
		t.Fatalf("tagGenreForPath = %q, want Hip-Hop", got)
	}
	if tagGenreForPath(path, "hiphop") == "hiphop" && got == "" {
		t.Fatal("expected id3 genre")
	}
}

func TestTagGenreForPathFallback(t *testing.T) {
	got := tagGenreForPath("/nonexistent/track.mp3", "dubstep")
	if got != "dubstep" {
		t.Fatalf("fallback = %q", got)
	}
}
