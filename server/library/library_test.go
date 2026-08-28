package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestBrowseRejectsEscape(t *testing.T) {
	root := t.TempDir()
	env := library.Env{MusicRoot: root}
	_, err := library.Browse(env, library.BrowseOptions{Rel: "../outside"})
	if err == nil {
		t.Fatal("expected error for escaped path")
	}
}

func TestOpenDBCreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.sqlite3")
	db, err := library.OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
