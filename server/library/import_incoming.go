package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/tags"
)

func RunImportCtx(ctx context.Context, env Env, rep jobs.Reporter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep == nil {
		rep = jobs.NopReporter
	}
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

	pending := incomingAudioFiles(entries, incoming)
	rep.Line(jobs.LogInfo("importing .incoming"))
	rep.Line(jobs.LogInfof("%d files", len(pending)))
	if len(pending) == 0 {
		rep.Line(jobs.LogInfo("nothing to import"))
		fmt.Fprintln(os.Stderr, "evoplayer: nothing to import in .incoming/")
		return nil
	}

	moved := 0
	failed := 0
	skipped := 0
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	total := len(pending)
	for i, src := range pending {
		if err := ctx.Err(); err != nil {
			_ = tx.Rollback()
			return err
		}
		base := filepath.Base(src)
		rep.Progress(jobs.Progress{Phase: base, Done: i, Total: total})
		info, err := os.Stat(src)
		if err != nil || info.Size() == 0 {
			skipped++
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
			continue
		}
		probed, _ := tags.ProbeImport(src)
		dest, err := incomingDest(env, src, probed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: skip import (no dest): %s\n", src)
			rep.Line(jobs.LogSkip(base + " (no genre)"))
			skipped++
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			failed++
			rep.Line(jobs.LogFail(base))
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
			continue
		}
		if _, err := os.Stat(dest); err == nil {
			fmt.Fprintf(os.Stderr, "evoplayer: skip existing dest: %s\n", dest)
			rep.Line(jobs.LogSkip(relMusicPath(env.MusicRoot, dest) + " (exists)"))
			skipped++
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
			continue
		}
		if err := os.Rename(src, dest); err != nil {
			failed++
			rep.Line(jobs.LogFail(base))
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
			continue
		}
		st, statErr := os.Stat(dest)
		if statErr != nil {
			failed++
			rep.Line(jobs.LogFail(base))
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
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
		rep.Line(jobs.LogOK(relMusicPath(env.MusicRoot, dest)))
		rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_ = SyncLiked(db, env)
	rep.Line(jobs.LogInfof("imported %d", moved))
	fmt.Fprintf(os.Stderr, "evoplayer: imported %d file(s) from .incoming/\n", moved)
	if skipped > 0 {
		rep.Line(jobs.LogInfof("skipped %d", skipped))
	}
	if failed > 0 {
		rep.Line(jobs.LogFail(fmt.Sprintf("%d failed", failed)))
		return fmt.Errorf("evoplayer: import failed for %d file(s)", failed)
	}
	return nil
}

func incomingAudioFiles(entries []os.DirEntry, incoming string) []string {
	out := make([]string, 0, len(entries))
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
		info, err := os.Stat(src)
		if err != nil || info.Size() == 0 {
			continue
		}
		out = append(out, src)
	}
	return out
}

func relMusicPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, "..") {
		return filepath.Base(path)
	}
	return rel
}

func incomingSkipFile(path string) bool {
	base := filepath.Base(path)
	if base == incomingTagsFile {
		return true
	}
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
	genre := overlayGenre(env, path)
	if genre == "" {
		genre = genreFolderFromTags(env, tag)
	}
	if genre == "" && strings.TrimSpace(tag.Genre) != "" {
		return "", fmt.Errorf("unknown genre for %s: %s", path, tag.Genre)
	}
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
	if incomingIsMix(path, base, probed.Duration) {
		year := mixYear(tag.Year, path, base)
		dir = filepath.Join(env.MusicRoot, genre, "mixes", year)
	} else if isYouTubeSource(path, tag) {
		year := mixYear(tag.Year, path, base)
		dir = filepath.Join(env.MusicRoot, genre, "youtube", year)
	}
	return filepath.Join(dir, base), nil
}

func incomingIsMix(incomingPath, destBase string, dur float64) bool {
	return IsMix(incomingPath, dur) || IsMix(destBase, dur)
}

func genreFolderFromTags(env Env, tag tags.TagInfo) string {
	return MatchLibraryGenre(env, tag.Genre)
}

func matchLibraryFolder(env Env, name string) string {
	return MatchLibraryGenre(env, name)
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

func mixYear(yearTag string, paths ...string) string {
	year := strings.TrimSpace(yearTag)
	if len(year) >= 4 {
		return year[:4]
	}
	for _, path := range paths {
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		for i := 0; i+4 <= len(stem); i++ {
			part := stem[i : i+4]
			if part >= "1985" && part <= "2026" {
				return part
			}
		}
	}
	return "unknown"
}
