package library

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

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

// GenreFromPath returns the top-level library folder containing path.
func GenreFromPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}
	return strings.Split(rel, string(os.PathSeparator))[0]
}
