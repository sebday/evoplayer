package library

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/server/playback"
)

// PruneArt removes duplicate legacy per-track art files when a folder alias exists.
func PruneArt(env Env) (int, error) {
	pruned := 0
	err := filepath.WalkDir(env.MusicRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !playback.IsSupportedPath(path) {
			return nil
		}
		legacy := artPathLegacy(env, path)
		folder := artPathFolder(env, path)
		if legacy == folder {
			return nil
		}
		if !isDecodableArtFile(legacy) || !isDecodableArtFile(folder) {
			return nil
		}
		if sameFile(legacy, folder) {
			if err := os.Remove(legacy); err == nil {
				pruned++
			}
			return nil
		}
		lb, err := os.ReadFile(legacy)
		if err != nil {
			return nil
		}
		fb, err := os.ReadFile(folder)
		if err != nil {
			return nil
		}
		if bytes.Equal(lb, fb) {
			if err := os.Remove(legacy); err == nil {
				pruned++
			}
		}
		return nil
	})
	return pruned, err
}
