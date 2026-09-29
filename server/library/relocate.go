package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/config"
)

func topFolderMap(root string) map[string]string {
	names, err := listGenreNames(root)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, name := range names {
		key := config.NormalizeGenreKey(name)
		if key == "" {
			continue
		}
		if _, ok := out[key]; !ok {
			out[key] = name
		}
	}
	return out
}

// FoldedGenreFolder returns the top-level library folder whose normalized name matches name.
func FoldedGenreFolder(root, name string) string {
	key := config.NormalizeGenreKey(name)
	if key == "" {
		return ""
	}
	return topFolderMap(root)[key]
}

func relUnderRootFold(root, path string) string {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	sep := string(os.PathSeparator)
	prefix := strings.TrimRight(root, sep) + sep
	if len(path) <= len(prefix) {
		return ""
	}
	if !strings.EqualFold(path[:len(prefix)], prefix) {
		return ""
	}
	return path[len(prefix):]
}

func resolveCaseInsensitive(root, rel string) string {
	cur := filepath.Clean(root)
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		entries, err := os.ReadDir(cur)
		if err != nil {
			return ""
		}
		match := ""
		for _, e := range entries {
			if strings.EqualFold(e.Name(), part) {
				match = e.Name()
				break
			}
		}
		if match == "" {
			return ""
		}
		cur = filepath.Join(cur, match)
	}
	if fileExists(cur) {
		return cur
	}
	return ""
}

type fileIndex struct {
	ci   map[string]string
	base map[string][]string
}

func indexLibraryFiles(root string) *fileIndex {
	idx := &fileIndex{ci: map[string]string{}, base: map[string][]string{}}
	_ = walkLibrary(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		idx.ci[strings.ToLower(path)] = path
		b := strings.ToLower(d.Name())
		idx.base[b] = append(idx.base[b], path)
		return nil
	})
	return idx
}

func (idx *fileIndex) relocate(root, path string) string {
	if next := RelocatePath(root, path); next != path && fileExists(next) {
		return next
	}
	if idx == nil {
		return RelocatePath(root, path)
	}
	if actual, ok := idx.ci[strings.ToLower(path)]; ok {
		return actual
	}
	if rel := relUnderRootFold(root, path); rel != "" {
		guess := filepath.Join(root, rel)
		if actual, ok := idx.ci[strings.ToLower(guess)]; ok {
			return actual
		}
	}
	parent := filepath.Base(filepath.Dir(path))
	match := ""
	for _, hit := range idx.base[strings.ToLower(filepath.Base(path))] {
		if !strings.EqualFold(filepath.Base(filepath.Dir(hit)), parent) {
			continue
		}
		if match != "" {
			return RelocatePath(root, path)
		}
		match = hit
	}
	if match != "" {
		return match
	}
	return RelocatePath(root, path)
}

// RelocatePath rewrites a library path when the music root or a folder was renamed.
func RelocatePath(root, path string) string {
	if root == "" || path == "" {
		return path
	}
	if fileExists(path) {
		return path
	}
	rel := relUnderRootFold(root, path)
	if rel == "" {
		return path
	}
	if found := resolveCaseInsensitive(root, rel); found != "" {
		return found
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	folders := topFolderMap(root)
	if folders == nil {
		return path
	}
	actual, ok := folders[config.NormalizeGenreKey(parts[0])]
	if !ok || actual == parts[0] {
		return path
	}
	parts[0] = actual
	if found := resolveCaseInsensitive(root, filepath.Join(parts...)); found != "" {
		return found
	}
	return filepath.Join(root, filepath.Join(parts...))
}

// RelocateLibraryPaths rewrites likes, playlists, and player state onto current folder names.
func RelocateLibraryPaths(env Env) error {
	if env.MusicRoot == "" {
		return nil
	}
	idx := indexLibraryFiles(env.MusicRoot)
	changed := false
	if n, err := relocateLikesFile(env, idx); err != nil {
		return err
	} else if n > 0 {
		changed = true
	}
	if env.StateDir != "" {
		if err := relocateM3UDir(env.MusicRoot, filepath.Join(env.StateDir, "playlists"), idx); err != nil {
			return err
		}
		_ = relocatePlayerState(env.MusicRoot, filepath.Join(env.StateDir, "player.json"), idx)
	}
	if changed {
		InvalidateLikesCache(env.LikesFile)
	}
	return nil
}

func relocateLikesFile(env Env, idx *fileIndex) (int, error) {
	raw, err := os.ReadFile(env.LikesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var likes map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &likes) != nil {
		return 0, nil
	}
	out := make(map[string]json.RawMessage, len(likes))
	moved := 0
	for path, meta := range likes {
		next := idx.relocate(env.MusicRoot, path)
		if next != path {
			moved++
		}
		if _, ok := out[next]; !ok {
			out[next] = meta
		}
	}
	if moved == 0 {
		return 0, nil
	}
	return moved, writeJSONAtomic(env.LikesFile, out)
}

func relocateM3UDir(root, dir string, idx *fileIndex) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(dir, name)
		switch {
		case strings.HasSuffix(name, ".m3u"):
			if err := RewriteM3U(path, func(p string) string { return idx.relocate(root, p) }); err != nil {
				return err
			}
		case name == "current.tracks.json":
			_ = relocateTracksJSON(root, path, idx)
		}
	}
	return nil
}

// RewriteM3U maps every entry line of an m3u through fn and saves the file when any entry changed.
func RewriteM3U(path string, fn func(string) string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(raw), "\n")
	changed := false
	for i, line := range lines {
		trim := M3UEntry(line, i == 0)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if next := fn(trim); next != trim {
			if strings.HasSuffix(line, "\r") {
				next += "\r"
			}
			lines[i] = next
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return WriteFileAtomic(path, []byte(strings.Join(lines, "\n")), 0o644)
}

// M3UEntry trims a raw m3u line, dropping CRLF endings and a UTF-8 BOM on the first line.
func M3UEntry(line string, first bool) string {
	if first {
		line = strings.TrimPrefix(line, "\ufeff")
	}
	return strings.TrimSpace(strings.TrimSuffix(line, "\r"))
}

func relocateTracksJSON(root, path string, idx *fileIndex) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var items []map[string]any
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	changed := false
	for _, item := range items {
		p, _ := item["path"].(string)
		next := idx.relocate(root, p)
		if next != p {
			item["path"] = next
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return writeJSONAtomic(path, items)
}

func relocatePlayerState(root, path string, idx *fileIndex) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var st map[string]any
	if json.Unmarshal(raw, &st) != nil {
		return nil
	}
	p, _ := st["path"].(string)
	next := idx.relocate(root, p)
	if next == p {
		return nil
	}
	st["path"] = next
	return writeJSONAtomic(path, st)
}

func writeJSONAtomic(path string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, raw, 0o644)
}
