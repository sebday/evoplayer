package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/internal/audio"
	"github.com/sebday/evoplayer/internal/tags"
)

func RunImport(env Env) error {
	incoming := filepath.Join(env.MusicRoot, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(incoming)
	if err != nil {
		return err
	}
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return err
	}
	defer db.Close()

	moved := 0
	failed := 0
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(incoming, e.Name())
		if incomingSkipFile(src) {
			continue
		}
		if !audio.IsAudio(src) {
			continue
		}
		if _, _, err := tags.StandardizePath(env.MusicRoot, src); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: tag standardize failed: %s\n", src)
		}
		probed, _ := tags.Probe(src)
		dest, err := incomingDest(env, src, probed)
		if err != nil {
			failed++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			failed++
			continue
		}
		if _, err := os.Stat(dest); err == nil {
			fmt.Fprintf(os.Stderr, "evoplayer: skip existing dest: %s\n", dest)
			failed++
			continue
		}
		if err := os.Rename(src, dest); err != nil {
			failed++
			continue
		}
		st, statErr := os.Stat(dest)
		if statErr != nil {
			failed++
			continue
		}
		genre := probed.Tag.Genre
		if genre == "" {
			genre = genreFromPath(env.MusicRoot, dest)
		}
		item := Track{
			Path:     dest,
			Genre:    genre,
			Title:    probed.Tag.Title,
			Artist:   probed.Tag.Artist,
			Album:    probed.Tag.Album,
			Year:     probed.Tag.Year,
			Label:    probed.Tag.Label,
			Duration: probed.Duration,
		}
		if err := upsertTrack(tx, env, item, st.ModTime().UnixNano(), st.Size()); err != nil {
			_ = tx.Rollback()
			return err
		}
		moved++
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_ = SyncLiked(db, env)
	if moved == 0 && failed == 0 {
		fmt.Fprintln(os.Stderr, "evoplayer: nothing to import in .incoming/")
		return nil
	}
	fmt.Fprintf(os.Stderr, "evoplayer: imported %d file(s) from .incoming/\n", moved)
	if failed > 0 {
		return fmt.Errorf("evoplayer: import failed for %d file(s)", failed)
	}
	return nil
}

func incomingSkipFile(path string) bool {
	base := filepath.Base(path)
	if strings.Contains(base, ".part") || strings.Contains(base, ".ytdl") || strings.Contains(base, ".temp") {
		return true
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	switch ext {
	case "part", "ytdl", "temp", "jpg", "jpeg", "png", "webp", "gif":
		return true
	}
	return false
}

func incomingDest(env Env, path string, probed tags.ProbeResult) (string, error) {
	tag := probed.Tag
	genre := genreFolderFromTags(path, tag)
	if genre == "" {
		return "", fmt.Errorf("unknown genre for %s", path)
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		ext = "mp3"
	}
	base := trackFilename(tag.Artist, tag.Title, path, ext)
	if base == "" {
		return "", fmt.Errorf("cannot name %s", path)
	}
	dir := filepath.Join(env.MusicRoot, genre, "soundcloud")
	if isYouTubeSource(path, tag) {
		year := mixYear(tag.Year, path)
		dir = filepath.Join(env.MusicRoot, genre, "youtube", year)
	} else if isMix(path, probed.Duration) {
		year := mixYear(tag.Year, path)
		dir = filepath.Join(env.MusicRoot, genre, "mixes", year)
	}
	return filepath.Join(dir, base), nil
}

func genreFolderFromTags(path string, tag tags.TagInfo) string {
	stem := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	stems := []struct {
		pattern string
		folder  string
	}{
		{"excision-darkside_dubstep", "dubstep"},
		{"future_beats_radio_show", "drum&bass"},
		{"nas", "hiphop"},
		{"kendrick", "hiphop"},
	}
	for _, item := range stems {
		if strings.Contains(stem, item.pattern) {
			return item.folder
		}
	}
	genreMap := map[string]string{
		"drum & bass": "drum&bass", "drum and bass": "drum&bass", "dnb": "drum&bass",
		"dubstep": "dubstep", "grime": "grime", "hip-hop": "hiphop", "hip hop": "hiphop",
		"house": "house", "misc": "misc",
	}
	for _, field := range []string{tag.Genre, tag.Artist, tag.Title} {
		lower := strings.ToLower(strings.TrimSpace(field))
		if folder, ok := genreMap[lower]; ok {
			return folder
		}
	}
	return ""
}

func trackFilename(artist, title, path, ext string) string {
	artistSlug := tags.Slugify(artist)
	titleSlug := tags.Slugify(title)
	if artistSlug != "" && titleSlug != "" {
		if titleSlug == artistSlug {
			titleSlug = ""
		} else if strings.HasPrefix(titleSlug, artistSlug+"_") {
			titleSlug = strings.TrimPrefix(titleSlug, artistSlug+"_")
		}
	}
	var base string
	switch {
	case artistSlug != "" && titleSlug != "":
		base = artistSlug + "-" + titleSlug
	case titleSlug != "":
		base = titleSlug
	case artistSlug != "":
		base = artistSlug
	default:
		base = tags.Slugify(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	}
	if base == "" {
		return ""
	}
	return base + "." + ext
}

func isYouTubeSource(path string, tag tags.TagInfo) bool {
	if strings.Contains(strings.ToLower(tag.Comment), "source:youtube") {
		return true
	}
	stem := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	return strings.Contains(stem, "youtube")
}

func isMix(path string, dur float64) bool {
	if dur > 45*60 {
		return true
	}
	stem := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	markers := []string{"mixed", "essential", "euphoria", "session", "boiler", "radio_show", "radio-show", "-mix", "_mix", "live_mixed"}
	for _, m := range markers {
		if strings.Contains(stem, m) {
			return true
		}
	}
	return false
}

func mixYear(yearTag, path string) string {
	year := strings.TrimSpace(yearTag)
	if len(year) >= 4 {
		return year[:4]
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for i := 0; i+4 <= len(stem); i++ {
		part := stem[i : i+4]
		if part >= "1985" && part <= "2026" {
			return part
		}
	}
	return "unknown"
}
