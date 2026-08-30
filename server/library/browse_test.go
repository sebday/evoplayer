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

func TestPropagateFolderArt(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(album, "a.mp3")
	b := filepath.Join(album, "b.mp3")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	art := filepath.Join(artDir, "grime_2008.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	tracks := []Track{
		{Path: a, Type: "track"},
		{Path: b, Type: "track"},
	}
	propagateFolderArt(env, tracks)
	for i, want := range []string{art, art} {
		if tracks[i].Art != want {
			t.Fatalf("track %d art = %q, want %q", i, tracks[i].Art, want)
		}
		if tracks[i].Thumb != "" {
			t.Fatalf("track %d thumb = %q, want empty (no sync thumb generation)", i, tracks[i].Thumb)
		}
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
