package find

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func Tracks(cacheDir, mode, query string) ([]map[string]any, error) {
	dbPath := filepath.Join(filepath.Dir(cacheDir), "library.sqlite3")
	if items, err := tracksFromSQLite(dbPath, mode, query); err == nil && len(items) > 0 {
		return items, nil
	}
	return tracksFromJSON(cacheDir, mode, query)
}

func tracksFromSQLite(dbPath, mode, query string) ([]map[string]any, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT path, genre, title, artist, album, year, label, duration FROM tracks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	needle := strings.ToLower(query)
	out := make([]map[string]any, 0)
	for rows.Next() {
		var path, genre, title, artist, album, year, label string
		var duration float64
		if err := rows.Scan(&path, &genre, &title, &artist, &album, &year, &label, &duration); err != nil {
			continue
		}
		item := map[string]any{
			"path": path, "genre": genre, "title": title, "artist": artist,
			"album": album, "year": year, "label": label, "duration": duration,
		}
		if matchItem(item, mode, query, needle) {
			out = append(out, item)
		}
	}
	return out, rows.Err()
}

func tracksFromJSON(cacheDir, mode, query string) ([]map[string]any, error) {
	needle := strings.ToLower(query)
	matches, err := filepath.Glob(filepath.Join(cacheDir, "*.tags.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	seen := map[string]struct{}{}
	out := make([]map[string]any, 0)
	for _, cache := range matches {
		raw, err := os.ReadFile(cache)
		if err != nil {
			continue
		}
		var items []map[string]any
		if err := json.Unmarshal(raw, &items); err != nil {
			continue
		}
		for _, item := range items {
			path, _ := item["path"].(string)
			if path == "" {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			if !matchItem(item, mode, query, needle) {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, item)
		}
	}
	return out, nil
}

func matchItem(item map[string]any, mode, query, needle string) bool {
	switch mode {
	case "artist":
		return strings.Contains(strings.ToLower(str(item["artist"])), needle)
	case "album":
		return strings.Contains(strings.ToLower(str(item["album"])), needle)
	case "label":
		return strings.Contains(strings.ToLower(str(item["label"])), needle)
	case "genre":
		return strings.Contains(strings.ToLower(str(item["genre"])), needle)
	case "year":
		return str(item["year"]) == query
	default:
		hay := strings.ToLower(strings.Join([]string{
			str(item["title"]),
			str(item["artist"]),
			str(item["album"]),
			str(item["genre"]),
			str(item["year"]),
			str(item["label"]),
		}, " "))
		return strings.Contains(hay, needle)
	}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return ""
	}
}
