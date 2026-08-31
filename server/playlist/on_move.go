package playlist

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/library"
)

func OnTrackMoved(env Env, from, to string) error {
	if from == "" || to == "" || from == to {
		return nil
	}
	likes, err := readLikes(env.LikesFile)
	if err != nil {
		return err
	}
	if entry, ok := likes[from]; ok {
		delete(likes, from)
		likes[to] = entry
		if err := writeLikes(env.LikesFile, likes); err != nil {
			return err
		}
		library.InvalidateLikesCache(env.LikesFile)
	}
	if paths, err := ReadCurrentPaths(env); err == nil {
		changed := false
		for i, p := range paths {
			if p == from {
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
				_ = replaceM3UPath(filepath.Join(env.PlaylistDir, e.Name()), from, to)
			}
		}
	}
	oldGenre := genreFromPath(env.MusicRoot, from)
	newGenre := genreFromPath(env.MusicRoot, to)
	_ = writeLikesM3U(env, filepath.Join(env.PlaylistDir, "all.m3u"))
	_ = writeMixesM3U(env)
	if oldGenre != "" {
		_ = writeGenreM3U(env, oldGenre)
	}
	if newGenre != "" && newGenre != oldGenre {
		_ = writeGenreM3U(env, newGenre)
	}
	return nil
}

func replaceM3UPath(path, from, to string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	text := string(raw)
	if !strings.Contains(text, from) {
		return nil
	}
	lines := strings.Split(text, "\n")
	changed := false
	for i, line := range lines {
		trim := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if trim == from {
			lines[i] = to
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}
