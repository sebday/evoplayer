package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureArtResolvesPerTrackCache(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	track := filepath.Join(album, "a.mp3")
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	trackArt := artPathLegacy(env, track)
	if err := os.WriteFile(trackArt, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	art, built, err := EnsureArt(env, track)
	if err != nil {
		t.Fatal(err)
	}
	if art != trackArt {
		t.Fatalf("art = %q, want %q", art, trackArt)
	}
	if built {
		t.Fatal("expected built false for per-track cache hit")
	}
}
