package library

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sebday/evoplayer/server/playback"
)

type BrowseEntry struct {
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
	Count int    `json:"count,omitempty"`
	Track
}

type BrowseResult struct {
	Path        string        `json:"path"`
	Parent      *string       `json:"parent"`
	Entries     []BrowseEntry `json:"entries,omitempty"`
	Tracks      []Track       `json:"tracks,omitempty"`
	Paths       []string      `json:"paths,omitempty"`
	TrackTotal  int           `json:"trackTotal,omitempty"`
	TrackOffset int           `json:"trackOffset,omitempty"`
	TrackLimit  int           `json:"trackLimit,omitempty"`
}

type BrowseOptions struct {
	Rel            string
	Queue          bool
	QueuePathsOnly bool
	Offset         int
	Limit          int
}

func Browse(env Env, opt BrowseOptions) (BrowseResult, error) {
	rel := strings.Trim(strings.TrimPrefix(opt.Rel, "/"), "/")
	dir, parent, err := resolveBrowse(env.MusicRoot, rel)
	if err != nil {
		return BrowseResult{}, err
	}
	out := BrowseResult{Path: rel}
	if parent == "" {
		out.Parent = nil
	} else {
		out.Parent = &parent
	}
	if opt.Queue {
		if opt.QueuePathsOnly {
			paths, err := collectQueuePaths(env, rel, dir)
			if err != nil {
				return BrowseResult{}, err
			}
			out.Paths = paths
			return out, nil
		}
		tracks, err := collectQueueTracks(env, rel, dir)
		if err != nil {
			return BrowseResult{}, err
		}
		out.Tracks = tracks
		return out, nil
	}
	db, dbErr := EnsureDB(env)
	if rel == "" {
		genres, err := listGenres(env, db, dbErr)
		if err != nil {
			return BrowseResult{}, err
		}
		for _, g := range genres {
			out.Entries = append(out.Entries, BrowseEntry{
				Type:  "dir",
				Name:  g.Name,
				Count: g.Count,
				Track: Track{Path: g.Name},
			})
		}
		return out, nil
	}
	if opt.Offset == 0 {
		subs, _ := listSubdirs(dir)
		for _, sub := range subs {
			out.Entries = append(out.Entries, BrowseEntry{
				Type: "dir",
				Name: filepath.Base(sub),
				Track: Track{
					Path: filepath.ToSlash(filepath.Join(rel, filepath.Base(sub))),
				},
			})
		}
	}
	limit := opt.Limit
	if limit <= 0 {
		limit = 50
	}
	tracks, total, err := listTracksPage(env, db, dbErr, rel, dir, opt.Offset, limit)
	if err != nil {
		return BrowseResult{}, err
	}
	propagateFolderArt(env, tracks)
	for _, t := range tracks {
		out.Entries = append(out.Entries, BrowseEntry{Type: "track", Track: t})
	}
	if opt.Offset > 0 || opt.Limit > 0 {
		out.TrackTotal = total
		out.TrackOffset = opt.Offset
		out.TrackLimit = limit
	}
	return out, nil
}

func resolveBrowse(root, rel string) (dir, parent string, err error) {
	root = filepath.Clean(root)
	if rel == "" {
		return root, "", nil
	}
	dir = filepath.Join(root, filepath.FromSlash(rel))
	dir = filepath.Clean(dir)
	if !strings.HasPrefix(dir, root+string(os.PathSeparator)) && dir != root {
		return "", "", os.ErrNotExist
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return "", "", os.ErrNotExist
	}
	if strings.Contains(rel, "/") {
		parent = filepath.ToSlash(filepath.Dir(rel))
	}
	return dir, parent, nil
}

type genreCount struct {
	Name  string
	Count int
}

func listGenres(env Env, db *sql.DB, dbErr error) ([]genreCount, error) {
	entries, err := os.ReadDir(env.MusicRoot)
	if err != nil {
		return nil, err
	}
	out := make([]genreCount, 0)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		name := e.Name()
		dir := filepath.Join(env.MusicRoot, name)
		count := 0
		if db != nil && dbErr == nil {
			count, _ = CountUnder(db, dir)
		}
		if count == 0 {
			count = countAudioFilesInDir(dir)
		}
		out = append(out, genreCount{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func countAudioFilesInDir(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if playback.IsSupportedPath(path) {
			n++
		}
		return nil
	})
	return n
}

func listSubdirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}

