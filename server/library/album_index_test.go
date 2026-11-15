package library

import (
	"path/filepath"
	"testing"
)

func TestAlbumIndex(t *testing.T) {
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
		{Path: filepath.Join(dir, "a.mp3"), Title: "A", Album: "Kindred"},
		{Path: filepath.Join(dir, "b.mp3"), Title: "B", Album: "kindred"},
		{Path: filepath.Join(dir, "c.mp3"), Title: "C", Album: "FabricLive"},
		{Path: filepath.Join(dir, "d.mp3"), Title: "D"},
	} {
		if err := upsertTrack(tx, env, item, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	lists, err := Index(env, "album")
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 2 || lists[0].Name != "FabricLive" || lists[0].Count != 1 || lists[1].Name != "Kindred" || lists[1].Count != 2 {
		t.Fatalf("albums = %+v", lists)
	}
	page, err := Entity(env, "album", "Kindred", "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("kindred tracks = %d", page.Total)
	}
}
