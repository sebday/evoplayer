package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLikedTagPlaylists(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "library.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	env := Env{MusicRoot: filepath.Join(dir, "music")}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	house := Track{Path: filepath.Join(dir, "a.mp3"), Genre: "House", Title: "A", Liked: true, Tags: []string{"Space Jazz", "space jazz"}}
	dnb := Track{Path: filepath.Join(dir, "b.mp3"), Genre: "Drum & Bass", Title: "B", Liked: true, Tags: []string{"Space Jazz"}}
	plain := Track{Path: filepath.Join(dir, "c.mp3"), Genre: "House", Title: "C", Tags: []string{"Space Jazz"}}
	for _, item := range []Track{house, dnb, plain} {
		if err := upsertTrack(tx, env, item, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(env.MusicRoot, "House"), 0o755); err != nil {
		t.Fatal(err)
	}
	houseTag := Track{Path: filepath.Join(dir, "d.mp3"), Genre: "House", Title: "D", Liked: true, Tags: []string{"House"}}
	if err := func() error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := upsertTrack(tx, env, houseTag, 1, 1); err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}(); err != nil {
		t.Fatal(err)
	}
	lists, err := LikedTagPlaylists(db, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || lists[0].Name != "Space Jazz" || lists[0].Count != 2 {
		t.Fatalf("lists = %+v", lists)
	}
	paths, err := LikedPathsForTag(db, "space jazz")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
}
