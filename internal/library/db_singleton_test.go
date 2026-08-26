package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/library"
)

func TestEnsureDBSingleton(t *testing.T) {
	cache := t.TempDir()
	env := library.Env{
		MusicRoot:      t.TempDir(),
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	library.CloseCachedDB(env.LibraryDB)

	d1, err := library.EnsureDB(env)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := library.EnsureDB(env)
	if err != nil {
		t.Fatal(err)
	}
	if d1 != d2 {
		t.Fatal("expected cached db handle")
	}
}

func TestMetaManyCallsSameDB(t *testing.T) {
	cache := t.TempDir()
	root := t.TempDir()
	path := filepath.Join(root, "track.mp3")
	if err := os.WriteFile(path, []byte{0xFF, 0xFB}, 0o644); err != nil {
		t.Fatal(err)
	}
	env := library.Env{
		MusicRoot:      root,
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	library.CloseCachedDB(env.LibraryDB)

	for i := 0; i < 500; i++ {
		if _, err := library.Meta(env, path, ""); err != nil {
			t.Fatalf("meta %d: %v", i, err)
		}
	}
}
