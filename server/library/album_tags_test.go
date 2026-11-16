package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/tags"
)

func TestUpdateAlbumTagsWritesChosenFields(t *testing.T) {
	dir := t.TempDir()
	env := Env{
		MusicRoot: dir,
		LibraryDB: filepath.Join(dir, "library.sqlite3"),
		CacheDir:  dir,
		ArtDir:    filepath.Join(dir, "art"),
		StateDir:  filepath.Join(dir, "state"),
	}
	a := writeAlbumFixture(t, dir, "a.mp3")
	b := writeAlbumFixture(t, dir, "b.mp3")
	album := "Mdz02"
	genre := "Drum & Bass"
	label := "Metalheadz"
	n, err := UpdateAlbumTags(env, []string{a, b}, AlbumTagPatch{
		Album: &album,
		Genre: &genre,
		Label: &label,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("updated %d", n)
	}
	for _, path := range []string{a, b} {
		got, err := tags.ReadTags(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Title != "Keep" || got.Album != "Mdz02" || got.Year != "2005" || got.Genre != "Drum & Bass" || got.Label != "Metalheadz" {
			t.Fatalf("%s title=%q album=%q year=%q genre=%q label=%q", path, got.Title, got.Album, got.Year, got.Genre, got.Label)
		}
		if tags.FormatTagList(got.Tags) != "bass" {
			t.Fatalf("%s tags %q", path, tags.FormatTagList(got.Tags))
		}
	}
	cleared := ""
	if _, err := UpdateAlbumTags(env, []string{a}, AlbumTagPatch{Tags: &cleared}); err != nil {
		t.Fatal(err)
	}
	got, err := tags.ReadTags(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tags) != 0 || got.Album != "Mdz02" {
		t.Fatalf("cleared tags %#v album %q", got.Tags, got.Album)
	}
}

func writeAlbumFixture(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, make([]byte, 32), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tags.Write(path, map[string]string{
		"title":     "Keep",
		"album":     "Old",
		"year":      "2005",
		"genre":     "Jungle",
		"publisher": "Other",
		"tags":      "bass",
	}); err != nil {
		t.Fatal(err)
	}
	return path
}
