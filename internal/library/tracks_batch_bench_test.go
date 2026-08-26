package library_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/library"
)

func TestTracksForPathsUsesStoredGenre(t *testing.T) {
	cache := t.TempDir()
	root := t.TempDir()
	dbPath := filepath.Join(cache, "library.sqlite3")
	db, err := library.OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	path := filepath.Join(root, "a.mp3")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		path, "drum&bass", filepath.Dir(path), "Title", "Artist", "Album", "2020", "", 180, "", "", 0,
	); err != nil {
		t.Fatal(err)
	}
	env := library.Env{
		MusicRoot:      root,
		LibraryDB:      dbPath,
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	library.CloseCachedDB(env.LibraryDB)
	got := library.TracksForPaths(env, []string{path}, "")
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Genre != "drum&bass" {
		t.Fatalf("genre = %q", got[0].Genre)
	}
}

func BenchmarkTracksForPaths45(b *testing.B) {
	benchmarkTracksForPaths(b, 45)
}

func benchmarkTracksForPaths(b *testing.B, n int) {
	cache := b.TempDir()
	root := b.TempDir()
	dbPath := filepath.Join(cache, "library.sqlite3")
	db, err := library.OpenDB(dbPath)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	paths := make([]string, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(root, fmt.Sprintf("t%d.mp3", i))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			b.Fatal(err)
		}
		paths[i] = p
		if _, err := db.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			p, "genre", filepath.Dir(p), fmt.Sprintf("t%d", i), "Artist", "Album", "2020", "", 180, "", "", 0,
		); err != nil {
			b.Fatal(err)
		}
	}
	env := library.Env{
		MusicRoot:      root,
		LibraryDB:      dbPath,
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	library.CloseCachedDB(env.LibraryDB)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = library.TracksForPaths(env, paths, "bench")
	}
}
