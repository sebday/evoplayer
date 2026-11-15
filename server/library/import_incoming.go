package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/syncarchive"
	"github.com/sebday/evoplayer/server/tags"
)

func RunImportCtx(ctx context.Context, env Env, rep jobs.Reporter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep == nil {
		rep = jobs.NopReporter
	}
	incoming := paths.IncomingDir(env.MusicRoot)
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
	removed := 0
	var imported []string
	defer func() { _ = removeIncomingOverlay(env, imported) }()
	known, err := trackBases(db)
	if err != nil {
		return err
	}
	var scArchive *syncarchive.Archive
	if env.StateDir != "" {
		scArchive, err = syncarchive.Load(syncarchive.Path(env.StateDir))
		if err != nil {
			return err
		}
	}
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
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src), "."))
		name := trackFilename(probed.Tag.Artist, probed.Tag.Title, src, ext)
		dest, destErr := incomingDest(env, src, probed)
		candidates := []string{name}
		if destErr == nil {
			candidates = append(candidates, filepath.Base(dest))
		}
		gone, rmErr := removeIncomingDuplicate(src, known, candidates...)
		if rmErr != nil {
			failed++
			rep.Line(jobs.LogFail(base))
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
			continue
		}
		if gone {
			noteSoundcloudID(scArchive, probed.Tag.SoundcloudID)
			removed++
			rep.Line(jobs.LogSkip(base + " (already in library)"))
			rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
			continue
		}
		if destErr != nil {
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
			if err := os.Remove(src); err != nil {
				failed++
				rep.Line(jobs.LogFail(base))
				rep.Progress(jobs.Progress{Phase: base, Done: i + 1, Total: total})
				continue
			}
			noteSoundcloudID(scArchive, probed.Tag.SoundcloudID)
			removed++
			fmt.Fprintf(os.Stderr, "evoplayer: removed existing %s\n", relMusicPath(env.MusicRoot, dest))
			rep.Line(jobs.LogSkip(base + " (already in library)"))
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
		imported = append(imported, base)
		genre := probed.Tag.Genre
		if genre == "" {
			genre = GenreFromPath(env.MusicRoot, dest)
		}
		if err := upsertTrack(tx, env, trackFromProbe(dest, genre, probed), st.ModTime().UnixNano(), st.Size()); err != nil {
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
	if removed > 0 {
		rep.Line(jobs.LogInfof("removed %d already in library", removed))
		fmt.Fprintf(os.Stderr, "evoplayer: removed %d incoming file(s) already in the library\n", removed)
	}
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

func noteSoundcloudID(archive *syncarchive.Archive, id string) {
	if archive == nil {
		return
	}
	n, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil || n == 0 {
		return
	}
	_ = archive.AddSC(n)
}

func trackBases(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT path FROM tracks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	known := map[string]string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		known[filepath.Base(path)] = path
	}
	return known, rows.Err()
}

// removeIncomingDuplicate deletes src when one of bases is already a real library file.
func removeIncomingDuplicate(src string, known map[string]string, bases ...string) (bool, error) {
	for _, base := range bases {
		if base == "" {
			continue
		}
		prev, ok := known[base]
		if !ok {
			continue
		}
		if _, err := os.Stat(prev); err != nil {
			continue
		}
		if err := os.Remove(src); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
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
		genre = MatchLibraryGenre(env, tag.Genre)
	}
	if genre == "" && strings.TrimSpace(tag.Genre) != "" {
		return "", fmt.Errorf("unknown genre for %s: %s", path, tag.Genre)
	}
	if genre == "" {
		return "", fmt.Errorf("unknown genre for %s", path)
	}
	return trackDestForFolder(env, path, genre, probed)
}

func trackDestForFolder(env Env, path, folder string, probed tags.ProbeResult) (string, error) {
	tag := probed.Tag
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		ext = "mp3"
	}
	base := trackFilename(tag.Artist, tag.Title, path, ext)
	if base == "" {
		return "", fmt.Errorf("cannot name %s", path)
	}
	bucket := "soundcloud"
	year := ""
	if IsMix(probed.Duration) {
		bucket = "mixes"
		year = mixYear(tag.Year, path, base)
	} else if isYouTubeSource(path, tag) {
		bucket = "youtube"
		year = mixYear(tag.Year, path, base)
	}
	dir := libraryBucket(env.MusicRoot, folder, bucket)
	if year != "" {
		dir = filepath.Join(dir, year)
	}
	return filepath.Join(dir, base), nil
}

// libraryBucket returns the genre child directory, reusing an existing folder
// whose name matches ignoring case. When none exists, the name has a capital
// first letter (Soundcloud, Mixes, Youtube) so a later create does not split the library.
func libraryBucket(root, genre, name string) string {
	dir := filepath.Join(root, genre)
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.EqualFold(e.Name(), name) {
				return filepath.Join(dir, e.Name())
			}
		}
	}
	return filepath.Join(dir, capitaliseFirst(name))
}

func capitaliseFirst(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// ImportBasename is the filename import uses for an artist and title.
func ImportBasename(artist, title string) string {
	return trackFilename(artist, title, "", "mp3")
}

// FindImported returns the library file for an artist and title, if one is already filed.
func FindImported(db *sql.DB, artist, title string) (string, error) {
	if db == nil {
		return "", nil
	}
	base := ImportBasename(artist, title)
	if base != "" {
		suffix := "/" + base
		var path string
		err := db.QueryRow(`SELECT path FROM tracks WHERE substr(path, -length(?)) = ? LIMIT 1`, suffix, suffix).Scan(&path)
		if err == nil && path != "" {
			return path, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
	}
	artist = strings.TrimSpace(artist)
	title = strings.TrimSpace(title)
	if artist == "" && title == "" {
		return "", nil
	}
	var path string
	err := db.QueryRow(`SELECT path FROM tracks WHERE lower(artist) = lower(?) AND lower(title) = lower(?) LIMIT 1`, artist, title).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return path, err
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
	maxYear := strconv.Itoa(time.Now().Year() + 1)
	for _, path := range paths {
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		for i := 0; i+4 <= len(stem); i++ {
			part := stem[i : i+4]
			if part >= "1985" && part <= maxYear {
				return part
			}
		}
	}
	return "unknown"
}
