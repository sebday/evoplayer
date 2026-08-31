package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowseEntryTrackPathJSON(t *testing.T) {
	raw, err := json.Marshal(BrowseEntry{
		Type: "track",
		Track: Track{
			Path:  "/music/album/track.mp3",
			Title: "Track",
			Art:   "/music/.cache/art/album.jpg",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out["path"] != "/music/album/track.mp3" {
		t.Fatalf("track path missing from browse json: %#v", out)
	}
	if out["art"] != "/music/.cache/art/album.jpg" {
		t.Fatalf("track art missing from browse json: %#v", out)
	}
}

func TestCollectQueueTracksNestedFolder(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	genre := filepath.Join(music, "grime")
	album := filepath.Join(genre, "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(album, "track.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music}
	tracks, err := collectQueueTracks(env, "grime", genre)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 nested track, got %d", len(tracks))
	}
}
