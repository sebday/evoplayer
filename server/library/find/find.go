package find

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/library"
)

func Tracks(cacheDir, mode, query string) ([]map[string]any, error) {
	dbPath := filepath.Join(filepath.Dir(cacheDir), "library.sqlite3")
	return tracksFromSQLite(dbPath, mode, query)
}

func tracksFromSQLite(dbPath, mode, query string) ([]map[string]any, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	db, err := sql.Open("sqlite", library.SQLiteDSN(dbPath))
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