func listTracksPage(env Env, db *sql.DB, dbErr error, rel, dir string, offset, limit int) ([]Track, int, error) {
	total := 0
	if dbErr == nil && db != nil {
		var err error
		total, err = CountInDir(db, dir)
		if err == nil && total > 0 {
			rows, err := db.Query(`
SELECT path, genre, title, artist, album, year, label, duration, art, waveform, liked
FROM tracks WHERE parent_dir=? ORDER BY path LIMIT ? OFFSET ?`, dir, limit, offset)
			if err == nil {
				defer rows.Close()
				out := make([]Track, 0)
				for rows.Next() {
					t, err := scanTrack(rows, env)
					if err == nil {
						t.Type = "track"
						out = append(out, t)
					}
				}
				return out, total, nil
			}
		}
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, err
	}
	all := make([]Track, 0)
	for _, e := range files {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if !playback.IsSupportedPath(p) {
			continue
		}
		t, _ := Meta(env, p, "")
		t.Type = "track"
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Path < all[j].Path })
	total = len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func scanTrack(rows *sql.Rows, env Env) (Track, error) {
	var t Track
	var liked int
	err := rows.Scan(
		&t.Path, &t.Genre, &t.Title, &t.Artist, &t.Album, &t.Year, &t.Label,
		&t.Duration, &t.Art, &t.Waveform, &liked,
	)
	if err != nil {
		return Track{}, err
	}
	t.Liked = liked == 1
	enrichTrackAssets(env, &t)
	applyTagGenre(&t)
	return t, nil
}

// CollectQueuePaths returns audio file paths under a folder tree.
func CollectQueuePaths(env Env, rel, dir string) ([]string, error) {
	return collectQueuePaths(env, rel, dir)
}

func collectQueuePaths(env Env, rel, dir string) ([]string, error) {
	dir = filepath.Clean(dir)
	if db, err := EnsureDB(env); err == nil {
		if paths, err := collectQueuePathsDB(db, dir); err == nil && len(paths) > 0 {
			return paths, nil
		}
	}
	return collectQueuePathsFS(env, rel, dir)
}

func collectQueuePathsDB(db *sql.DB, dir string) ([]string, error) {
	dir = filepath.Clean(dir)
	pattern := dir + string(os.PathSeparator) + "%"
	rows, err := db.Query(`SELECT path FROM tracks WHERE path LIKE ? ORDER BY path`, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil && path != "" {
			out = append(out, path)
		}
	}
	return out, nil
}

func collectQueuePathsFS(env Env, rel, dir string) ([]string, error) {
	out := make([]string, 0)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range files {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if playback.IsSupportedPath(p) {
			out = append(out, p)
		}
	}
	subs, _ := listSubdirs(dir)
	for _, sub := range subs {
		subRel := filepath.ToSlash(filepath.Join(rel, filepath.Base(sub)))
		nested, err := collectQueuePathsFS(env, subRel, sub)
		if err != nil {
			continue
		}
		out = append(out, nested...)
	}
	return out, nil
}

func collectQueueTracks(env Env, rel, dir string) ([]Track, error) {
	dir = filepath.Clean(dir)
	if db, err := EnsureDB(env); err == nil {
		if tracks, err := collectQueueTracksDB(env, db, dir); err == nil && len(tracks) > 0 {
			propagateFolderArtByDirectory(env, tracks)
			return tracks, nil
		}
	}
	tracks, err := collectQueueTracksFS(env, rel, dir)
	if err != nil {
		return nil, err
	}
	propagateFolderArtByDirectory(env, tracks)
	return tracks, nil
}

func collectQueueTracksDB(env Env, db *sql.DB, dir string) ([]Track, error) {
	dir = filepath.Clean(dir)
	pattern := dir + string(os.PathSeparator) + "%"
	rows, err := db.Query(`
SELECT path, genre, title, artist, album, year, label, duration, art, waveform, liked
FROM tracks WHERE path LIKE ? ORDER BY path`, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Track, 0)
	for rows.Next() {
		t, err := scanTrack(rows, env)
		if err == nil {
			t.Type = "track"
			out = append(out, t)
		}
	}
	return out, nil
}

