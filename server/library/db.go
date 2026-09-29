package library

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
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

// SQLiteDSN applies per-connection pragmas so every pooled connection waits on locks.
func SQLiteDSN(path string) string {
	if path == "" {
		return ""
	}
	return path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
}

func OpenDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", SQLiteDSN(path))
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
	n, err := countTracks(db)
	return err == nil && n > 0
}

func countTracks(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM tracks`).Scan(&n)
	return n, err
}

func EnsureDB(env Env) (*sql.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db, ok := dbCache[env.LibraryDB]; ok {
		return db, nil
	}
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return nil, err
	}
	if Ready(db) {
		_ = SyncLiked(db, env)
	}
	dbCache[env.LibraryDB] = db
	return db, nil
}

// PathPrefixRange returns bounds such that lo <= path < hi selects paths strictly under dir.
func PathPrefixRange(dir string) (lo, hi string) {
	sep := string(os.PathSeparator)
	dir = strings.TrimRight(filepath.Clean(dir), sep)
	return dir + sep, dir + string(rune(os.PathSeparator+1))
}

const pathUnderSQL = `(path = ? OR (path >= ? AND path < ?))`

func pathUnderArgs(dir string) []any {
	dir = filepath.Clean(dir)
	lo, hi := PathPrefixRange(dir)
	return []any{dir, lo, hi}
}

func CountInDir(db *sql.DB, dir string) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE parent_dir=?`, dir).Scan(&n)
	return n, err
}

func CountUnder(db *sql.DB, dir string) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE `+pathUnderSQL, pathUnderArgs(dir)...).Scan(&n)
	return n, err
}

func ListTrackPaths(env Env) ([]string, error) {
	return listTrackPathsQuery(env, "")
}

// ListTrackPathsInDir returns indexed track paths under a library-relative folder (e.g. drum&bass/soundcloud).
func ListTrackPathsInDir(env Env, rel string) ([]string, error) {
	rel = strings.Trim(strings.TrimPrefix(rel, "/"), "/")
	if rel == "" {
		return ListTrackPaths(env)
	}
	return listTrackPathsQuery(env, filepath.Join(env.MusicRoot, filepath.FromSlash(rel)))
}

func listTrackPathsQuery(env Env, dir string) ([]string, error) {
	db, err := OpenDB(env.LibraryDB)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	q := `SELECT path FROM tracks ORDER BY path`
	var args []any
	if dir != "" {
		q = `SELECT path FROM tracks WHERE ` + pathUnderSQL + ` ORDER BY path`
		args = pathUnderArgs(dir)
	}
	rows, err := db.Query(q, args...)
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

type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func loadFileStats(q queryer, prefix string) (map[string]fileStat, error) {
	query := `SELECT path, mtime, size FROM tracks`
	var args []any
	if prefix != "" {
		query += ` WHERE ` + pathUnderSQL
		args = pathUnderArgs(prefix)
	}
	rows, err := q.Query(query, args...)
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

// replaceTrack swaps the row for from with item in one transaction.
func replaceTrack(db *sql.DB, env Env, from string, item Track, mtime, size int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if from != "" && from != item.Path {
		if _, err := tx.Exec(`DELETE FROM tracks WHERE path=?`, from); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := upsertTrack(tx, env, item, mtime, size); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func pruneMissing(tx *sql.Tx, prefix string, seen map[string]struct{}) (int, error) {
	stats, err := loadFileStats(tx, prefix)
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
