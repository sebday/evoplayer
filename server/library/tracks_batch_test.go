package library_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestTracksForPathsBatch(t *testing.T) {
	cache := t.TempDir()
	root := t.TempDir()
	paths := make([]string, 3)
	for i := range paths {
		p := filepath.Join(root, fmt.Sprintf("t%d.mp3", i))
		if err := os.WriteFile(p, []byte{0xFF, 0xFB}, 0o644); err != nil {
			t.Fatal(err)
		}
		paths[i] = p
	}
	env := library.Env{
		MusicRoot:      root,
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	library.CloseCachedDB(env.LibraryDB)

	got := library.TracksForPaths(env, paths, "current")
	if len(got) != len(paths) {
		t.Fatalf("len = %d, want %d", len(got), len(paths))
	}
	for i, row := range got {
		if row.Path != paths[i] {
			t.Fatalf("row %d path = %q, want %q", i, row.Path, paths[i])
		}
	}
}
