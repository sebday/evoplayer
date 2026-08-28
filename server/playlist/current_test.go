package playlist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestSaveAndReadCurrentPaths(t *testing.T) {
	root := t.TempDir()
	env := Env{
		Env:         libraryEnv(root),
		PlaylistDir: filepath.Join(root, "state", "playlists"),
		MusicRoot:   filepath.Join(root, "music"),
	}
	trackA := filepath.Join(root, "music", "a.flac")
	trackB := filepath.Join(root, "music", "b.mp3")
	if err := os.MkdirAll(filepath.Dir(trackA), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{trackA, trackB} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := SaveCurrent(env, []string{trackA, trackB}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadCurrentPaths(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != trackA || got[1] != trackB {
		t.Fatalf("unexpected paths: %#v", got)
	}
	if _, err := os.Stat(env.currentTracksJSON()); err != nil {
		t.Fatalf("tracks json missing: %v", err)
	}
}

func TestExtendCurrent(t *testing.T) {
	root := t.TempDir()
	music := filepath.Join(root, "music", "g")
	dirA := filepath.Join(music, "album-a")
	dirB := filepath.Join(music, "album-b")
	trackA := filepath.Join(dirA, "01.flac")
	trackB := filepath.Join(dirB, "01.flac")
	for _, p := range []string{dirA, dirB} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{trackA, trackB} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{
		Env:         libraryEnv(root),
		PlaylistDir: filepath.Join(root, "state", "playlists"),
		MusicRoot:   filepath.Join(root, "music"),
	}
	if err := SaveCurrent(env, []string{trackA}); err != nil {
		t.Fatal(err)
	}
	result, added, merged, atEnd, err := ExtendCurrent(env, trackA)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || len(added) != 1 || added[0] != trackB {
		t.Fatalf("unexpected extend: %#v %#v", result, added)
	}
	if len(merged) != 2 || merged[1] != trackB {
		t.Fatalf("unexpected merged: %#v", merged)
	}
	if !atEnd {
		t.Fatal("expected atEnd")
	}
}

func libraryEnv(root string) library.Env {
	return library.Env{
		StateDir: filepath.Join(root, "state"),
	}
}
