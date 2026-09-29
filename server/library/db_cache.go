package library

import (
	"database/sql"
	"sync"
)

var (
	dbMu    sync.Mutex
	dbCache = map[string]*sql.DB{}
)
