package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/playback"
)

func testEnv(t *testing.T) paths.Env {
	t.Helper()
	root := t.TempDir()
	music := filepath.Join(root, "music")
	cache := filepath.Join(root, "cache")
	state := filepath.Join(root, "state", "panel", "player")
	for _, dir := range []string{music, cache, state} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return paths.Env{
		MusicRoot:      music,
		StateDir:       state,
		CacheDir:       cache,
		TracksCacheDir: filepath.Join(cache, "tracks"),
		WaveformDir:    filepath.Join(cache, "waveforms"),
		ArtDir:         filepath.Join(cache, "art"),
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
	}
}

func TestEnrichFilenameMeta(t *testing.T) {
	env := testEnv(t)
	track := filepath.Join(env.MusicRoot, "grime", "2008", "loefah-midnight.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := playback.Status{
		Path:  track,
		Title: "loefah-midnight",
		State: "playing",
	}
	enriched := Enrich(env, raw)
	if enriched.Title == raw.Title {
		t.Fatalf("title still raw %q", enriched.Title)
	}
	if enriched.Title != "midnight" || enriched.Artist != "loefah" {
		t.Fatalf("got title=%q artist=%q", enriched.Title, enriched.Artist)
	}
}

func TestEnrichFolderArt(t *testing.T) {
	env := testEnv(t)
	track := filepath.Join(env.MusicRoot, "grime", "2008", "loefah-midnight.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	art := filepath.Join(env.ArtDir, "grime_2008.jpg")
	if err := os.MkdirAll(env.ArtDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := playback.Status{Path: track, Title: "loefah-midnight", State: "playing"}
	enriched := Enrich(env, raw)
	if enriched.Art == "" {
		t.Fatalf("expected folder art, got %#v", enriched)
	}
}
