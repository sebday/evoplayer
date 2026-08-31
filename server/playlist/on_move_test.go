package playlist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestOnTrackMovedRewritesLikesAndCurrent(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	from := filepath.Join(root, "garage", "soundcloud", "a.mp3")
	to := filepath.Join(root, "grime", "soundcloud", "a.mp3")
	for _, p := range []string{from, to} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(from, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	likes := filepath.Join(state, "likes.json")
	raw, _ := json.Marshal(map[string]any{from: map[string]string{"title": "a"}})
	if err := os.WriteFile(likes, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{
		MusicRoot:   root,
		PlaylistDir: filepath.Join(state, "playlists"),
		Env: library.Env{
			MusicRoot: root,
			StateDir:  state,
			LikesFile: likes,
		},
	}
	if err := os.MkdirAll(env.PlaylistDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveCurrent(env, []string{from, "/other.mp3"}); err != nil {
		t.Fatal(err)
	}
	if err := OnTrackMoved(env, from, to); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(likes)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if strings.Contains(text, from) {
		t.Fatalf("likes not rewritten: %s", text)
	}
	if !strings.Contains(text, to) {
		t.Fatalf("likes missing dest: %s", text)
	}
	paths, err := ReadCurrentPaths(env)
	if err != nil {
		t.Fatal(err)
	}
	if paths[0] != to {
		t.Fatalf("current=%v", paths)
	}
}
