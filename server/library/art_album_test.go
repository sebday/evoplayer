package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathsSharingAlbum(t *testing.T) {
	files := []string{"/dir/a.mp3", "/dir/b.mp3", "/dir/c.mp3"}
	albums := map[string]string{
		"/dir/a.mp3": "JBLP25",
		"/dir/b.mp3": "jblp25",
		"/dir/c.mp3": "OTHER",
	}
	got := pathsSharingAlbum("/dir/a.mp3", "JBLP25", files, func(p string) string { return albums[p] })
	if len(got) != 2 || got[0] != "/dir/a.mp3" || got[1] != "/dir/b.mp3" {
		t.Fatalf("shared album: %#v", got)
	}
	if !coversPaths(files, files) {
		t.Fatal("full directory should count as one album folder")
	}
	if coversPaths(got, files) {
		t.Fatal("partial album must not publish a folder cover")
	}
}

func TestPathsSharingAlbumEmptyTag(t *testing.T) {
	files := []string{"/mixes/a.mp3", "/mixes/b.mp3"}
	got := pathsSharingAlbum("/mixes/a.mp3", "  ", files, func(string) string { return "" })
	if len(got) != 1 || got[0] != "/mixes/a.mp3" {
		t.Fatalf("empty album tag should stay on the one track, got %#v", got)
	}
}

func TestDirectoryIsAlbum(t *testing.T) {
	root := t.TempDir()
	write := func(dir string, names ...string) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	sc := filepath.Join(root, "dubstep", "soundcloud")
	write(sc, "a.mp3", "b.mp3")
	if directoryIsAlbum(sc, func(string) string { return "Bronca EP" }) {
		t.Fatal("a soundcloud folder is not an album")
	}
	album := filepath.Join(root, "dubstep", "JBLP25")
	write(album, "a.mp3", "b.mp3")
	if !directoryIsAlbum(album, func(string) string { return "JBLP25" }) {
		t.Fatal("a shared album tag should cover the folder")
	}
	mixes := filepath.Join(root, "dubstep", "mixes")
	write(mixes, "a.mp3", "b.mp3")
	if directoryIsAlbum(mixes, func(string) string { return "  " }) {
		t.Fatal("an empty album tag should not cover the folder")
	}
	if directoryIsAlbum(mixes, func(p string) string {
		if strings.HasSuffix(p, "a.mp3") {
			return "One"
		}
		return "Two"
	}) {
		t.Fatal("mixed album tags should not cover the folder")
	}
}

func TestSharedDumpCovers(t *testing.T) {
	root := t.TempDir()
	cache := t.TempDir()
	dir := filepath.Join(root, "dubstep", "soundcloud")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: root, ArtDir: cache}
	paths := []string{
		filepath.Join(dir, "a.mp3"),
		filepath.Join(dir, "b.mp3"),
		filepath.Join(dir, "c.mp3"),
	}
	for _, p := range paths {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	shared := filepath.Join(cache, "shared.jpg")
	if err := os.WriteFile(shared, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(shared, artPathTrack(env, paths[0])); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(shared, artPathTrack(env, paths[1])); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artPathTrack(env, paths[2]), []byte("solo"), 0o644); err != nil {
		t.Fatal(err)
	}
	folder := artPathFolder(env, paths[0])
	if err := os.WriteFile(folder, []byte("folder"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirs, tracks, err := SharedDumpCovers(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 1 || dirs[0] != dir {
		t.Fatalf("dirs: %#v", dirs)
	}
	if len(tracks) != 2 || tracks[0] != paths[0] || tracks[1] != paths[1] {
		t.Fatalf("shared tracks: %#v", tracks)
	}
	ClearDumpFolderCover(env, dir)
	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		t.Fatalf("folder cover should be removed, err=%v", err)
	}
}
