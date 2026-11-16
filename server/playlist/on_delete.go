package playlist

import (
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/server/library"
)

// OnTrackDeleted drops a removed file from likes, playlists, and saved track order.
func OnTrackDeleted(env Env, path string) error {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil
	}
	likes, err := readLikes(env.LikesFile)
	if err != nil {
		return err
	}
	if _, ok := likes[path]; ok {
		delete(likes, path)
		if err := writeLikes(env.LikesFile, likes); err != nil {
			return err
		}
		library.InvalidateLikesCache(env.LikesFile)
	}
	if env.PlaylistDir != "" {
		entries, err := os.ReadDir(env.PlaylistDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || filepath.Ext(e.Name()) != ".m3u" {
					continue
				}
				_ = library.RewriteM3U(filepath.Join(env.PlaylistDir, e.Name()), func(p string) string {
					if filepath.Clean(p) == path {
						return ""
					}
					return p
				})
			}
		}
	}
	_ = writeLikesM3U(env, filepath.Join(env.PlaylistDir, "all.m3u"))
	_ = writeMixesM3U(env)
	if genre := library.GenreFromPath(env.MusicRoot, path); genre != "" {
		_ = writeGenreM3U(env, genre)
	}
	return forgetTrackPath(env, path)
}
