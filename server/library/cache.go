package library

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

const cacheWorkers = 8

type CacheResult struct {
	Built   int      `json:"built"`
	Skipped int      `json:"skipped"`
	Total   int      `json:"total"`
	Pruned  int      `json:"pruned,omitempty"`
	Paths   []string `json:"paths,omitempty"`
}

type cacheFile struct {
	path  string
	genre string
	mtime int64
	size  int64
}

func CacheGenre(env Env, genre string, force bool) (CacheResult, error) {
	return CacheGenreCtx(context.Background(), env, genre, force, nil)
}

func CacheGenreCtx(ctx context.Context, env Env, genre string, force bool, onProgress CacheProgressFunc) (CacheResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	dir := filepath.Join(env.MusicRoot, genre)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return CacheResult{}, fmt.Errorf("evoplayer: unknown genre: %s", genre)
	}
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return CacheResult{}, err
	}
	defer db.Close()
	files, err := listAudioFiles(dir, genre)
	if err != nil {
		return CacheResult{}, err
	}
	return cacheFilesIntoCtx(ctx, db, env, files, dir, force, onProgress)
}

type CacheProgress struct {
	Done   int
	Total  int
	Folder string
}

type CacheProgressFunc func(CacheProgress)

func CacheAll(env Env, force bool) (CacheResult, error) {
	return CacheAllCtx(context.Background(), env, force, nil)
}

func CacheAllCtx(ctx context.Context, env Env, force bool, onProgress CacheProgressFunc) (CacheResult, error) {
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return CacheResult{}, err
	}
	defer db.Close()
	return cacheAllIntoCtx(ctx, db, env, force, onProgress)
}

func cacheAllIntoCtx(ctx context.Context, db *sql.DB, env Env, force bool, onProgress CacheProgressFunc) (CacheResult, error) {
	files, err := listAudioFiles(env.MusicRoot, "")
	if err != nil {
		return CacheResult{}, err
	}
	return cacheFilesIntoCtx(ctx, db, env, files, env.MusicRoot, force, onProgress)
}

