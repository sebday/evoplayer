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
	track := artPathTrack(env, path)
	if nonEmptyFile(track) {
		return track
	}
	if legacy := legacyArtFile(env, path, false); legacy != "" {
		if artLinkFolderAlias(track, legacy) == nil && nonEmptyFile(track) {
			return track
		}
		return legacy
	}
	// A directory cover is only a stand-in when the directory is one album.
	// Empty album tags and dump folders (soundcloud) stay track-only.
	if !folderArtAllowed(env, path) {
		return ""
	}
	folder := artPathFolder(env, path)
	if folder != track && nonEmptyFile(folder) {
		return folder
	}
	if legacy := legacyArtFile(env, path, true); legacy != "" && legacy != folder {
		if artLinkFolderAlias(folder, legacy) == nil && nonEmptyFile(folder) {
			return folder
		}
		return legacy
	}
	return ""
}

// legacyLibraryRel maps a library path from the capitalised folder names back
// to the names the art cache was built with.
func legacyLibraryRel(rel string) string {
	parts := strings.Split(rel, "/")
	changed := false
	for i, part := range parts {
		old, ok := legacyFolderNames[part]
		if !ok {
			continue
		}
		parts[i] = old
		changed = true
	}
	if !changed {
		return ""
	}
	return strings.Join(parts, "/")
}

func legacyArtFile(env Env, path string, folder bool) string {
	key := legacyCacheKey(env.MusicRoot, path, folder)
	if key == "" {
		return ""
	}
	candidate := filepath.Join(env.ArtDir, key+".jpg")
	if nonEmptyFile(candidate) {
		return candidate
	}
	return ""
}

func legacyCacheKey(musicRoot, path string, folder bool) string {
	old := legacyLibraryRel(relUnderRoot(musicRoot, path))
	if old == "" {
		return ""
	}
	if !folder {
		return trackCacheSlug(old)
	}
	if trackInGenreRoot(old) {
		return trackCacheSlug(old)
	}
	dir := filepath.Dir(old)
	if dir == "." || dir == "" {
		dir = old
	}
	return trackCacheSlug(filepath.ToSlash(dir))
}

// legacyFolderNames is the capitalised rename from the previous library layout.
var legacyFolderNames = map[string]string{
	"Drum & Bass": "drum&bass",
	"Dubstep":     "dubstep",
	"Electronic":  "electronic",
	"Garage":      "garage",
	"Grime":       "grime",
	"Hiphop":      "hiphop",
	"House":       "house",
	"Trap":        "trap",
	"Albums":      "albums",
	"Mixes":       "mixes",
	"Soundcloud":  "soundcloud",
	"Vinyl":       "vinyl",
	"Youtube":     "youtube",
	"Prodigy":     "prodigy",
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
	if key := legacyCacheKey(env.MusicRoot, path, false); key != "" {
		candidate = filepath.Join(env.WaveformDir, key+".json")
		if nonEmptyFile(candidate) {
			return candidate
		}
	}
	return ""
}
