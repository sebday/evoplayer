package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/config"
)

func TestLoadUsesTomlMusicRoot(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	state := filepath.Join(dir, "state")
	configHome := filepath.Join(dir, "config")
	root := filepath.Join(dir, "library")
	for _, p := range []string{home, state, configHome, root} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("EVO_PLAYER_MUSIC_STATE", state)
	t.Setenv("EVO_PLAYER_MUSIC_CACHE", filepath.Join(dir, "cache"))
	if err := config.Set(config.MusicConfigPath(), "paths", "root", root); err != nil {
		t.Fatal(err)
	}
	env := Load("")
	if env.MusicRoot != root {
		t.Fatalf("MusicRoot: got %q want %q", env.MusicRoot, root)
	}
	if env.MusicConfig != config.MusicConfigPath() {
		t.Fatalf("MusicConfig: got %q want %q", env.MusicConfig, config.MusicConfigPath())
	}
}

func TestLoadFallsBackToHomeMusic(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	state := filepath.Join(dir, "state")
	configHome := filepath.Join(dir, "config")
	music := filepath.Join(home, "music")
	for _, p := range []string{home, state, configHome, music} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("EVO_PLAYER_MUSIC_STATE", state)
	t.Setenv("EVO_PLAYER_MUSIC_CACHE", filepath.Join(dir, "cache"))
	env := Load("")
	if env.MusicRoot != music {
		t.Fatalf("MusicRoot: got %q want %q", env.MusicRoot, music)
	}
}

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
