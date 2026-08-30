package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureArtResolvesFolderAlias(t *testing.T) {
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
	folderArt := artPathFolder(env, track)
	if err := os.WriteFile(folderArt, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	art, built, err := EnsureArt(env, track)
	if err != nil {
		t.Fatal(err)
	}
	if art != folderArt {
		t.Fatalf("art = %q, want %q", art, folderArt)
	}
	if built {
		t.Fatal("expected built false for folder alias hit")
	}
}
