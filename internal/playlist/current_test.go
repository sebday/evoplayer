package playlist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/library"
)

func TestSaveAndReadCurrentPaths(t *testing.T) {
	root := t.TempDir()
	env := Env{
		Env: libraryEnv(root),
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

func TestSaveCurrentFastSkipsUnchangedM3U(t *testing.T) {
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
	paths := []string{trackA, trackB}
	if err := SaveCurrent(env, paths); err != nil {
		t.Fatal(err)
	}
	changed, err := SaveCurrentFast(env, paths)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected unchanged M3U to skip write")
	}
}

func TestReadCurrentPathsMigratesLegacy(t *testing.T) {
	root := t.TempDir()
	env := Env{
		Env:         libraryEnv(root),
		PlaylistDir: filepath.Join(root, "state", "playlists"),
		MusicRoot:   filepath.Join(root, "music"),
	}
	track := filepath.Join(root, "music", "a.flac")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(root, "state", "current.m3u")
	if err := writeM3U(legacy, []string{track}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadCurrentPaths(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != track {
		t.Fatalf("unexpected migrated paths: %#v", got)
	}
	if _, err := os.Stat(env.currentM3U()); err != nil {
		t.Fatalf("current m3u not migrated: %v", err)
	}
}

func TestNextSiblingFolder(t *testing.T) {
	root := t.TempDir()
	music := filepath.Join(root, "music", "genre")
	dirs := []string{
		filepath.Join(music, "album-a"),
		filepath.Join(music, "album-b"),
		filepath.Join(music, "album-c"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{
		Env:         libraryEnv(root),
		PlaylistDir: filepath.Join(root, "state", "playlists"),
		MusicRoot:   filepath.Join(root, "music"),
	}
	next, err := nextSiblingFolder(env, dirs[0])
	if err != nil {
		t.Fatal(err)
	}
	if next != dirs[1] {
		t.Fatalf("expected %s, got %s", dirs[1], next)
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

func TestLoadCurrentReusesJSONWhenM3UNewer(t *testing.T) {
	root := t.TempDir()
	env := Env{
		Env:         libraryEnv(root),
		PlaylistDir: filepath.Join(root, "state", "playlists"),
		MusicRoot:   filepath.Join(root, "music"),
	}
	track := filepath.Join(root, "music", "a.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveCurrent(env, []string{track}); err != nil {
		t.Fatal(err)
	}
	cachePath := env.currentTracksJSON()
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	var items []library.Track
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	items[0].Title = "Pinned Title"
	items[0].Artist = "Pinned Artist"
	baked, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, baked, 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(env.currentM3U(), later, later); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCurrent(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Pinned Title" || got[0].Artist != "Pinned Artist" {
		t.Fatalf("wanted pinned json meta, got %#v", got)
	}
}

func libraryEnv(root string) library.Env {
	return library.Env{
		StateDir: filepath.Join(root, "state"),
	}
}
