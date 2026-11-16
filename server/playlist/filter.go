package playlist

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/library"
)

func likedOnlyPath(env Env) string {
	return filepath.Join(env.StateDir, "playlist-liked-only")
}

// LikedOnly reports whether tag, artist, and album playlists list liked tracks only.
func LikedOnly(env Env) bool {
	raw, err := os.ReadFile(likedOnlyPath(env))
	if err != nil {
		return false
	}
	switch strings.TrimSpace(string(raw)) {
	case "1", "true":
		return true
	default:
		return false
	}
}

// SetLikedOnly stores the playlist filter. False lists every track.
func SetLikedOnly(env Env, on bool) error {
	if env.StateDir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(env.StateDir, 0o755); err != nil {
		return err
	}
	value := "0\n"
	if on {
		value = "1\n"
	}
	return library.WriteFileAtomic(likedOnlyPath(env), []byte(value), 0o644)
}
