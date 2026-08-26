package library_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/library"
)

func TestTracksForPathsWarmDBUnderThreshold(t *testing.T) {
	const n = 45
	const maxDuration = 100 * time.Millisecond

	cache := t.TempDir()
	root := t.TempDir()
	dbPath := filepath.Join(cache, "library.sqlite3")
	db, err := library.OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	paths := make([]string, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(root, fmt.Sprintf("t%d.mp3", i))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths[i] = p
		if _, err := db.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			p, "genre", filepath.Dir(p), fmt.Sprintf("t%d", i), "Artist", "Album", "2020", "", 180, "", "", 0,
		); err != nil {
			t.Fatal(err)
		}
	}
	env := library.Env{
		MusicRoot:      root,
		LibraryDB:      dbPath,
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	library.CloseCachedDB(env.LibraryDB)

	start := time.Now()
	got := library.TracksForPaths(env, paths, "current")
	elapsed := time.Since(start)
	if len(got) != n {
		t.Fatalf("len = %d, want %d", len(got), n)
	}
	if elapsed > maxDuration {
		t.Fatalf("TracksForPaths(%d) took %v, want <%v", n, elapsed, maxDuration)
	}
}
