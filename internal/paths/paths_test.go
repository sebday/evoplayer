package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirsCreatesCacheSubdirs(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state")
	cache := filepath.Join(dir, "cache")
	env := Env{
		StateDir:       state,
		CacheDir:       cache,
		PlaylistDir:    filepath.Join(state, "playlists"),
		ArtDir:         filepath.Join(cache, "art"),
		WaveformDir:    filepath.Join(cache, "waveforms"),
		TracksCacheDir: filepath.Join(cache, "tracks"),
	}
	if err := env.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	for _, path := range []string{
		state,
		cache,
		env.PlaylistDir,
		env.ArtDir,
		env.WaveformDir,
		env.TracksCacheDir,
	} {
		if st, err := os.Stat(path); err != nil || !st.IsDir() {
			t.Fatalf("expected dir %q, stat err=%v", path, err)
		}
	}
}
