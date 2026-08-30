package library

import (
	"database/sql"
	"sync"
)

var (
	dbMu    sync.Mutex
	dbCache = map[string]*sql.DB{}
)

func cachedDB(path string) (*sql.DB, bool) {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, ok := dbCache[path]
	return db, ok
}

func storeCachedDB(path string, db *sql.DB) {
	dbMu.Lock()
	dbCache[path] = db
	dbMu.Unlock()
}

// CloseCachedDB closes a cached pool (tests only).
func CloseCachedDB(path string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db, ok := dbCache[path]; ok {
		_ = db.Close()
		delete(dbCache, path)
	}
}
