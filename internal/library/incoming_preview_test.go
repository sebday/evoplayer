package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/library"
)

func TestPreviewIncomingReadsTags(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".incoming", "track.mp3")
	writeTinyMP3(t, path, map[string]string{
		"title":  "Fabriclive",
		"artist": "DJ Hype",
		"genre":  "Drum & Bass",
		"year":   "2026",
	})
	env := testCacheEnv(t, root)
	got := library.PreviewIncoming(env, path)
	if got["title"] != "Fabriclive" || got["artist"] != "DJ Hype" {
		t.Fatalf("preview = %#v", got)
	}
	if got["path"] != path {
		t.Fatalf("path = %v", got["path"])
	}
	if got["genre"] != "Drum & Bass" {
		t.Fatalf("genre = %v", got["genre"])
	}
	if got["needs_genre"] == true {
		t.Fatal("known genre should not need a picker")
	}
}

func TestPreviewIncomingNeedsGenre(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".incoming", "untagged.mp3")
	writeTinyMP3(t, path, map[string]string{"title": "Set", "artist": "DJ"})
	got := library.PreviewIncoming(testCacheEnv(t, root), path)
	if got["needs_genre"] != true {
		t.Fatalf("preview = %#v, want needs_genre", got)
	}
}

func TestSetIncomingGenreAllowsImport(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".incoming", "untagged.mp3")
	writeTinyMP3(t, path, map[string]string{"title": "Set", "artist": "DJ"})
	env := testCacheEnv(t, root)
	got, err := library.SetIncomingGenre(env, path, "drum&bass")
	if err != nil {
		t.Fatal(err)
	}
	if got["needs_genre"] == true || got["genre"] != "drum&bass" || got["folder"] != "drum&bass" {
		t.Fatalf("after tag = %#v", got)
	}
	if err := library.RunImport(env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("tagged incoming file should be imported")
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
