package warm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
)

func TestTrackAssetsUsesCachedArt(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	cache := filepath.Join(dir, "cache")
	artDir := filepath.Join(cache, "art")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	track := filepath.Join(album, "a.mp3")
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	art := filepath.Join(artDir, "grime_2008.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := paths.Env{
		MusicRoot: music,
		CacheDir:  cache,
		ArtDir:    artDir,
	}
	res, err := TrackAssets(env, track)
	if err != nil {
		t.Fatal(err)
	}
	if res.Art == "" {
		t.Fatal("expected cached art path")
	}
}
