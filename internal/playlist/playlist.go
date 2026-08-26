package playlist

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sebday/evoplayer/internal/library"
)

type IndexItem struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Kind    string `json:"kind"`
	Starred bool   `json:"starred"`
}

type TracksPage struct {
	Total  int             `json:"total"`
	Offset int             `json:"offset"`
	Items  []library.Track `json:"items"`
}

type StarResult struct {
	Name    string   `json:"name"`
	Starred bool     `json:"starred"`
	Stars   []string `json:"stars"`
}

type FavoriteResult struct {
	Path  string `json:"path"`
	Liked bool   `json:"liked"`
}

func ListIndex(env Env) ([]IndexItem, error) {
	if err := os.MkdirAll(env.PlaylistDir, 0o755); err != nil {
		return nil, err
	}
	stars, err := loadStars(env.PlaylistStars)
	if err != nil {
		return nil, err
	}
	pruneStars(env, stars)
	stars, _ = loadStars(env.PlaylistStars)

	db, dbErr := library.EnsureDB(env.Env)
	var out []IndexItem

	allCount, err := countLikedAll(env, db, dbErr)
	if err != nil {
		return nil, err
	}
	if allCount > 0 {
		out = append(out, IndexItem{
			Name:    "all",
			Count:   allCount,
			Kind:    "system",
			Starred: isStarred(stars, "all"),
		})
	}

	genres, err := listMusicGenres(env.MusicRoot)
	if err != nil {
		return nil, err
	}
	for _, genre := range genres {
		count, err := countLikedGenre(env, db, dbErr, genre)
		if err != nil {
			return nil, err
		}
		if count <= 0 {
			continue
		}
		out = append(out, IndexItem{
			Name:    genre,
			Count:   count,
			Kind:    playlistKind(env, genre),
			Starred: isStarred(stars, genre),
		})
	}
	out = append(out, listExtraPlaylists(env, stars, out)...)
	return out, nil
}

func TracksPageFor(env Env, name string, offset, limit int) (TracksPage, error) {
	if name == "" {
		return TracksPage{}, fmt.Errorf("playlist name required")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	db, dbErr := library.EnsureDB(env.Env)
	if dbErr == nil && db != nil {
		if page, ok, err := tracksPageFromDB(env, db, name, offset, limit); ok || err != nil {
			return page, err
		}
	}

	listPath, err := ensurePlaylistM3U(env, name)
	if err != nil {
		return TracksPage{}, err
	}
	paths, err := readM3UPaths(listPath)
	if err != nil {
		return TracksPage{}, err
	}
	total := len(paths)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	items := tracksFromPaths(env.Env, paths[offset:end], name)
	return TracksPage{Total: total, Offset: offset, Items: items}, nil
}

func TracksAll(env Env, name string) ([]library.Track, error) {
	page, err := TracksPageFor(env, name, 0, 1<<30)
	if err != nil {
		return nil, err
	}
	if page.Total <= len(page.Items) {
		return page.Items, nil
	}
	page, err = TracksPageFor(env, name, 0, page.Total)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func StarToggle(env Env, name string) (StarResult, error) {
	if name == "" {
		return StarResult{}, fmt.Errorf("playlist name required")
	}
	if _, err := ensurePlaylistM3U(env, name); err != nil {
		return StarResult{}, err
	}
	stars, err := loadStars(env.PlaylistStars)
	if err != nil {
		return StarResult{}, err
	}
	stars, starred := toggleStar(stars, name)
	if err := saveStars(env.PlaylistStars, stars); err != nil {
		return StarResult{}, err
	}
	return StarResult{Name: name, Starred: starred, Stars: stars}, nil
}

func FavoriteToggle(env Env, path string) (FavoriteResult, error) {
	if path == "" {
		return FavoriteResult{}, fmt.Errorf("path required")
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return FavoriteResult{}, fmt.Errorf("not a file: %s", path)
	}
	likes, err := readLikes(env.LikesFile)
	if err != nil {
		return FavoriteResult{}, err
	}
	liked := false
	if _, ok := likes[path]; ok {
		delete(likes, path)
	} else {
		liked = true
		title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		artist := ""
		if row, err := library.Meta(env.Env, path, ""); err == nil {
			if row.Title != "" {
				title = row.Title
			}
			artist = row.Artist
		}
		likes[path] = likeEntry{
			Title:   title,
			Artist:  artist,
			LikedAt: time.Now().Format(time.RFC3339),
		}
	}
	if err := writeLikes(env.LikesFile, likes); err != nil {
		return FavoriteResult{}, err
	}
	library.InvalidateLikesCache(env.LikesFile)
	_ = writeLikesM3U(env, filepath.Join(env.PlaylistDir, "all.m3u"))
	if genre := genreFromPath(env.MusicRoot, path); genre != "" {
		_ = writeGenreM3U(env, genre)
	}
	if db, err := library.EnsureDB(env.Env); err == nil && db != nil {
		val := 0
		if liked {
			val = 1
		}
		_, _ = db.Exec(`UPDATE tracks SET liked=? WHERE path=?`, val, path)
	}
	return FavoriteResult{Path: path, Liked: liked}, nil
}

func LoadCurrent(env Env) ([]library.Track, error) {
	m3uPath := env.currentM3U()
	paths, err := readM3UPaths(m3uPath)
	if err != nil {
		return nil, err
	}
	cachePath := env.currentTracksJSON()
	if raw, err := os.ReadFile(cachePath); err == nil {
		var items []library.Track
		if json.Unmarshal(raw, &items) == nil && currentJSONMatchesPaths(items, paths) {
			return items, nil
		}
	}
	return tracksFromPaths(env.Env, paths, ""), nil
}

func currentJSONMatchesPaths(items []library.Track, paths []string) bool {
	if len(items) == 0 || len(items) != len(paths) {
		return false
	}
	for i := range paths {
		if items[i].Path != paths[i] {
			return false
		}
	}
	return true
}

type likeEntry struct {
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	LikedAt string `json:"liked_at"`
}

func readLikes(path string) (map[string]likeEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]likeEntry{}, nil
		}
		return nil, err
	}
	out := map[string]likeEntry{}
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]likeEntry{}, nil
	}
	return out, nil
}

