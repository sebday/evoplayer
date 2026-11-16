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

	"github.com/sebday/evoplayer/server/library"
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

	currentCount := 0
	if paths, err := ReadCurrentPaths(env); err == nil {
		currentCount = len(paths)
	}
	out = append(out, IndexItem{
		Name:    "current",
		Count:   currentCount,
		Kind:    "system",
		Starred: isStarred(stars, "current"),
	})

	mixCount, err := countLikedMixes(env, db, dbErr)
	if err != nil {
		return nil, err
	}
	if mixCount > 0 {
		out = append(out, IndexItem{
			Name:    "mixes",
			Count:   mixCount,
			Kind:    "system",
			Starred: isStarred(stars, "mixes"),
		})
	}

	if dbErr == nil && db != nil {
		genres, err := library.LikedGenrePlaylists(db)
		if err != nil {
			return nil, err
		}
		seen := map[string]struct{}{}
		for _, genre := range genres {
			seen[genre.Name] = struct{}{}
			out = append(out, IndexItem{
				Name:    genre.Name,
				Count:   genre.Count,
				Kind:    "genre",
				Starred: isStarred(stars, genre.Name),
			})
		}
		tagLists, err := library.TagPlaylists(db, LikedOnly(env))
		if err != nil {
			return nil, err
		}
		for _, tag := range tagLists {
			if _, ok := seen[tag.Name]; ok {
				continue
			}
			out = append(out, IndexItem{
				Name:    tag.Name,
				Count:   tag.Count,
				Kind:    "tag",
				Starred: isStarred(stars, tag.Name),
			})
			seen[tag.Name] = struct{}{}
		}
		if err := appendStarredPlaylists(env, db, stars, seen, &out, LikedOnly(env)); err != nil {
			return nil, err
		}
	}
	out = append(out, listExtraPlaylists(env, stars, out)...)
	sortPlaylists(out)
	return out, nil
}

func appendStarredPlaylists(env Env, db *sql.DB, stars []string, seen map[string]struct{}, out *[]IndexItem, likedOnly bool) error {
	if err := appendNamedPlaylists(db, stars, seen, out, "artist", library.ArtistStarNames(env.Env), likedOnly, library.PathsForArtist); err != nil {
		return err
	}
	return appendNamedPlaylists(db, stars, seen, out, "album", library.AlbumStarNames(env.Env), likedOnly, library.PathsForAlbum)
}

func appendNamedPlaylists(db *sql.DB, stars []string, seen map[string]struct{}, out *[]IndexItem, kind string, names []string, likedOnly bool, pathsFor func(*sql.DB, string, bool) ([]string, error)) error {
	for _, name := range names {
		if nameTaken(seen, name) || reservedPlaylist(name) {
			continue
		}
		paths, err := pathsFor(db, name, likedOnly)
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			continue
		}
		seen[name] = struct{}{}
		*out = append(*out, IndexItem{
			Name:    name,
			Count:   len(paths),
			Kind:    kind,
			Starred: isStarred(stars, name),
		})
	}
	return nil
}

func nameTaken(seen map[string]struct{}, name string) bool {
	if _, ok := seen[name]; ok {
		return true
	}
	low := strings.ToLower(name)
	for have := range seen {
		if strings.ToLower(have) == low {
			return true
		}
	}
	return false
}

func reservedPlaylist(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "all", "current", "mixes", "favorites":
		return true
	default:
		return false
	}
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
	if name == "current" {
		paths, err := ReadCurrentPaths(env)
		if err != nil {
			return TracksPage{Offset: offset}, nil
		}
		return paginatePaths(env, paths, name, offset, limit), nil
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
	paths, err = orderedPaths(env, name, paths)
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
	likes, _, err := favoriteLikes(env, path)
	if err != nil {
		return FavoriteResult{}, err
	}
	_, liked := likes[path]
	return writeFavorite(env, path, likes, !liked)
}

// SetFavorite sets the like on a library file without flipping it.
func SetFavorite(env Env, path string, liked bool) (FavoriteResult, error) {
	likes, _, err := favoriteLikes(env, path)
	if err != nil {
		return FavoriteResult{}, err
	}
	if _, has := likes[path]; has == liked {
		return FavoriteResult{Path: path, Liked: liked}, nil
	}
	return writeFavorite(env, path, likes, liked)
}

func favoriteLikes(env Env, path string) (map[string]likeEntry, os.FileInfo, error) {
	if path == "" {
		return nil, nil, fmt.Errorf("path required")
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return nil, nil, fmt.Errorf("not a file: %s", path)
	}
	likes, err := readLikes(env.LikesFile)
	if err != nil {
		return nil, nil, err
	}
	return likes, st, nil
}

