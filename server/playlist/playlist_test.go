package playlist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateUserName(t *testing.T) {
	root := t.TempDir()
	env := Env{
		MusicRoot:   root,
		PlaylistDir: filepath.Join(root, "playlists"),
	}
	if err := validateUserName(env, ""); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := validateUserName(env, "all"); err == nil {
		t.Fatal("expected error for reserved name")
	}
	if err := validateUserName(env, "mixes"); err == nil {
		t.Fatal("expected error for reserved mixes name")
	}
	if err := validateUserName(env, "My Mix"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
}

func TestWriteAndReadM3U(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.m3u")
	trackA := filepath.Join(dir, "a.flac")
	trackB := filepath.Join(dir, "b.mp3")
	for _, p := range []string{trackA, trackB} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths := []string{trackA, trackB}
	if err := writeM3U(path, paths); err != nil {
		t.Fatal(err)
	}
	got, err := readM3UPaths(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != paths[0] {
		t.Fatalf("unexpected paths: %#v", got)
	}
}

func TestCreateUserPlaylist(t *testing.T) {
	root := t.TempDir()
	env := Env{
		PlaylistDir: filepath.Join(root, "playlists"),
		MusicRoot:   filepath.Join(root, "music"),
	}
	if err := os.MkdirAll(env.MusicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateUser(env, "road trip"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(env.PlaylistDir, "road trip.m3u")); err != nil {
		t.Fatal(err)
	}
}
