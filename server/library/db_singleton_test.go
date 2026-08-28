package library_test

import (
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
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
