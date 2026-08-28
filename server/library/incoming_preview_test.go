package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestPreviewIncomingNeedsGenre(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".incoming", "untagged.mp3")
	writeTinyMP3(t, path, map[string]string{"title": "Set", "artist": "DJ"})
	got := library.PreviewIncoming(testCacheEnv(t, root), path)
	if got["needs_genre"] != true {
		t.Fatalf("preview = %#v, want needs_genre", got)
	}
}

func TestListIncomingSkipsSidecars(t *testing.T) {
	root := t.TempDir()
	incoming := filepath.Join(root, ".incoming")
	track := filepath.Join(incoming, "keep.mp3")
	writeTinyMP3(t, track, map[string]string{"title": "Keep", "artist": "A", "genre": "misc"})
	if err := os.WriteFile(filepath.Join(incoming, "keep.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := library.ListIncoming(testCacheEnv(t, root))
	if len(got) != 1 || got[0]["path"] != track {
		t.Fatalf("list = %#v", got)
	}
}
