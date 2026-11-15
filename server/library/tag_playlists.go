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
	if db == nil || !Ready(db) {
		return nil, nil
	}
	rows, err := db.Query(`
SELECT replace(trim(COALESCE(genre,'')), char(0), '') AS name, COUNT(*)
FROM tracks
WHERE liked=1 AND replace(trim(COALESCE(genre,'')), char(0), '') != ''
GROUP BY replace(trim(COALESCE(genre,'')), char(0), '')
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
	if db == nil || !Ready(db) {
		return nil, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	rows, err := db.Query(`
SELECT path FROM tracks
WHERE liked=1 AND replace(trim(COALESCE(genre,'')), char(0), '') = ?
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
