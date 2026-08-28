package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

func TestEnrichFilenameMeta(t *testing.T) {
	root := t.TempDir()
	music := filepath.Join(root, "music")
	cache := filepath.Join(root, "cache")
	state := filepath.Join(root, "state", "panel", "player")
	for _, dir := range []string{music, cache, state} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := paths.Env{
		MusicRoot:      music,
		StateDir:       state,
		CacheDir:       cache,
		TracksCacheDir: filepath.Join(cache, "tracks"),
		ArtDir:         filepath.Join(cache, "art"),
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
	}
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
