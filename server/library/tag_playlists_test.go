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