func writeLikes(path string, likes map[string]likeEntry) error {
	if likes == nil {
		likes = map[string]likeEntry{}
	}
	raw, err := json.Marshal(likes)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func tracksPageFromDB(env Env, db *sql.DB, name string, offset, limit int) (TracksPage, bool, error) {
	switch name {
	case "all":
		var total int
		if err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE liked=1`).Scan(&total); err != nil {
			return TracksPage{}, false, nil
		}
		rows, err := db.Query(`
SELECT path FROM tracks WHERE liked=1 ORDER BY path LIMIT ? OFFSET ?`, limit, offset)
		if err != nil {
			return TracksPage{}, true, err
		}
		defer rows.Close()
		paths, err := scanPaths(rows)
		if err != nil {
			return TracksPage{}, true, err
		}
		return TracksPage{
			Total:  total,
			Offset: offset,
			Items:  tracksFromPaths(env.Env, paths, name),
		}, true, nil
	default:
		if isGenreDir(env, name) {
			prefix := filepath.Join(env.MusicRoot, name) + string(os.PathSeparator)
			var total int
			if err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE liked=1 AND path LIKE ?`, prefix+"%").Scan(&total); err != nil || total == 0 {
				return TracksPage{}, false, nil
			}
			rows, err := db.Query(`
SELECT path FROM tracks WHERE liked=1 AND path LIKE ? ORDER BY path LIMIT ? OFFSET ?`,
				prefix+"%", limit, offset)
			if err != nil {
				return TracksPage{}, true, err
			}
			defer rows.Close()
			paths, err := scanPaths(rows)
			if err != nil {
				return TracksPage{}, true, err
			}
			return TracksPage{
				Total:  total,
				Offset: offset,
				Items:  tracksFromPaths(env.Env, paths, name),
			}, true, nil
		}
	}
	return TracksPage{}, false, nil
}

