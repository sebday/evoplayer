package config

import (
	"os"
	"path/filepath"
)

// MusicConfigPath returns the primary music.toml location (~/.config/evoplayer/music.toml).
func MusicConfigPath() string {
	if v := os.Getenv("EVOPLAYER_MUSIC_CONFIG"); v != "" {
		return v
	}
	return filepath.Join(xdgConfigHome(), "evoplayer", "music.toml")
}

// LegacyMusicConfigPath returns the pre-migration state-dir music.toml path.
func LegacyMusicConfigPath(stateDir string) string {
	if stateDir == "" {
		return ""
	}
	return filepath.Join(stateDir, "music.toml")
}

// MigrateMusicConfig copies legacy state-dir music.toml into the config path when needed.
func MigrateMusicConfig(stateDir string) error {
	configPath := MusicConfigPath()
	legacyPath := LegacyMusicConfigPath(stateDir)
	if legacyPath == "" {
		return ensureMusicConfig(configPath)
	}
	if _, err := os.Stat(configPath); err == nil {
		return SeedGenreConfig(configPath)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		return ensureMusicConfig(configPath)
	}
	raw, err := os.ReadFile(legacyPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		return err
	}
	return SeedGenreConfig(configPath)
}

func ensureMusicConfig(configPath string) error {
	if _, err := os.Stat(configPath); err == nil {
		return SeedGenreConfig(configPath)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(configPath, []byte{}, 0o644); err != nil {
		return err
	}
	return SeedGenreConfig(configPath)
}

func xdgConfigHome() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}