func collectQueueTracksFS(env Env, rel, dir string) ([]Track, error) {
	out := make([]Track, 0)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range files {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if !playback.IsSupportedPath(p) {
			continue
		}
		t, _ := Meta(env, p, "")
		if t.Path != "" {
			t.Type = "track"
			out = append(out, t)
		}
	}
	subs, _ := listSubdirs(dir)
	for _, sub := range subs {
		subRel := filepath.ToSlash(filepath.Join(rel, filepath.Base(sub)))
		nested, err := collectQueueTracksFS(env, subRel, sub)
		if err != nil {
			continue
		}
		out = append(out, nested...)
	}
	return out, nil
}

func Genres(env Env) ([]map[string]any, error) {
	db, dbErr := EnsureDB(env)
	rows, err := listGenres(env, db, dbErr)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, g := range rows {
		out = append(out, map[string]any{"name": g.Name, "count": g.Count})
	}
	return out, nil
}

func TracksForGenre(env Env, genre string) ([]Track, error) {
	dir := filepath.Join(env.MusicRoot, genre)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, os.ErrNotExist
	}
	if db, err := EnsureDB(env); err == nil && db != nil {
		tracks, err := tracksUnderDir(env, db, dir)
		if err == nil && len(tracks) > 0 {
			propagateFolderArt(env, tracks)
			return tracks, nil
		}
	}
	cache := filepath.Join(env.TracksCacheDir, genre+".tags.json")
	if raw, err := os.ReadFile(cache); err == nil {
		var items []Track
		if json.Unmarshal(raw, &items) == nil && len(items) > 0 {
			for i := range items {
				enrichTrackAssets(env, &items[i])
				if isLiked(env, items[i].Path) {
					items[i].Liked = true
				}
			}
			return items, nil
		}
	}
	tracks, _, err := listTracksPage(env, nil, os.ErrNotExist, genre, dir, 0, 100000)
	propagateFolderArt(env, tracks)
	return tracks, err
}

func tracksUnderDir(env Env, db *sql.DB, dir string) ([]Track, error) {
	dir = filepath.Clean(dir)
	rows, err := db.Query(`
SELECT path, genre, title, artist, album, year, label, duration, art, waveform, liked
FROM tracks WHERE path = ? OR path LIKE ? ORDER BY path`, dir, dir+string(os.PathSeparator)+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Track, 0)
	for rows.Next() {
		t, err := scanTrack(rows, env)
		if err == nil {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

func propagateFolderArtByDirectory(env Env, tracks []Track) {
	if len(tracks) == 0 {
		return
	}
	byDir := make(map[string][]int)
	order := make([]string, 0)
	for i, t := range tracks {
		if t.Path == "" {
			continue
		}
		dir := filepath.Dir(t.Path)
		if _, ok := byDir[dir]; !ok {
			order = append(order, dir)
		}
		byDir[dir] = append(byDir[dir], i)
	}
	for _, dir := range order {
		idxs := byDir[dir]
		group := make([]Track, len(idxs))
		for j, i := range idxs {
			group[j] = tracks[i]
		}
		propagateFolderArt(env, group)
		for j, i := range idxs {
			tracks[i].Art = group[j].Art
			tracks[i].Thumb = group[j].Thumb
		}
	}
}

func propagateFolderArt(env Env, tracks []Track) {
	if len(tracks) == 0 {
		return
	}
	var sharedArt, sharedThumb string
	for i := range tracks {
		if sharedArt == "" && tracks[i].Art != "" {
			sharedArt = tracks[i].Art
		}
		if sharedThumb == "" && tracks[i].Thumb != "" {
			sharedThumb = tracks[i].Thumb
		}
	}
	if sharedArt == "" {
		for _, t := range tracks {
			if t.Path == "" {
				continue
			}
			if art := artCacheFind(env, t.Path); art != "" {
				sharedArt = art
				sharedThumb = resolveThumb(env, art)
				break
			}
		}
	}
	if sharedArt == "" && sharedThumb == "" {
		return
	}
	for i := range tracks {
		if tracks[i].Art == "" && sharedArt != "" {
			tracks[i].Art = sharedArt
		}
		if tracks[i].Thumb == "" && sharedThumb != "" {
			tracks[i].Thumb = sharedThumb
		}
	}
}
