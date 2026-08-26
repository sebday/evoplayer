package paths

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const musicRootDefault = "/mnt/external/music"

type Config struct {
	Paths struct {
		Root string `toml:"root"`
	} `toml:"paths"`
}

type Env struct {
	MusicRoot      string
	StateDir       string
	CacheDir       string
	MusicConfig    string
	PlayerState    string
	PlaylistDir    string
	SocketPath     string
	DaemonLock     string
	LegacyRoot     string
	DisplayArtDir  string
	LikesFile      string
	TracksCacheDir string
	WaveformDir    string
	ArtDir         string
	LibraryDB      string
	ScrobbleLog    string
}

func Load(repoRoot string) Env {
	state := os.Getenv("EVO_PLAYER_MUSIC_STATE")
	if state == "" {
		state = filepath.Join(xdgState(), "evoplayer")
	}
	cache := os.Getenv("EVO_PLAYER_MUSIC_CACHE")
	if cache == "" {
		cache = os.Getenv("EVO_PLAYER_CACHE")
		if cache == "" {
			cache = filepath.Join(xdgCache(), "evoplayer")
		}
	}
	root := os.Getenv("EVO_PLAYER_MUSIC_ROOT")
	if root == "" {
		root = os.Getenv("MUSIC_ROOT")
	}
	if root == "" {
		root = readMusicRoot(state, cache)
	}
	if root == "" {
		root = musicRootDefault
	}
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = "/tmp"
	}
	socket := os.Getenv("EVOPLAYER_SOCKET")
	if socket == "" {
		socket = os.Getenv("EVOPLAYER_TEST_SOCK")
	}
	if socket == "" {
		socket = filepath.Join(runtime, "evoplayer.sock")
	}
	legacyRoot := repoRoot
	if legacyRoot == "" {
		legacyRoot = "."
	}
	return Env{
		MusicRoot:     root,
		StateDir:      state,
		CacheDir:      cache,
		MusicConfig:   filepath.Join(state, "music.toml"),
		PlayerState:   filepath.Join(state, "player.json"),
		PlaylistDir:   filepath.Join(state, "playlists"),
		SocketPath:    socket,
		DaemonLock:    filepath.Join(state, "daemon.lock"),
		LegacyRoot:    legacyRoot,
		DisplayArtDir: filepath.Join(xdgCache(), "omarchy", "display-art"),
		LikesFile:     filepath.Join(state, "likes.json"),
		TracksCacheDir: filepath.Join(cache, "tracks"),
		WaveformDir:   filepath.Join(cache, "waveforms"),
		ArtDir:        filepath.Join(cache, "art"),
		LibraryDB:     filepath.Join(cache, "library.sqlite3"),
		ScrobbleLog:   filepath.Join(state, "scrobble.jsonl"),
	}
}

func readMusicRoot(stateDir, cacheDir string) string {
	_ = cacheDir
	if root := parseRoot(filepath.Join(stateDir, "music.toml")); root != "" {
		return root
	}
	return ""
}

func parseRoot(path string) string {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return ""
	}
	return cfg.Paths.Root
}

func xdgState() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state")
}

func xdgCache() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache")
}

func (e Env) EnsureDirs() error {
	for _, dir := range []string{
		e.StateDir,
		e.CacheDir,
		e.PlaylistDir,
		e.ArtDir,
		e.WaveformDir,
		e.TracksCacheDir,
	} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
