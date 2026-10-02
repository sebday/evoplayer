package library

import "testing"

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
