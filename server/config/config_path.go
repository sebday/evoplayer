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

// EnsureMusicConfig creates music.toml when missing and seeds genre config.
func EnsureMusicConfig() error {
	return ensureMusicConfig(MusicConfigPath())
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
