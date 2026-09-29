package playlist

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/library"
)

func readM3UPaths(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var paths []string
	dir := filepath.Dir(path)
	sc := bufio.NewScanner(f)
	for first := true; sc.Scan(); first = false {
		line := library.M3UEntry(sc.Text(), first)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(dir, line)
		}
		if st, err := os.Stat(line); err != nil || st.IsDir() {
			continue
		}
		paths = append(paths, line)
	}
	return paths, sc.Err()
}

func writeLikesM3U(env Env, outPath string) error {
	raw, err := os.ReadFile(env.LikesFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var likes map[string]json.RawMessage
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &likes)
	}
	paths := make([]string, 0, len(likes))
	for p := range likes {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			continue
		}
		if !audio.IsAudio(p) {
			continue
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return writeM3U(outPath, paths)
}

func writeMixesM3U(env Env) error {
	db, dbErr := library.EnsureDB(env.Env)
	if dbErr != nil {
		db = nil
	}
	paths, err := library.LikedMixPaths(db, env.Env)
	if err != nil {
		return err
	}
	return writeM3U(filepath.Join(env.PlaylistDir, "mixes.m3u"), paths)
}

func writeGenreM3U(env Env, genre string) error {
	paths, err := likedPathsForGenre(env, genre)
	if err != nil {
		return err
	}
	out := filepath.Join(env.PlaylistDir, genre+".m3u")
	if len(paths) == 0 {
		_ = os.Remove(out)
		_ = os.Remove(filepath.Join(env.PlaylistDir, genre+"-fav.m3u"))
		return nil
	}
	return writeM3U(out, paths)
}

func writeM3U(path string, paths []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, p := range paths {
		b.WriteString(p + "\n")
	}
	return library.WriteFileAtomic(path, []byte(b.String()), 0o644)
}

func likedPathsForGenre(env Env, genre string) ([]string, error) {
	raw, err := os.ReadFile(env.LikesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var likes map[string]json.RawMessage
	if json.Unmarshal(raw, &likes) != nil {
		return nil, nil
	}
	paths := make([]string, 0)
	for p := range likes {
		if library.GenreFromPath(env.MusicRoot, p) != genre {
			continue
		}
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			continue
		}
		if !audio.IsAudio(p) {
			continue
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}
