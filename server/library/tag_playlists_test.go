package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLikedGenrePlaylists(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "library.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	env := Env{MusicRoot: filepath.Join(dir, "music")}
	if err := os.MkdirAll(filepath.Join(env.MusicRoot, "Drum & Bass"), 0o755); err != nil {
		t.Fatal(err)
	}
	liquid := Track{Path: filepath.Join(env.MusicRoot, "Drum & Bass", "liquid.mp3"), Genre: "Liquid", Title: "L", Liked: true, Tags: []string{"bass", "remix"}}
	jungle := Track{Path: filepath.Join(env.MusicRoot, "Drum & Bass", "jungle.mp3"), Genre: "Jungle", Title: "J", Liked: true}
	unliked := Track{Path: filepath.Join(env.MusicRoot, "Drum & Bass", "other.mp3"), Genre: "Liquid", Title: "U"}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []Track{liquid, jungle, unliked} {
		if err := upsertTrack(tx, env, item, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	genres, err := LikedGenrePlaylists(db)
	if err != nil {
		t.Fatal(err)
	}
	gotGenre := map[string]int{}
	for _, g := range genres {
		gotGenre[g.Name] = g.Count
	}
	if len(gotGenre) != 2 || gotGenre["Liquid"] != 1 || gotGenre["Jungle"] != 1 {
		t.Fatalf("genre playlists = %+v", genres)
	}
	liquidPaths, err := LikedPathsForGenreTag(db, "Liquid")
	if err != nil {
		t.Fatal(err)
	}
	if len(liquidPaths) != 1 || liquidPaths[0] != liquid.Path {
		t.Fatalf("liquid paths = %v", liquidPaths)
	}
}

func TestLikedTagPlaylists(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "library.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	env := Env{MusicRoot: filepath.Join(dir, "music")}
	if err := os.MkdirAll(filepath.Join(env.MusicRoot, "Drum & Bass"), 0o755); err != nil {
		t.Fatal(err)
	}
	liked := Track{Path: filepath.Join(env.MusicRoot, "Drum & Bass", "a.mp3"), Genre: "Liquid", Label: "31 Records", Title: "A", Liked: true, Tags: []string{"bass", "remix"}}
	other := Track{Path: filepath.Join(env.MusicRoot, "Drum & Bass", "b.mp3"), Genre: "Jungle", Label: "31 Records", Title: "B", Liked: true, Tags: []string{"bass"}}
	plain := Track{Path: filepath.Join(env.MusicRoot, "Drum & Bass", "c.mp3"), Genre: "Liquid", Label: "Ram", Title: "C", Tags: []string{"ram", "bass"}}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []Track{liked, other, plain} {
		if err := upsertTrack(tx, env, item, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	got, err := TagPlaylists(db)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, item := range got {
		counts[item.Name] = item.Count
	}
	if len(counts) != 3 || counts["bass"] != 3 || counts["remix"] != 1 || counts["ram"] != 1 {
		t.Fatalf("tag playlists = %+v", got)
	}
	if _, ok := counts["31 Records"]; ok {
		t.Fatalf("record label became a playlist: %+v", got)
	}
	paths, err := PathsForTag(db, "bass")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("paths = %v", paths)
	}
}

func TestSharedTagList(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "library.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	env := Env{MusicRoot: filepath.Join(dir, "music")}
	if err := os.MkdirAll(filepath.Join(env.MusicRoot, "Hiphop"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := Track{Path: filepath.Join(env.MusicRoot, "Hiphop", "a.mp3"), Artist: "Cypress Hill", Title: "A", Tags: []string{"bass", "remix"}}
	b := Track{Path: filepath.Join(env.MusicRoot, "Hiphop", "b.mp3"), Artist: "Cypress Hill", Title: "B", Tags: []string{"bass", "remix"}}
	c := Track{Path: filepath.Join(env.MusicRoot, "Hiphop", "c.mp3"), Artist: "Cypress Hill", Title: "C", Tags: []string{"club"}}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []Track{a, b, c} {
		if err := upsertTrack(tx, env, item, 1, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	same, uniform, err := SharedTagList(db, []string{a.Path, b.Path})
	if err != nil {
		t.Fatal(err)
	}
	if !uniform || same != "bass; remix" {
		t.Fatalf("shared = %q uniform=%v", same, uniform)
	}
	mixed, uniform, err := SharedTagList(db, []string{a.Path, c.Path})
	if err != nil {
		t.Fatal(err)
	}
	if uniform || mixed != "" {
		t.Fatalf("mixed = %q uniform=%v", mixed, uniform)
	}
}
