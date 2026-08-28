package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtCacheFindUsesAlbumFolderKey(t *testing.T) {
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
	art := filepath.Join(artDir, "dnb_album.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	got := artCacheFind(env, track)
	if got != art {
		t.Fatalf("artCacheFind = %q, want %q", got, art)
	}
}
