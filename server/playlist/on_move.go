package playlist

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/library"
)

func OnTrackMoved(env Env, from, to string) error {
	return OnTracksMoved(env, map[string]string{from: to})
}

// OnTracksMoved rewrites likes, the current queue and playlists for moved
// tracks (old path -> new path) and regenerates the affected auto lists.
func OnTracksMoved(env Env, moved map[string]string) error {
	moves := make(map[string]string, len(moved))
	for from, to := range moved {
		if from != "" && to != "" && from != to {
			moves[from] = to
		}
	}
	if len(moves) == 0 {
		return nil
	}
	likes, err := readLikes(env.LikesFile)
	if err != nil {
		return err
	}
	likesChanged := false
	for from, to := range moves {
		if entry, ok := likes[from]; ok {
			delete(likes, from)
			likes[to] = entry
			likesChanged = true
		}
	}
	if likesChanged {
		if err := writeLikes(env.LikesFile, likes); err != nil {
			return err
		}
		library.InvalidateLikesCache(env.LikesFile)
	}
	if paths, err := ReadCurrentPaths(env); err == nil {
		changed := false
		for i, p := range paths {
			if to, ok := moves[p]; ok {
				paths[i] = to
				changed = true
			}
		}
		if changed {
			_ = SaveCurrent(env, paths)
		}
	}
	if env.PlaylistDir != "" {
		entries, err := os.ReadDir(env.PlaylistDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".m3u") {
					continue
				}
				_ = library.RewriteM3U(filepath.Join(env.PlaylistDir, e.Name()), func(p string) string {
					if to, ok := moves[p]; ok {
						return to
					}
					return p
				})
			}
		}
	}
	_ = writeLikesM3U(env, filepath.Join(env.PlaylistDir, "all.m3u"))
	_ = writeMixesM3U(env)
	genres := map[string]struct{}{}
	for from, to := range moves {
		genres[library.GenreFromPath(env.MusicRoot, from)] = struct{}{}
		genres[library.GenreFromPath(env.MusicRoot, to)] = struct{}{}
	}
	for genre := range genres {
		if genre != "" {
			_ = writeGenreM3U(env, genre)
		}
	}
	return nil
}
