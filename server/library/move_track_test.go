package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/tags"
)

func TestMoveTrackToFolder(t *testing.T) {
	root := t.TempDir()
	genre := filepath.Join(root, "garage", "soundcloud")
	if err := os.MkdirAll(genre, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "electronic", "soundcloud", "artist-title.mp3")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: root, StateDir: t.TempDir(), CacheDir: t.TempDir()}
	res, err := MoveTrackToFolder(env, src, "garage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source should be gone: %v", err)
	}
	if _, err := os.Stat(res.To); err != nil {
		t.Fatalf("dest missing: %v", err)
	}
	if res.Folder != "garage" {
		t.Fatalf("folder=%q", res.Folder)
	}
}

func TestTrackDestForFolderMix(t *testing.T) {
	root := t.TempDir()
	env := Env{MusicRoot: root}
	path := filepath.Join(root, "a.mp3")
	probed := tags.ProbeResult{
		Duration: 3700,
		Tag: tags.TagInfo{
			Artist: "dj",
			Title:  "mix set",
			Year:   "2024",
		},
	}
	dest, err := trackDestForFolder(env, path, "drum&bass", probed)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "drum&bass", "mixes", "2024", "dj-mix_set.mp3")
	if dest != want {
		t.Fatalf("dest=%q want %q", dest, want)
	}
}
