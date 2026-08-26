package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtImageHashDisplayArt(t *testing.T) {
	dir := t.TempDir()
	tmp := filepath.Join(dir, "input.jpg")
	if err := os.WriteFile(tmp, []byte("\xff\xd8\xff\xe0\x00\x10JFIF"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := artImageHash(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("artImageHash returned empty hash")
	}
	displayDir := filepath.Join(dir, "display-art")
	if err := os.MkdirAll(displayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(displayDir, hash+".jpg")
	tmpcopy, err := os.CreateTemp(displayDir, filepath.Base(dest)+".*")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := tmpcopy.Name()
	if _, err := tmpcopy.Write([]byte("\xff\xd8\xff\xe0\x00\x10JFIF")); err != nil {
		t.Fatal(err)
	}
	if err := tmpcopy.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("display-art write failed: %v", err)
	}
}

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
