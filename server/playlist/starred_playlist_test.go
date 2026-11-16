package playlist

import (
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestStarredArtistAndAlbumPlaylists(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	dbPath := filepath.Join(dir, "library.sqlite3")
	db, err := library.OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	insert := `INSERT INTO tracks(path, genre, parent_dir, title, artist, album, liked) VALUES(?,?,?,?,?,?,?)`
	parent := filepath.Join(music, "Electronic")
	rows := []struct {
		path, artist, album string
		liked               int
	}{
		{filepath.Join(parent, "a.mp3"), "Fabric", "Nexxsound", 1},
		{filepath.Join(parent, "b.mp3"), "Fabric", "Nexxsound", 0},
		{filepath.Join(parent, "c.mp3"), "Other", "Elsewhere", 1},
	}
	for _, row := range rows {
		if _, err := db.Exec(insert, row.path, "Electronic", parent, "t", row.artist, row.album, row.liked); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{
		Env: library.Env{
			MusicRoot: music,
			StateDir:  filepath.Join(dir, "state"),
			LibraryDB: dbPath,
		},
		PlaylistDir: filepath.Join(dir, "playlists"),
	}
	if _, err := library.ToggleArtistStar(env.Env, "Fabric"); err != nil {
		t.Fatal(err)
	}
	if _, err := library.ToggleAlbumStar(env.Env, "Nexxsound"); err != nil {
		t.Fatal(err)
	}
	items, err := ListIndex(env)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]IndexItem{}
	for _, item := range items {
		if item.Kind == "artist" || item.Kind == "album" {
			got[item.Kind+"/"+item.Name] = item
		}
	}
	artist, ok := got["artist/Fabric"]
	if !ok || artist.Count != 2 {
		t.Fatalf("artist playlist = %+v all=%+v", artist, got)
	}
	album, ok := got["album/Nexxsound"]
	if !ok || album.Count != 2 {
		t.Fatalf("album playlist = %+v", album)
	}
	page, err := TracksPageFor(env, "Fabric", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("fabric tracks = %d", page.Total)
	}
}
