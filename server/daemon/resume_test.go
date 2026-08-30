package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/status"
)

func TestReadSavedPlayback(t *testing.T) {
	dir := t.TempDir()
	env := paths.Env{PlayerState: filepath.Join(dir, "player.json")}
	if _, _, ok := readSavedPlayback(env); ok {
		t.Fatal("expected missing file to be false")
	}

	path := filepath.Join(dir, "track.mp3")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw := `{"path":"` + path + `","position":12.5}`
	if err := os.WriteFile(env.PlayerState, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	gotPath, gotPos, ok := readSavedPlayback(env)
	if !ok {
		t.Fatal("expected saved playback")
	}
	if gotPath != path || gotPos != 12.5 {
		t.Fatalf("got path=%q pos=%v", gotPath, gotPos)
	}
}

func TestWriteThenReadSavedPlayback(t *testing.T) {
	dir := t.TempDir()
	env := paths.Env{PlayerState: filepath.Join(dir, "player.json")}
	path := filepath.Join(dir, "later.mp3")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := status.Write(env, playback.Status{Path: path, Position: 7}); err != nil {
		t.Fatal(err)
	}
	gotPath, gotPos, ok := readSavedPlayback(env)
	if !ok || gotPath != path || gotPos != 7 {
		t.Fatalf("got ok=%v path=%q pos=%v", ok, gotPath, gotPos)
	}
}
