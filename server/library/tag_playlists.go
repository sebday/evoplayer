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

// LikedTagPlaylists lists tags that appear on at least one liked track.
// A tag whose name matches a genre folder is skipped so it does not duplicate that playlist.
func LikedTagPlaylists(db *sql.DB, env Env) ([]TagPlaylist, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	rows, err := db.Query(`
SELECT t.tag_key, t.tag, t.path
FROM track_tags t
JOIN tracks tr ON tr.path = t.path
WHERE tr.liked = 1
ORDER BY t.rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	skip := genreFolderKeys(env)
	type acc struct {
		name  string
		paths map[string]struct{}
	}
	order := []string{}
	byKey := map[string]*acc{}
	for rows.Next() {
		var key, tag, path string
		if err := rows.Scan(&key, &tag, &path); err != nil {
			return nil, err
		}
		if key == "" || reservedTag(key) {
			continue
		}
		if _, blocked := skip[key]; blocked {
			continue
		}
		item := byKey[key]
		if item == nil {
			item = &acc{name: tag, paths: map[string]struct{}{}}
			byKey[key] = item
			order = append(order, key)
		}
		item.paths[path] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]TagPlaylist, 0, len(order))
	for _, key := range order {
		item := byKey[key]
		if len(item.paths) == 0 {
			continue
		}
		out = append(out, TagPlaylist{Name: item.name, Count: len(item.paths)})
	}
	return out, nil
}

// LikedPathsForTag returns liked tracks carrying the tag name.
func LikedPathsForTag(db *sql.DB, name string) ([]string, error) {
	if db == nil || !Ready(db) {
		return nil, nil
	}
	key := tags.TagKey(name)
	if key == "" {
		return nil, nil
	}
	rows, err := db.Query(`
SELECT tr.path
FROM track_tags t
JOIN tracks tr ON tr.path = t.path
WHERE tr.liked = 1 AND t.tag_key = ?
ORDER BY tr.path`, key)
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

func genreFolderKeys(env Env) map[string]struct{} {
	out := map[string]struct{}{}
	for _, name := range GenreChoices(env) {
		if key := tags.TagKey(name); key != "" {
			out[key] = struct{}{}
		}
	}
	return out
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
	rows, err := db.Query(`SELECT path FROM tracks WHERE lower(`+col+`)=lower(?) ORDER BY path`, value)
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
	case "artist", "label", "genre", "year":
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
	case "artist", "genre", "year":
		return field, nil
	case "label":
		return "publisher", nil
	default:
		return "", fmt.Errorf("unsupported field: %s", field)
	}
}
