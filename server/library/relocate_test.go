package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRelocateLibraryPathsRewritesLikes(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	dest := filepath.Join(root, "Dubstep", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "dubstep", "track.mp3")
	likesPath := filepath.Join(state, "likes.json")
	raw, _ := json.Marshal(map[string]any{old: map[string]string{"title": "T"}})
	if err := os.WriteFile(likesPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: root, StateDir: state, LikesFile: likesPath}
	if err := RelocateLibraryPaths(env); err != nil {
		t.Fatal(err)
	}
	gotRaw, err := os.ReadFile(likesPath)
	if err != nil {
		t.Fatal(err)
	}
	var likes map[string]any
	if err := json.Unmarshal(gotRaw, &likes); err != nil {
		t.Fatal(err)
	}
	if _, ok := likes[dest]; !ok {
		t.Fatalf("likes = %#v, want key %q", likes, dest)
	}
	if _, ok := likes[old]; ok {
		t.Fatal("old like path should be gone")
	}
}

func TestRelocateLibraryPathsRewritesRootCase(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "music")
	dest := filepath.Join(root, "dubstep", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(parent, "Music", "Dubstep", "track.mp3")
	likesPath := filepath.Join(t.TempDir(), "likes.json")
	raw, _ := json.Marshal(map[string]any{old: map[string]string{"title": "T"}})
	if err := os.WriteFile(likesPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: root, StateDir: t.TempDir(), LikesFile: likesPath}
	if err := RelocateLibraryPaths(env); err != nil {
		t.Fatal(err)
	}
	gotRaw, err := os.ReadFile(likesPath)
	if err != nil {
		t.Fatal(err)
	}
	var likes map[string]any
	if err := json.Unmarshal(gotRaw, &likes); err != nil {
		t.Fatal(err)
	}
	if _, ok := likes[dest]; !ok {
		t.Fatalf("likes = %#v, want key %q", likes, dest)
	}
}

func TestRelocateLibraryPathsUniqueBasename(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dubstep", "vinyl", "cont", "a-mrk1-never_warned-xtc.mp3")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "Dubstep", "vinyl", "con", "a-mrk1-never_warned-xtc.mp3")
	likesPath := filepath.Join(t.TempDir(), "likes.json")
	raw, _ := json.Marshal(map[string]any{old: map[string]string{"title": "T"}})
	if err := os.WriteFile(likesPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: root, StateDir: t.TempDir(), LikesFile: likesPath}
	if err := RelocateLibraryPaths(env); err != nil {
		t.Fatal(err)
	}
	gotRaw, err := os.ReadFile(likesPath)
	if err != nil {
		t.Fatal(err)
	}
	var likes map[string]any
	if err := json.Unmarshal(gotRaw, &likes); err != nil {
		t.Fatal(err)
	}
	if _, ok := likes[dest]; !ok {
		t.Fatalf("likes = %#v, want key %q", likes, dest)
	}
}
