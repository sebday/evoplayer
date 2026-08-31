package library

import (
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/server/playback"
)

// PruneArt removes duplicate folder-alias art files when per-track art exists.
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
		if err := os.Remove(folder); err == nil {
			pruned++
		}
		return nil
	})
	return pruned, err
}
