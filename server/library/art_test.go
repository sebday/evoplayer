package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtCacheFindUsesPerTrackKey(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	track := filepath.Join(music, "dnb", "album", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	perTrack := filepath.Join(artDir, cacheKey(music, track)+".jpg")
	if err := os.WriteFile(perTrack, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	folderArt := filepath.Join(artDir, "dnb_album.jpg")
	if err := os.WriteFile(folderArt, []byte("folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	got := artCacheFind(env, track)
	if got != perTrack {
		t.Fatalf("artCacheFind = %q, want per-track %q", got, perTrack)
	}
}

func TestArtCacheFindIgnoresFolderAlias(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	track := filepath.Join(music, "dnb", "soundcloud", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	folderArt := filepath.Join(artDir, "dnb_soundcloud.jpg")
	if err := os.WriteFile(folderArt, []byte("folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	if got := artCacheFind(env, track); got != "" {
		t.Fatalf("artCacheFind = %q, want empty without per-track art", got)
	}
}
