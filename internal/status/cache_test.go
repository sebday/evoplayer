package status_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/playback"
	"github.com/sebday/evoplayer/internal/status"
)

func TestEnrichLightPreservesTransportFields(t *testing.T) {
	status.InvalidateAllMeta()
	root := t.TempDir()
	track := filepath.Join(root, "music", "a.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := paths.Env{
		MusicRoot:      filepath.Join(root, "music"),
		StateDir:       filepath.Join(root, "state"),
		CacheDir:       filepath.Join(root, "cache"),
		LikesFile:      filepath.Join(root, "state", "likes.json"),
		TracksCacheDir: filepath.Join(root, "cache", "tracks"),
	}
	full := status.EnrichFull(env, playback.Status{
		Path:  track,
		State: "playing",
	})
	light := status.EnrichLight(env, playback.Status{
		Path:     track,
		State:    "playing",
		Position: 42,
	})
	if light.Position != 42 {
		t.Fatalf("position = %v, want 42", light.Position)
	}
	if light.Title != full.Title {
		t.Fatalf("title = %q, want cached %q", light.Title, full.Title)
	}
}