func writeFavorite(env Env, path string, likes map[string]likeEntry, liked bool) (FavoriteResult, error) {
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	artist := ""
	genre := ""
	if row, err := library.Meta(env.Env, path, ""); err == nil {
		if row.Title != "" {
			title = row.Title
		}
		artist = row.Artist
		genre = strings.TrimSpace(row.Genre)
	}
	if liked {
		likes[path] = likeEntry{
			Title:   title,
			Artist:  artist,
			LikedAt: time.Now().Format(time.RFC3339),
		}
	} else {
		delete(likes, path)
	}
	if err := writeLikes(env.LikesFile, likes); err != nil {
		return FavoriteResult{}, err
	}
	library.InvalidateLikesCache(env.LikesFile)
	if db, err := library.EnsureDB(env.Env); err == nil && db != nil {
		val := 0
		if liked {
			val = 1
		}
		_, _ = db.Exec(`UPDATE tracks SET liked=? WHERE path=?`, val, path)
	}
	_ = writeLikesM3U(env, filepath.Join(env.PlaylistDir, "all.m3u"))
	_ = writeMixesM3U(env)
	if genre != "" {
		_ = writeGenreM3U(env, genre)
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
		if json.Unmarshal(raw, &items) == nil && currentJSONMatchesPaths(items, paths) && currentJSONHasMeta(items) {
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

func currentJSONHasMeta(items []library.Track) bool {
	if len(items) == 0 {
		return false
	}
	n := len(items)
	if n > 12 {
		n = 12
	}
	for i := 0; i < n; i++ {
		if strings.TrimSpace(items[i].Artist) != "" || strings.TrimSpace(items[i].Album) != "" {
			return true
		}
		title := strings.TrimSpace(items[i].Title)
		if title != "" && !looksLikeAudioFilename(title) {
			return true
		}
	}
	return false
}

func looksLikeAudioFilename(title string) bool {
	lower := strings.ToLower(title)
	for _, ext := range []string{".mp3", ".flac", ".ogg", ".m4a", ".wav", ".opus", ".aac", ".wma"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
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
	return library.WriteFileAtomic(path, raw, 0o644)
}

func tracksPageFromDB(env Env, db *sql.DB, name string, offset, limit int) (TracksPage, bool, error) {
	switch name {
	case "mixes":
		paths, err := library.LikedMixPaths(db, env.Env)
		if err != nil {
			return TracksPage{}, true, err
		}
		paths, err = orderedPaths(env, name, paths)
		if err != nil {
			return TracksPage{}, true, err
		}
		return paginatePaths(env, paths, name, offset, limit), true, nil
	case "all":
		rows, err := db.Query(`SELECT path FROM tracks WHERE liked=1 ORDER BY path`)
		if err != nil {
			return TracksPage{}, true, err
		}
		defer rows.Close()
		paths, err := scanPaths(rows)
		if err != nil {
			return TracksPage{}, true, err
		}
		paths, err = orderedPaths(env, name, paths)
		if err != nil {
			return TracksPage{}, true, err
		}
		return paginatePaths(env, paths, name, offset, limit), true, nil
	default:
		if paths, err := library.LikedPathsForGenreTag(db, name); err != nil {
			return TracksPage{}, true, err
		} else if len(paths) > 0 {
			paths, err = orderedPaths(env, name, paths)
			if err != nil {
				return TracksPage{}, true, err
			}
			return paginatePaths(env, paths, name, offset, limit), true, nil
		}
		if paths, err := library.PathsForTag(db, name, LikedOnly(env)); err != nil {
			return TracksPage{}, true, err
		} else if len(paths) > 0 {
			paths, err = orderedPaths(env, name, paths)
			if err != nil {
				return TracksPage{}, true, err
			}
			return paginatePaths(env, paths, name, offset, limit), true, nil
		}
		if library.ArtistStarred(env.Env, name) {
			paths, err := library.PathsForArtist(db, name, LikedOnly(env))
			if err != nil {
				return TracksPage{}, true, err
			}
			if len(paths) > 0 {
				paths, err = orderedPaths(env, name, paths)
				if err != nil {
					return TracksPage{}, true, err
				}
				return paginatePaths(env, paths, name, offset, limit), true, nil
			}
		}
		if library.AlbumStarred(env.Env, name) {
			paths, err := library.PathsForAlbum(db, name, LikedOnly(env))
			if err != nil {
				return TracksPage{}, true, err
			}
			if len(paths) > 0 {
				paths, err = orderedPaths(env, name, paths)
				if err != nil {
					return TracksPage{}, true, err
				}
				return paginatePaths(env, paths, name, offset, limit), true, nil
			}
		}
	}
	return TracksPage{}, false, nil
}

func paginatePaths(env Env, paths []string, name string, offset, limit int) TracksPage {
	total := len(paths)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return TracksPage{
		Total:  total,
		Offset: offset,
		Items:  tracksFromPaths(env.Env, paths[offset:end], name),
	}
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
	case "mixes":
		out := filepath.Join(env.PlaylistDir, "mixes.m3u")
		if err := writeMixesM3U(env); err != nil {
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

func countLikedMixes(env Env, db *sql.DB, dbErr error) (int, error) {
	if dbErr != nil {
		db = nil
	}
	paths, err := library.LikedMixPaths(db, env.Env)
	if err != nil {
		return 0, err
	}
	return len(paths), nil
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
		if name == "" || name == "all" || name == "favorites" || name == "current" || name == "mixes" {
			continue
		}
		if _, ok := seen[name]; ok || isGenreDir(env, name) {
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

func isGenreDir(env Env, name string) bool {
	st, err := os.Stat(filepath.Join(env.MusicRoot, name))
	return err == nil && st.IsDir()
}

func playlistKind(env Env, name string) string {
	switch name {
	case "all", "favorites", "mixes":
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
		if name == "all" || name == "mixes" {
			if _, err := ensurePlaylistM3U(env, name); err == nil {
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
