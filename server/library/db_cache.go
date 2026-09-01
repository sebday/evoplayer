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
