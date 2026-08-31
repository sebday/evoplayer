package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestMatchLibraryGenreNormalizes(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"drum&bass", "dubstep", "grime"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := library.Env{MusicRoot: root}
	if got := library.MatchLibraryGenre(env, "Drum & Bass"); got != "drum&bass" {
		t.Fatalf("MatchLibraryGenre = %q, want drum&bass", got)
	}
	if got := library.MatchLibraryGenre(env, "DUBSTEP"); got != "dubstep" {
		t.Fatalf("MatchLibraryGenre dubstep = %q", got)
	}
	if got := library.MatchLibraryGenre(env, "Electronic"); got != "" {
		t.Fatalf("MatchLibraryGenre unknown = %q, want empty", got)
	}
}

func TestNormalizeGenreKey(t *testing.T) {
	if got := library.NormalizeGenreKey("Drum & Bass"); got != "drumandbass" {
		t.Fatalf("NormalizeGenreKey = %q", got)
	}
}
