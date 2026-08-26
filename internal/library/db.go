package library

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
PRAGMA synchronous=NORMAL;
CREATE TABLE IF NOT EXISTS tracks (
  path TEXT PRIMARY KEY NOT NULL,
  genre TEXT NOT NULL DEFAULT '',
  parent_dir TEXT NOT NULL,
  title TEXT,
  artist TEXT,
  album TEXT,
  year TEXT,
  label TEXT,
  duration REAL DEFAULT 0,
  art TEXT,
  waveform TEXT,
  liked INTEGER DEFAULT 0,
  mtime INTEGER NOT NULL DEFAULT 0,
  size INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_tracks_parent ON tracks(parent_dir);
CREATE INDEX IF NOT EXISTS idx_tracks_genre ON tracks(genre);
CREATE INDEX IF NOT EXISTS idx_tracks_liked ON tracks(liked);
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT
);
`

func OpenDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrateTracks(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrateTracks(db *sql.DB) error {
	for _, col := range []string{
		`ALTER TABLE tracks ADD COLUMN mtime INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tracks ADD COLUMN size INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := db.Exec(col); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
	}
	return nil
}

func Ready(db *sql.DB) bool {
	if db == nil {
		return false
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tracks`).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func EnsureDB(env Env) (*sql.DB, error) {
	if db, ok := cachedDB(env.LibraryDB); ok {
		return db, nil
	}
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return nil, err
	}
		if !Ready(db) {
			if hasTagsJSON(env) {
				if err := rebuildFromJSON(db, env); err != nil {
					_ = db.Close()
					return nil, err
				}
			}
		}
	storeCachedDB(env.LibraryDB, db)
	return db, nil
}

func Rebuild(db *sql.DB, env Env) error {
	return rebuildFromJSON(db, env)
}

func rebuildFromJSON(db *sql.DB, env Env) error {
	_, _ = db.Exec(`DELETE FROM tracks`)
	if err := ImportTagsCaches(db, env); err != nil {
		return err
	}
	if err := SyncLiked(db, env); err != nil {
		return err
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('built_at', datetime('now'))`)
	return err
}

func CountInDir(db *sql.DB, dir string) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE parent_dir=?`, dir).Scan(&n)
	return n, err
}

func CountUnder(db *sql.DB, dir string) (int, error) {
	var n int
	dir = filepath.Clean(dir)
	err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE path = ? OR path LIKE ?`, dir, dir+string(os.PathSeparator)+"%").Scan(&n)
	return n, err
}

func ListTrackPaths(env Env) ([]string, error) {
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT path FROM tracks ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		if path != "" {
			out = append(out, path)
		}
	}
	return out, rows.Err()
}

type fileStat struct {
	mtime int64
	size  int64
}

func loadFileStats(db *sql.DB, prefix string) (map[string]fileStat, error) {
	q := `SELECT path, mtime, size FROM tracks`
	var args []any
	if prefix != "" {
		q += ` WHERE path = ? OR path LIKE ?`
		args = []any{prefix, prefix + string(os.PathSeparator) + "%"}
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]fileStat)
	for rows.Next() {
		var path string
		var st fileStat
		if err := rows.Scan(&path, &st.mtime, &st.size); err != nil {
			return nil, err
		}
		out[path] = st
	}
	return out, rows.Err()
}

func upsertTrack(tx *sql.Tx, env Env, item Track, mtime, size int64) error {
	liked := 0
	if item.Liked || isLiked(env, item.Path) {
		liked = 1
	}
	_, err := tx.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked,mtime,size)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(path) DO UPDATE SET
  genre=excluded.genre,
  parent_dir=excluded.parent_dir,
  title=excluded.title,
  artist=excluded.artist,
  album=excluded.album,
  year=excluded.year,
  label=excluded.label,
  duration=excluded.duration,
  mtime=excluded.mtime,
  size=excluded.size`,
		item.Path, item.Genre, filepath.Dir(item.Path),
		item.Title, item.Artist, item.Album, item.Year, item.Label,
		item.Duration, item.Art, item.Waveform, liked, mtime, size,
	)
	return err
}

func pruneMissing(tx *sql.Tx, prefix string, seen map[string]struct{}) (int, error) {
	stats, err := loadFileStatsTx(tx, prefix)
	if err != nil {
		return 0, err
	}
	n := 0
	for path := range stats {
		if _, ok := seen[path]; ok {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM tracks WHERE path=?`, path); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func loadFileStatsTx(tx *sql.Tx, prefix string) (map[string]fileStat, error) {
	q := `SELECT path, mtime, size FROM tracks`
	var args []any
	if prefix != "" {
		q += ` WHERE path = ? OR path LIKE ?`
		args = []any{prefix, prefix + string(os.PathSeparator) + "%"}
	}
	rows, err := tx.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]fileStat)
	for rows.Next() {
		var path string
		var st fileStat
		if err := rows.Scan(&path, &st.mtime, &st.size); err != nil {
			return nil, err
		}
		out[path] = st
	}
	return out, rows.Err()
}
