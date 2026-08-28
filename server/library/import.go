package library

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func hasTagsJSON(env Env) bool {
	matches, err := filepath.Glob(filepath.Join(env.TracksCacheDir, "*.tags.json"))
	return err == nil && len(matches) > 0
}

func ImportTagsCaches(db *sql.DB, env Env) error {
	matches, err := filepath.Glob(filepath.Join(env.TracksCacheDir, "*.tags.json"))
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, cache := range matches {
		raw, err := os.ReadFile(cache)
		if err != nil {
			continue
		}
		var items []Track
		if err := json.Unmarshal(raw, &items); err != nil {
			continue
		}
		for _, item := range items {
			if item.Path == "" {
				continue
			}
			st, err := os.Stat(item.Path)
			if err != nil || st.IsDir() {
				continue
			}
			genre := item.Genre
			if genre == "" {
				genre = genreFromPath(env.MusicRoot, item.Path)
			}
			item.Genre = genre
			if err := upsertTrack(tx, env, item, st.ModTime().UnixNano(), st.Size()); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func SyncLiked(db *sql.DB, env Env) error {
	_ = RelocateLibraryPaths(env)
	raw, err := os.ReadFile(env.LikesFile)
	if err != nil {
		_, _ = db.Exec(`UPDATE tracks SET liked=0`)
		return nil
	}
	var likes map[string]any
	if json.Unmarshal(raw, &likes) != nil {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE tracks SET liked=0`); err != nil {
		_ = tx.Rollback()
		return err
	}
	paths := make([]string, 0, len(likes))
	for path := range likes {
		if path != "" {
			paths = append(paths, path)
		}
	}
	const batch = 400
	for i := 0; i < len(paths); i += batch {
		end := i + batch
		if end > len(paths) {
			end = len(paths)
		}
		chunk := paths[i:end]
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, p := range chunk {
			placeholders[j] = "?"
			args[j] = p
		}
		q := `UPDATE tracks SET liked=1 WHERE path IN (` + strings.Join(placeholders, ",") + `)`
		if _, err := tx.Exec(q, args...); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func genreFromPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
