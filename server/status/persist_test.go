package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

func TestWriteSkipsEmptyPath(t *testing.T) {
	dir := t.TempDir()
	env := paths.Env{PlayerState: filepath.Join(dir, "player.json")}
	if err := Write(env, playback.Status{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(env.PlayerState); !os.IsNotExist(err) {
		t.Fatal("empty path should not create player.json")
	}
}

func TestWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	env := paths.Env{PlayerState: filepath.Join(dir, "player.json")}
	path := filepath.Join(dir, "track.mp3")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := playback.Status{
		Path:     path,
		Title:    "Track",
		Artist:   "Artist",
		Genre:    "Electronic",
		Playlist: "current",
		Position: 12.5,
	}
	if err := Write(env, st); err != nil {
		t.Fatal(err)
	}
	got := Saved(env)
	if got.Path != path || got.Position != 12.5 {
		t.Fatalf("got path=%q pos=%v", got.Path, got.Position)
	}
}
