package library

import (
	"path/filepath"
	"strings"
	"unicode"
)

func resolveArt(env Env, path, existing string) string {
	if existing != "" && nonEmptyFile(existing) {
		return existing
	}
	return artCacheFind(env, path)
}

func artCacheFind(env Env, path string) string {
	if path == "" {
		return ""
	}
	track := filepath.Join(env.ArtDir, CacheKey(env.MusicRoot, path)+".jpg")
	if nonEmptyFile(track) {
		return track
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

func resolveWaveform(env Env, path, existing string) string {
	if existing != "" && nonEmptyFile(existing) {
		return existing
	}
	return waveformCacheFind(env, path)
}

func waveformCacheFind(env Env, path string) string {
	if path == "" {
		return ""
	}
	candidate := filepath.Join(env.WaveformDir, CacheKey(env.MusicRoot, path)+".json")
	if nonEmptyFile(candidate) {
		return candidate
	}
	return ""
}
