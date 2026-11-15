package library

import (
	"path/filepath"
	"testing"
)

func TestYearIndex(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "library.sqlite3")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	env := Env{LibraryDB: dbPath, MusicRoot: filepath.Join(dir, "music"), CacheDir: filepath.Join(dir, "cache")}
	t.Cleanup(func() {
		db.Close()
		dbMu.Lock()
		if cached, ok := dbCache[dbPath]; ok {
			cached.Close()
			delete(dbCache, dbPath)
		}
		dbMu.Unlock()
	})
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []Track{
		{Path: filepath.Join(dir, "a.mp3"), Title: "A", Year: "2005"},
		{Path: filepath.Join(dir, "b.mp3"), Title: "B", Year: "2005-06-01"},
		{Path: filepath.Join(dir, "c.mp3"), Title: "C", Year: "1999"},
	} {
		if err := upsertTrack(tx, env, item, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	lists, err := Index(env, "year")
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 2 || lists[0].Name != "2005" || lists[0].Count != 2 || lists[1].Name != "1999" {
		t.Fatalf("years = %+v", lists)
	}
	page, err := Entity(env, "year", "2005", "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Name != "2005" || page.Total != 2 {
		t.Fatalf("page = %+v", page)
	}
	paths, err := PathsForField(db, "year", "2005")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
}