func scanPaths(rows *sql.Rows) ([]string, error) {
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

func tracksFromPaths(env library.Env, paths []string, playlistName string) []library.Track {
	return library.TracksForPaths(env, paths, playlistName)
}

func ensurePlaylistM3U(env Env, name string) (string, error) {
	switch name {
	case "all":
		out := filepath.Join(env.PlaylistDir, "all.m3u")
		if err := writeLikesM3U(env, out); err != nil {
			return "", err
		}
		return out, nil
	default:
		if isGenreDir(env, name) {
			out := filepath.Join(env.PlaylistDir, name+".m3u")
			if err := writeGenreM3U(env, name); err != nil {
				return "", err
			}
			if _, err := os.Stat(out); err != nil {
				return "", fmt.Errorf("unknown playlist: %s", name)
			}
			return out, nil
		}
	}
	out := filepath.Join(env.PlaylistDir, name+".m3u")
	if _, err := os.Stat(out); err != nil {
		return "", fmt.Errorf("unknown playlist: %s", name)
	}
	return out, nil
}

func countLikedAll(env Env, db *sql.DB, dbErr error) (int, error) {
	if dbErr == nil && db != nil && library.Ready(db) {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE liked=1`).Scan(&n); err == nil {
			return n, nil
		}
	}
	likes, err := readLikes(env.LikesFile)
	if err != nil {
		return 0, err
	}
	return len(likes), nil
}

func countLikedGenre(env Env, db *sql.DB, dbErr error, genre string) (int, error) {
	paths, err := likedPathsForGenre(env, genre)
	if err != nil {
		return 0, err
	}
	if len(paths) > 0 {
		return len(paths), nil
	}
	if dbErr == nil && db != nil && library.Ready(db) {
		prefix := filepath.Join(env.MusicRoot, genre) + string(os.PathSeparator)
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE liked=1 AND path LIKE ?`, prefix+"%").Scan(&n); err == nil && n > 0 {
			return n, nil
		}
	}
	if paths, err := readM3UPaths(filepath.Join(env.PlaylistDir, genre+".m3u")); err == nil {
		return len(paths), nil
	}
	return 0, nil
}

func listExtraPlaylists(env Env, stars []string, existing []IndexItem) []IndexItem {
	seen := map[string]struct{}{}
	for _, item := range existing {
		seen[item.Name] = struct{}{}
	}
	entries, err := os.ReadDir(env.PlaylistDir)
	if err != nil {
		return nil
	}
	var out []IndexItem
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".m3u") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".m3u")
		if name == "" || name == "all" || name == "favorites" || name == "current" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		paths, err := readM3UPaths(filepath.Join(env.PlaylistDir, e.Name()))
		if err != nil || len(paths) == 0 {
			continue
		}
		out = append(out, IndexItem{
			Name:    name,
			Count:   len(paths),
			Kind:    playlistKind(env, name),
			Starred: isStarred(stars, name),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func listMusicGenres(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var genres []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if skipDir(name) {
			continue
		}
		genres = append(genres, name)
	}
	sort.Strings(genres)
	return genres, nil
}

func skipDir(name string) bool {
	switch name {
	case ".incoming", ".Trash", ".trash", ".cache", ".git":
		return true
	}
	return strings.HasPrefix(name, ".")
}

func isGenreDir(env Env, name string) bool {
	st, err := os.Stat(filepath.Join(env.MusicRoot, name))
	return err == nil && st.IsDir()
}

func playlistKind(env Env, name string) string {
	switch name {
	case "all", "favorites":
		return "system"
	default:
		if isGenreDir(env, name) {
			return "fav"
		}
		return "other"
	}
}

func pruneStars(env Env, stars []string) {
	kept := make([]string, 0, len(stars))
	for _, name := range stars {
		if name == "all" {
			if _, err := ensurePlaylistM3U(env, "all"); err == nil {
				kept = append(kept, name)
			}
			continue
		}
		if isGenreDir(env, name) {
			if _, err := ensurePlaylistM3U(env, name); err == nil {
				kept = append(kept, name)
			}
			continue
		}
		if _, err := os.Stat(filepath.Join(env.PlaylistDir, name+".m3u")); err == nil {
			kept = append(kept, name)
		}
	}
	_ = saveStars(env.PlaylistStars, kept)
}
