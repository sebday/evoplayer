package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/library"
)

func TestBrowseRejectsEscape(t *testing.T) {
	root := t.TempDir()
	env := library.Env{MusicRoot: root}
	_, err := library.Browse(env, library.BrowseOptions{Rel: "../outside"})
	if err == nil {
		t.Fatal("expected error for escaped path")
	}
}

func TestMetaMissingFile(t *testing.T) {
	env := library.Env{MusicRoot: t.TempDir()}
	_, err := library.Meta(env, filepath.Join(t.TempDir(), "missing.mp3"), "")
	if err == nil {
		t.Fatal("expected error")
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
