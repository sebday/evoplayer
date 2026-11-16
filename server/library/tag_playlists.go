package library

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sebday/evoplayer/server/tags"
)

type TagPlaylist struct {
	Name  string
	Count int
}

// LikedGenrePlaylists lists genre tags that appear on at least one liked track.
// The folder a file sits in is not a genre.
func LikedGenrePlaylists(db *sql.DB) ([]TagPlaylist, error) {
	return likedColumnPlaylists(db, "genre")
}

// TagPlaylists lists tags on library tracks.
// likedOnly keeps tags that appear on a liked track, and counts only those tracks.
func TagPlaylists(db *sql.DB, likedOnly bool) ([]TagPlaylist, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	liked := ""
	if likedOnly {
		liked = "tracks.liked=1 AND "
	}
	rows, err := db.Query(`
SELECT replace(trim(track_tags.tag), char(0), '') AS name, COUNT(DISTINCT track_tags.path)
FROM track_tags
JOIN tracks ON tracks.path = track_tags.path
WHERE ` + liked + `replace(trim(track_tags.tag), char(0), '') != ''
GROUP BY replace(trim(track_tags.tag), char(0), '')
ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagPlaylist
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		name = strings.TrimSpace(name)
		if name == "" || reservedTag(tags.TagKey(name)) {
			continue
		}
		out = append(out, TagPlaylist{Name: name, Count: count})
	}
	return out, rows.Err()
}

func likedColumnPlaylists(db *sql.DB, col string) ([]TagPlaylist, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	rows, err := db.Query(`
SELECT replace(trim(COALESCE(` + col + `,'')), char(0), '') AS name, COUNT(*)
FROM tracks
WHERE liked=1 AND replace(trim(COALESCE(` + col + `,'')), char(0), '') != ''
GROUP BY replace(trim(COALESCE(` + col + `,'')), char(0), '')
ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagPlaylist
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, err
		}
		name = strings.TrimSpace(name)
		if name == "" || reservedTag(tags.TagKey(name)) {
			continue
		}
		out = append(out, TagPlaylist{Name: name, Count: count})
	}
	return out, rows.Err()
}

// LikedPathsForGenreTag returns liked tracks whose genre tag is name.
func LikedPathsForGenreTag(db *sql.DB, name string) ([]string, error) {
	return likedPathsForColumn(db, "genre", name)
}

