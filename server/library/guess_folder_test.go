package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuessFolderNameUsesRawText(t *testing.T) {
	for _, in := range []string{"Deep Dubstep", "Jungle / DnB", "Pop", "Dance & EDM", "techno", "drum & bass"} {
		if got := GuessFolderName(in); got != "" {
			t.Fatalf("GuessFolderName(%q) = %q", in, got)
		}
	}
}

func TestMatchLibraryGenreIsExact(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Electronic", "House", "Drum & Bass"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{MusicRoot: root}
	if got := MatchLibraryGenre(env, "Dance & EDM"); got != "" {
		t.Fatalf("Dance & EDM filed into %q", got)
	}
	if got := MatchLibraryGenre(env, "techno"); got != "" {
		t.Fatalf("techno filed into %q", got)
	}
	if got := MatchLibraryGenre(env, "house"); got != "House" {
		t.Fatalf("house = %q", got)
	}
	if got := MatchLibraryGenre(env, "Drum & Bass"); got != "Drum & Bass" {
		t.Fatalf("drum & bass = %q", got)
	}
}
