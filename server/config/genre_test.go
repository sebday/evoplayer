package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/config"
)

func TestGenreConfigFromToml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	if err := os.WriteFile(path, []byte(`
[genres]
drumandbass = "drum&bass"
garage = "garage"

[genre_aliases]
Jungle = "drumandbass"

[playlist_folders]
trap = "electronic"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	aliases, err := config.GenreAliases(path)
	if err != nil {
		t.Fatal(err)
	}
	if aliases["jungle"] != config.GenreDrumAndBass {
		t.Fatalf("jungle alias = %q", aliases["jungle"])
	}
	if aliases["dnb"] != config.GenreDrumAndBass {
		t.Fatalf("default dnb alias should remain")
	}

	folders, err := config.GenreFolders(path)
	if err != nil {
		t.Fatal(err)
	}
	if folders[config.GenreDrumAndBass] != "drum&bass" {
		t.Fatalf("folder map = %#v", folders)
	}
	if config.PlaylistFolder(path, "trap") != "electronic" {
		t.Fatalf("playlist folder map missing trap")
	}
}

func TestMigrateMusicConfigCopiesLegacy(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state", "evoplayer")
	configHome := filepath.Join(dir, "config")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", configHome)
	legacy := filepath.Join(state, "music.toml")
	if err := os.WriteFile(legacy, []byte("[paths]\nroot = \"/music\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.MigrateMusicConfig(state); err != nil {
		t.Fatal(err)
	}
	newPath := config.MusicConfigPath()
	raw, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "/music") {
		t.Fatalf("migrated config missing root: %s", raw)
	}
	if !strings.Contains(string(raw), "[genre_aliases]") {
		t.Fatalf("migrated config should seed genre_aliases: %s", raw)
	}
}

func TestMusicConfigPathUsesConfigHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want := filepath.Join(dir, "evoplayer", "music.toml")
	if got := config.MusicConfigPath(); got != want {
		t.Fatalf("MusicConfigPath = %q want %q", got, want)
	}
}