// PathsForTag returns library tracks that carry tag name.
// likedOnly drops tracks that are not liked.
func PathsForTag(db *sql.DB, name string, likedOnly bool) ([]string, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	liked := ""
	if likedOnly {
		liked = "tracks.liked=1 AND "
	}
	rows, err := db.Query(`
SELECT tracks.path
FROM track_tags
JOIN tracks ON tracks.path = track_tags.path
WHERE `+liked+`replace(trim(track_tags.tag), char(0), '') = ?
ORDER BY tracks.path`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

// PathsForArtist returns tracks credited to name.
// likedOnly drops tracks that are not liked.
func PathsForArtist(db *sql.DB, name string, likedOnly bool) ([]string, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	lname := strings.ToLower(name)
	liked := ""
	if likedOnly {
		liked = " AND liked=1"
	}
	rows, err := db.Query(`
SELECT path, COALESCE(artist,'') FROM tracks
WHERE lower(artist) LIKE ? ESCAPE '\'`+liked+`
ORDER BY path`, likeEscape(lname))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var path, artist string
		if err := rows.Scan(&path, &artist); err != nil {
			return nil, err
		}
		if creditedTo(artist, lname) {
			out = append(out, path)
		}
	}
	return out, rows.Err()
}

// PathsForLabel returns tracks released on the label.
// likedOnly drops tracks that are not liked.
func PathsForLabel(db *sql.DB, name string, likedOnly bool) ([]string, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	liked := ""
	if likedOnly {
		liked = " AND liked=1"
	}
	rows, err := db.Query(`
SELECT path FROM tracks
WHERE lower(trim(COALESCE(label,''))) = lower(?)`+liked+`
ORDER BY path`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

// PathsForAlbum returns tracks on the album.
// likedOnly drops tracks that are not liked.
func PathsForAlbum(db *sql.DB, name string, likedOnly bool) ([]string, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	liked := ""
	if likedOnly {
		liked = " AND liked=1"
	}
	rows, err := db.Query(`
SELECT path FROM tracks
WHERE lower(trim(COALESCE(album,''))) = lower(?)`+liked+`
ORDER BY path`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

func likedPathsForColumn(db *sql.DB, col, name string) ([]string, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	rows, err := db.Query(`
SELECT path FROM tracks
WHERE liked=1 AND replace(trim(COALESCE(`+col+`,'')), char(0), '') = ?
ORDER BY path`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

// SharedTagList is the tag list when every path has the same tags.
// The bool is false when the paths disagree, so a blank editor does not wipe them.
func SharedTagList(db *sql.DB, paths []string) (string, bool, error) {
	if db == nil || !Ready(db) || len(paths) == 0 {
		return "", true, nil
	}
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	if len(unique) == 0 {
		return "", true, nil
	}
	grouped := map[string][]string{}
	const chunk = 400
	for start := 0; start < len(unique); start += chunk {
		end := start + chunk
		if end > len(unique) {
			end = len(unique)
		}
		part := unique[start:end]
		hold := strings.TrimRight(strings.Repeat("?,", len(part)), ",")
		args := make([]any, len(part))
		for i, path := range part {
			args[i] = path
		}
		rows, err := db.Query(`SELECT path, tag FROM track_tags WHERE path IN (`+hold+`) ORDER BY rowid`, args...)
		if err != nil {
			return "", false, err
		}
		for rows.Next() {
			var path, tag string
			if err := rows.Scan(&path, &tag); err != nil {
				rows.Close()
				return "", false, err
			}
			grouped[path] = append(grouped[path], tag)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return "", false, err
		}
		rows.Close()
	}
	first := tags.FormatTagList(grouped[unique[0]])
	for _, path := range unique[1:] {
		if tags.FormatTagList(grouped[path]) != first {
			return "", false, nil
		}
	}
	return first, true, nil
}

func reservedTag(key string) bool {
	switch key {
	case "all", "current", "mixes", "favorites":
		return true
	default:
		return false
	}
}

// PathsForField returns every track whose artist or label matches value, ignoring case.
// Genre matches the tag text exactly.
func PathsForField(db *sql.DB, field, value string) ([]string, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	if field == "year" {
		return pathsForYear(db, value)
	}
	col, err := fieldColumn(field)
	if err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("%s required", field)
	}
	match := `lower(` + col + `)=lower(?)`
	if field == "genre" {
		match = `trim(` + col + `)=?`
	}
	rows, err := db.Query(`SELECT path FROM tracks WHERE `+match+` ORDER BY path`, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

func pathsForYear(db *sql.DB, value string) ([]string, error) {
	want := yearGroup(value)
	if want == "" {
		return nil, fmt.Errorf("year required")
	}
	rows, err := db.Query(`SELECT path, COALESCE(year,'') FROM tracks WHERE trim(COALESCE(year,'')) != '' ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var path, year string
		if err := rows.Scan(&path, &year); err != nil {
			return nil, err
		}
		if yearGroup(year) == want {
			out = append(out, path)
		}
	}
	return out, rows.Err()
}

func fieldColumn(field string) (string, error) {
	switch field {
	case "artist", "album", "label", "genre", "year":
		return field, nil
	default:
		return "", fmt.Errorf("unsupported field: %s", field)
	}
}

// UpdateField writes one tag field on each path.
func UpdateField(env Env, field, value string, paths []string) (int, error) {
	key, err := fieldWriteKey(field)
	if err != nil {
		return 0, err
	}
	value = strings.TrimSpace(value)
	n := 0
	seen := map[string]struct{}{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		if _, err := writeTagFields(env, path, map[string]string{key: value}); err != nil {
			return n, err
		}
		n++
	}
	if n == 0 {
		return 0, fmt.Errorf("path required")
	}
	return n, nil
}

func fieldWriteKey(field string) (string, error) {
	switch field {
	case "artist", "album", "genre", "year":
		return field, nil
	case "label":
		return "publisher", nil
	default:
		return "", fmt.Errorf("unsupported field: %s", field)
	}
}
