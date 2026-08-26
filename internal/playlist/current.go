package playlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sebday/evoplayer/internal/audio"
	"github.com/sebday/evoplayer/internal/config"
	"github.com/sebday/evoplayer/internal/library"
)

type ExtendResult struct {
	Folder string `json:"folder"`
	Added  int    `json:"added"`
}

func SaveCurrent(env Env, paths []string) error {
	filtered := filterAudioPaths(paths)
	if err := saveCurrentM3U(env, filtered); err != nil {
		return err
	}
	return writeCurrentTracksJSON(env, filtered)
}

// SaveCurrentFast writes the M3U when paths changed. Returns whether the M3U changed.
func SaveCurrentFast(env Env, paths []string) (bool, error) {
	filtered := filterAudioPaths(paths)
	existing, err := readM3UPaths(env.currentM3U())
	if err == nil && pathsEqual(existing, filtered) {
		return false, nil
	}
	return true, saveCurrentM3U(env, filtered)
}

// EnrichCurrentTracksJSON rebuilds current.tracks.json from paths.
func EnrichCurrentTracksJSON(env Env, paths []string) error {
	return writeCurrentTracksJSON(env, filterAudioPaths(paths))
}

func saveCurrentM3U(env Env, paths []string) error {
	if err := os.MkdirAll(env.PlaylistDir, 0o755); err != nil {
		return err
	}
	return writeM3U(env.currentM3U(), paths)
}

func writeCurrentTracksJSON(env Env, paths []string) error {
	items := library.TracksForPaths(env.Env, paths, "")
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	tmp := env.currentTracksJSON() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, env.currentTracksJSON())
}

func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ClearCurrent(env Env) error {
	_ = os.Remove(env.currentM3U())
	_ = os.Remove(env.currentTracksJSON())
	return nil
}

func ReadCurrentPaths(env Env) ([]string, error) {
	paths, err := readM3UPaths(env.currentM3U())
	if err != nil {
		return nil, err
	}
	if len(paths) > 0 {
		return paths, nil
	}
	legacy := filepath.Join(env.StateDir, "current.m3u")
	legacyPaths, err := readM3UPaths(legacy)
	if err != nil || len(legacyPaths) == 0 {
		return paths, nil
	}
	_ = SaveCurrent(env, legacyPaths)
	_ = os.Remove(legacy)
	return legacyPaths, nil
}

func UpNextTracks(env Env, currentPath string, limit int) ([]library.Track, error) {
	if limit <= 0 {
		limit = 5
	}
	cachePath := env.currentTracksJSON()
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, nil
	}
	var items []library.Track
	if json.Unmarshal(raw, &items) != nil {
		return nil, nil
	}
	idx := -1
	for i, item := range items {
		if item.Path == currentPath {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, nil
	}
	end := idx + 1 + limit
	if end > len(items) {
		end = len(items)
	}
	if idx+1 >= end {
		return nil, nil
	}
	return items[idx+1:end], nil
}

func ExtendCurrent(env Env, currentPath string) (ExtendResult, []string, []string, bool, error) {
	paths, err := ReadCurrentPaths(env)
	if err != nil {
		return ExtendResult{}, nil, nil, false, err
	}
	if len(paths) == 0 {
		return ExtendResult{}, nil, nil, false, fmt.Errorf("current queue empty")
	}
	lastDir := filepath.Dir(paths[len(paths)-1])
	nextDir, err := nextSiblingFolder(env, lastDir)
	if err != nil {
		return ExtendResult{}, nil, nil, false, err
	}
	newPaths, err := audioFilesInDir(nextDir)
	if err != nil {
		return ExtendResult{}, nil, nil, false, err
	}
	if len(newPaths) == 0 {
		return ExtendResult{}, nil, nil, false, fmt.Errorf("no tracks in %s", nextDir)
	}
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		seen[p] = struct{}{}
	}
	var added []string
	for _, p := range newPaths {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		added = append(added, p)
	}
	if len(added) == 0 {
		return ExtendResult{}, nil, nil, false, fmt.Errorf("no new tracks to append")
	}
	merged := append(paths, added...)
	atEnd := false
	if currentPath != "" {
		for i, p := range paths {
			if p == currentPath && i == len(paths)-1 {
				atEnd = true
				break
			}
		}
	}
	return ExtendResult{Folder: nextDir, Added: len(added)}, added, merged, atEnd, nil
}

func filterAudioPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		st, err := os.Stat(p)
		if err != nil || st.IsDir() || !audio.IsAudio(p) {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func audioFilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if audio.IsAudio(p) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func nextSiblingFolder(env Env, curDir string) (string, error) {
	parent := filepath.Dir(curDir)
	base := filepath.Base(curDir)
	if parent == curDir {
		return "", fmt.Errorf("no sibling folder for %s", curDir)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", err
	}
	skip, _ := config.SkipDirs(env.MusicConfig)
	skipSet := map[string]struct{}{
		".incoming": {},
		"incoming":  {},
	}
	for _, name := range skip {
		skipSet[name] = struct{}{}
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, ok := skipSet[name]; ok {
			continue
		}
		dirs = append(dirs, filepath.Join(parent, name))
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no sibling folders under %s", parent)
	}
	sort.Strings(dirs)
	idx := -1
	for i, d := range dirs {
		if d == curDir {
			idx = i
			break
		}
	}
	if idx < 0 {
		for _, d := range dirs {
			if filepath.Base(d) > base {
				return d, nil
			}
		}
		return dirs[0], nil
	}
	next := idx + 1
	if next >= len(dirs) {
		next = 0
	}
	return dirs[next], nil
}
