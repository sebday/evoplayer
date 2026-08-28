package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFolderKey(t *testing.T) {
	cases := map[string]string{
		"Drum&Bass":   "drumandbass",
		"drum&bass":   "drumandbass",
		"Drum & Bass": "drumandbass",
		"Hip-Hop":     "hiphop",
		"hiphop":      "hiphop",
		"dubstep":     "dubstep",
		"Dubstep":     "dubstep",
	}
	for in, want := range cases {
		if got := folderKey(in); got != want {
			t.Fatalf("folderKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRelocatePathMapsTopFolder(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "Drum&Bass", "mixes", "set.mp3")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(root, "drum&bass", "mixes", "set.mp3")
	got := RelocatePath(root, old)
	if got != dest {
		t.Fatalf("got %q, want %q", got, dest)
	}
}

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