func cacheFilesIntoCtx(ctx context.Context, db *sql.DB, env Env, files []cacheFile, prefix string, force bool, onProgress CacheProgressFunc) (CacheResult, error) {
	out := CacheResult{Total: len(files)}
	existing, err := loadFileStats(db, prefix)
	if err != nil {
		return out, err
	}
	seen := make(map[string]struct{}, len(files))
	dirty := make([]cacheFile, 0, len(files))
	for _, f := range files {
		seen[f.path] = struct{}{}
		if !force {
			if st, ok := existing[f.path]; ok && st.mtime == f.mtime && st.size == f.size {
				out.Skipped++
				continue
			}
		}
		dirty = append(dirty, f)
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	probed := probeFilesCtx(ctx, env, dirty, onProgress)
	if err := ctx.Err(); err != nil {
		return out, err
	}

	tx, err := db.Begin()
	if err != nil {
		return out, err
	}
	built := 0
	paths := make([]string, 0, len(probed))
	for _, p := range probed {
		if p.track.Path == "" {
			continue
		}
		if err := upsertTrack(tx, env, p.track, p.mtime, p.size); err != nil {
			_ = tx.Rollback()
			return out, err
		}
		built++
		paths = append(paths, p.track.Path)
	}
	out.Built = built
	out.Paths = paths
	pruned, err := pruneMissing(tx, prefix, seen)
	if err != nil {
		_ = tx.Rollback()
		return out, err
	}
	out.Pruned = pruned
	if err := tx.Commit(); err != nil {
		return out, err
	}
	_ = SyncLiked(db, env)
	_, _ = db.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('built_at', datetime('now'))`)
	return out, nil
}

type probedFile struct {
	track Track
	mtime int64
	size  int64
}

func probeFilesCtx(ctx context.Context, env Env, files []cacheFile, onProgress CacheProgressFunc) []probedFile {
	if len(files) == 0 {
		return nil
	}
	workers := cacheWorkers
	if n := runtime.NumCPU(); n > 0 && n < workers {
		workers = n
	}
	if workers > len(files) {
		workers = len(files)
	}
	type job struct {
		idx int
		f   cacheFile
	}
	jobs := make(chan job)
	out := make([]probedFile, len(files))
	var wg sync.WaitGroup
	var doneMu sync.Mutex
	done := 0
	report := func(path string) {
		if onProgress == nil {
			return
		}
		doneMu.Lock()
		done++
		cur := done
		doneMu.Unlock()
		onProgress(CacheProgress{
			Done:   cur,
			Total:  len(files),
			Folder: cacheFolderLabel(env.MusicRoot, path),
		})
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					continue
				}
				out[j.idx] = probeCacheFile(env, j.f)
				report(j.f.path)
			}
		}()
	}
	for i, f := range files {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return out
		default:
		}
		jobs <- job{idx: i, f: f}
	}
	close(jobs)
	wg.Wait()
	return out
}

func cacheFolderLabel(musicRoot, path string) string {
	rel := strings.Trim(filepath.ToSlash(relUnderRoot(musicRoot, path)), "/")
	if rel == "" {
		return ""
	}
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		return rel[:idx]
	}
	return ""
}

// CacheFolderLabel returns the library-relative parent folder for a track path.
func CacheFolderLabel(env Env, path string) string {
	return cacheFolderLabel(env.MusicRoot, path)
}

func probeCacheFile(env Env, f cacheFile) probedFile {
	res, _ := tags.Probe(f.path)
	genre := strings.TrimSpace(res.Tag.Genre)
	if genre == "" {
		genre = f.genre
	}
	if genre == "" {
		genre = genreFromPath(env.MusicRoot, f.path)
	}
	return probedFile{
		track: Track{
			Path:     f.path,
			Genre:    genre,
			Title:    res.Tag.Title,
			Artist:   res.Tag.Artist,
			Album:    res.Tag.Album,
			Year:     res.Tag.Year,
			Label:    res.Tag.Label,
			Duration: res.Duration,
		},
		mtime: f.mtime,
		size:  f.size,
	}
}

// walkRoot follows a symlink at music root so WalkDir can enter ~/Music -> /mnt/...,
// while remapWalkPath keeps playlist/sqlite paths on the original root.
func walkRoot(root string) (walkFrom, displayRoot string) {
	displayRoot = filepath.Clean(root)
	walkFrom = displayRoot
	if resolved, err := filepath.EvalSymlinks(displayRoot); err == nil && resolved != "" {
		walkFrom = resolved
	}
	return walkFrom, displayRoot
}

func remapWalkPath(walkFrom, displayRoot, path string) string {
	if walkFrom == displayRoot {
		return path
	}
	rel, err := filepath.Rel(walkFrom, path)
	if err != nil {
		return path
	}
	if rel == "." {
		return displayRoot
	}
	return filepath.Join(displayRoot, rel)
}

func listAudioFiles(root, folderGenre string) ([]cacheFile, error) {
	out := make([]cacheFile, 0)
	walkFrom, displayRoot := walkRoot(root)
	err := filepath.WalkDir(walkFrom, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		displayPath := remapWalkPath(walkFrom, displayRoot, path)
		name := d.Name()
		if d.IsDir() {
			if path != walkFrom && (strings.HasPrefix(name, ".") || name == ".incoming") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if !playback.IsSupportedPath(displayPath) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		genre := folderGenre
		if genre == "" {
			genre = genreFromPath(displayRoot, displayPath)
		}
		out = append(out, cacheFile{
			path:  displayPath,
			genre: genre,
			mtime: info.ModTime().UnixNano(),
			size:  info.Size(),
		})
		return nil
	})
	return out, err
}

func listGenreNames(env Env) ([]string, error) {
	entries, err := os.ReadDir(env.MusicRoot)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// NeedsBootstrap is true when the music root has audio files and the sqlite index is missing or empty.
func NeedsBootstrap(env Env) (bool, error) {
	if env.MusicRoot == "" {
		return false, nil
	}
	if st, err := os.Stat(env.MusicRoot); err != nil || !st.IsDir() {
		return false, nil
	}
	if env.LibraryDB != "" {
		if st, err := os.Stat(env.LibraryDB); err == nil && !st.IsDir() {
			db, err := OpenDB(env.LibraryDB)
			if err == nil {
				ready := Ready(db)
				_ = db.Close()
				if ready {
					return false, nil
				}
			}
		}
	}
	files, err := listAudioFiles(env.MusicRoot, "")
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// NeedsScan is true when the sqlite index is stale versus on-disk files (add/remove/mtime/size).
func NeedsScan(env Env) (bool, error) {
	if env.MusicRoot == "" {
		return false, nil
	}
	if st, err := os.Stat(env.MusicRoot); err != nil || !st.IsDir() {
		return false, nil
	}
	files, err := listAudioFiles(env.MusicRoot, "")
	if err != nil {
		return false, err
	}
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return len(files) > 0, nil
	}
	defer db.Close()
	existing, err := loadFileStats(db, env.MusicRoot)
	if err != nil {
		return true, err
	}
	if len(files) == 0 {
		return len(existing) > 0, nil
	}
	if len(existing) == 0 {
		return true, nil
	}
	seen := make(map[string]struct{}, len(files))
	for _, f := range files {
		seen[f.path] = struct{}{}
		st, ok := existing[f.path]
		if !ok || st.mtime != f.mtime || st.size != f.size {
			return true, nil
		}
	}
	for path := range existing {
		if _, ok := seen[path]; !ok {
			return true, nil
		}
	}
	return false, nil
}
