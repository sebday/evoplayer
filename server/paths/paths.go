package paths

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/server/config"
)

type Env struct {
	MusicRoot       string
	StateDir        string
	CacheDir        string
	MusicConfig     string
	PlayerState     string
	PlaylistDir     string
	SocketPath      string
	DaemonLock      string
	RepoRoot        string
	DisplayArtDir   string
	LikesFile       string
	TracksCacheDir  string
	WaveformDir     string
	ArtDir          string
	LibraryDB       string
	ScrobbleLog     string
	ScrobblePending string
	DaemonLog       string
}

func Load(repoRoot string) Env {
	configPath := config.MusicConfigPath()
	_ = config.EnsureMusicConfig()
	root := config.ResolveRoot(configPath)
	state := os.Getenv("EVO_PLAYER_MUSIC_STATE")
	if state == "" && root != "" {
		// Keep likes, playlists, and the queue on the music disk so a
		// reinstall of the home directory does not wipe them.
		state = filepath.Join(root, ".evoplayer")
	}
	if state == "" {
		state = legacyStateDir()
	}
	cache := os.Getenv("EVO_PLAYER_MUSIC_CACHE")
	if cache == "" {
		cache = filepath.Join(xdgCache(), "evoplayer")
	}
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = "/tmp"
	}
	socket := os.Getenv("EVOPLAYER_SOCKET")
	if socket == "" {
		socket = filepath.Join(runtime, "evoplayer.sock")
	}
	repo := repoRoot
	if repo == "" {
		repo = "."
	}
	return Env{
		MusicRoot:       root,
		StateDir:        state,
		CacheDir:        cache,
		MusicConfig:     configPath,
		PlayerState:     filepath.Join(state, "player.json"),
		PlaylistDir:     filepath.Join(state, "playlists"),
		SocketPath:      socket,
		DaemonLock:      filepath.Join(state, "daemon.lock"),
		RepoRoot:        repo,
		DisplayArtDir:   filepath.Join(cache, "display-art"),
		LikesFile:       filepath.Join(state, "likes.json"),
		TracksCacheDir:  filepath.Join(cache, "tracks"),
		WaveformDir:     filepath.Join(cache, "waveforms"),
		ArtDir:          filepath.Join(cache, "art"),
		LibraryDB:       filepath.Join(cache, "library.sqlite3"),
		ScrobbleLog:     filepath.Join(state, "scrobble.jsonl"),
		ScrobblePending: filepath.Join(state, "scrobble-pending.json"),
		DaemonLog:       filepath.Join(state, "daemon.log"),
	}
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
	return migrateLegacyState(e.StateDir)
}

func legacyStateDir() string {
	return filepath.Join(xdgState(), "evoplayer")
}

// IncomingDir is where downloads wait before they are filed into the library.
func IncomingDir(musicRoot string) string {
	return filepath.Join(musicRoot, ".evoplayer", "incoming")
}

// migrateLegacyState copies likes, playlists, and the queue out of
// ~/.local/state/evoplayer when that folder still has them and the music
// folder does not. Files already in the music folder are left alone.
func migrateLegacyState(stateDir string) error {
	legacy := legacyStateDir()
	if stateDir == "" || samePath(legacy, stateDir) {
		return nil
	}
	info, err := os.Stat(legacy)
	if err != nil || !info.IsDir() {
		return nil
	}
	for _, name := range []string{
		"likes.json",
		"player.json",
		"playlist-stars.json",
		"scrobble.jsonl",
		"scrobble-pending.json",
		"discover-dismissed.json",
		"sync-archive.txt",
	} {
		if err := copyFileIfMissing(filepath.Join(legacy, name), filepath.Join(stateDir, name)); err != nil {
			return err
		}
	}
	return copyDirFilesIfDestEmpty(filepath.Join(legacy, "playlists"), filepath.Join(stateDir, "playlists"))
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return aa == bb
}

func copyFileIfMissing(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}

func copyDirFilesIfDestEmpty(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	existing, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := copyFileIfMissing(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
