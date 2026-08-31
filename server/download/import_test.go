package download_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/download"
	"github.com/sebday/evoplayer/server/paths"
)

func TestImportLibraryIncomingLeavesUntaggedInIncoming(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	incoming := filepath.Join(root, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(incoming, "Artist - Title.mp3")
	if err := os.WriteFile(src, []byte("not-really-mp3"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := paths.Env{
		MusicRoot: root,
		StateDir:  state,
		LibraryDB: filepath.Join(state, "library.sqlite3"),
	}
	if err := download.ImportLibraryIncoming(context.Background(), env, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal("expected untagged file to remain in .incoming")
	}
}
