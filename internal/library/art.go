package library

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func resolveArt(env Env, path, existing string) string {
	if existing != "" && isDecodableArtFile(existing) {
		return existing
	}
	return artCacheFind(env, path)
}

func artCacheFind(env Env, path string) string {
	if path == "" {
		return ""
	}
	folder := filepath.Join(env.ArtDir, artFolderKey(env.MusicRoot, path)+".jpg")
	if isDecodableArtFile(folder) {
		return folder
	}
	legacy := filepath.Join(env.ArtDir, cacheKey(env.MusicRoot, path)+".jpg")
	if isDecodableArtFile(legacy) {
		return legacy
	}
	return ""
}

func artFolderKey(musicRoot, path string) string {
	rel := relUnderRoot(musicRoot, path)
	if trackInGenreRoot(rel) {
		return trackCacheSlug(rel)
	}
	dir := filepath.Dir(rel)
	if dir == "." || dir == "" {
		dir = rel
	}
	return trackCacheSlug(filepath.ToSlash(dir))
}

func CacheKey(musicRoot, path string) string {
	return trackCacheSlug(relUnderRoot(musicRoot, path))
}

func cacheKey(musicRoot, path string) string {
	return CacheKey(musicRoot, path)
}

func relUnderRoot(musicRoot, path string) string {
	root := filepath.Clean(musicRoot)
	p := filepath.Clean(path)
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.Base(p)
	}
	return filepath.ToSlash(rel)
}

func trackInGenreRoot(rel string) bool {
	rel = strings.Trim(rel, "/")
	if rel == "" {
		return false
	}
	return strings.Count(rel, "/") == 1
}

func trackCacheSlug(s string) string {
	s = filepath.ToSlash(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '&' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func isDecodableArtFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

func resolveWaveform(env Env, path, existing string) string {
	if existing != "" && isWaveformFile(existing) {
		return existing
	}
	return waveformCacheFind(env, path)
}

func waveformCacheFind(env Env, path string) string {
	if path == "" {
		return ""
	}
	candidate := filepath.Join(env.WaveformDir, cacheKey(env.MusicRoot, path)+".json")
	if isWaveformFile(candidate) {
		return candidate
	}
	return ""
}

func isWaveformFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}
