package library

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/playback"
)

// DeleteTrack removes one audio file from the music root and the library index.
func DeleteTrack(env Env, path string) (string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return "", fmt.Errorf("path required")
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("not a file: %s", path)
	}
	if !playback.IsSupportedPath(path) {
		return "", fmt.Errorf("unsupported file: %s", path)
	}
	root := filepath.Clean(env.MusicRoot)
	if root == "" || root == "." {
		return "", fmt.Errorf("music root not set")
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("not under music root: %s", path)
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	if db, err := EnsureDB(env); err == nil && db != nil {
		_ = forgetTrackRow(db, path)
	}
	return path, nil
}

func forgetTrackRow(db *sql.DB, path string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM track_tags WHERE path=?`, path); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tracks WHERE path=?`, path); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
