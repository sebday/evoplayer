package library

import (
	"path/filepath"
	"testing"
)

func TestGenreIndexKeepsSpellings(t *testing.T) {
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
	tracks := []Track{
		{Path: filepath.Join(dir, "a.mp3"), Title: "A", Genre: "Dubstep"},
		{Path: filepath.Join(dir, "b.mp3"), Title: "B", Genre: "Dubstep"},
		{Path: filepath.Join(dir, "c.mp3"), Title: "C", Genre: "dubstep"},
		{Path: filepath.Join(dir, "d.mp3"), Title: "D", Genre: "Drum & Bass"},
		{Path: filepath.Join(dir, "e.mp3"), Title: "E", Genre: "drum&bass"},
		{Path: filepath.Join(dir, "f.mp3"), Title: "F", Genre: "JUNGLE"},
		{Path: filepath.Join(dir, "g.mp3"), Title: "G", Genre: "Jungle"},
		{Path: filepath.Join(dir, "h.mp3"), Title: "H", Genre: "jungle"},
	}
	for _, item := range tracks {
		if err := upsertTrack(tx, env, item, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	lists, err := Index(env, "genre")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, row := range lists {
		got[row.Name] = row.Count
	}
	want := map[string]int{
		"Dubstep":     2,
		"dubstep":     1,
		"Drum & Bass": 1,
		"drum&bass":   1,
		"JUNGLE":      1,
		"Jungle":      1,
		"jungle":      1,
	}
	if len(got) != len(want) {
		t.Fatalf("genres = %+v", lists)
	}
	for name, count := range want {
		if got[name] != count {
			t.Fatalf("%s = %d, genres = %+v", name, got[name], lists)
		}
	}
	page, err := Entity(env, "genre", "dubstep", "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Name != "dubstep" || page.Total != 1 {
		t.Fatalf("page = %+v", page)
	}
	paths, err := PathsForField(db, "genre", "Dubstep")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
}
