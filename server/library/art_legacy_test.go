package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyArtFind(t *testing.T) {
	root := t.TempDir()
	art := t.TempDir()
	env := Env{MusicRoot: root, ArtDir: art}
	track := filepath.Join(root, "Drum & Bass", "Albums", "Classics", "105-dune.mp3")
	oldName := "drum&bass_albums_Classics_105-dune_mp3.jpg"
	if err := os.WriteFile(filepath.Join(art, oldName), []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := artCacheFind(env, track)
	if got == "" {
		t.Fatal("legacy cover was not found")
	}
	linked := filepath.Join(art, trackCacheSlug(relUnderRoot(root, track))+".jpg")
	if !nonEmptyFile(linked) {
		t.Fatalf("legacy cover was not linked to %s", linked)
	}
}

func TestGenreGroupKeepsFolderGenres(t *testing.T) {
	aliases := map[string]string{
		"garage":    "house",
		"house":     "house",
		"trap":      "hiphop",
		"techhouse": "house",
	}
	folders := map[string]string{
		"house":  "House",
		"garage": "Garage",
		"trap":   "Trap",
		"hiphop": "Hiphop",
	}
	if key, shown := genreGroup("House", aliases, folders); key != "dir:house" || shown != "House" {
		t.Fatalf("house: %s %s", key, shown)
	}
	if key, shown := genreGroup("Garage", aliases, folders); key != "dir:garage" || shown != "Garage" {
		t.Fatalf("garage: %s %s", key, shown)
	}
	if key, shown := genreGroup("Trap", aliases, folders); key != "dir:trap" || shown != "Trap" {
		t.Fatalf("trap: %s %s", key, shown)
	}
	if key, _ := genreGroup("techhouse", aliases, folders); key != "house" {
		t.Fatalf("techhouse should still fold into house, got %s", key)
	}
}
