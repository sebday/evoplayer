package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestMatchLibraryGenre(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"drum&bass", "dubstep", "grime", "hiphop", "house", "electronic"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := library.Env{MusicRoot: root}

	for _, tc := range []struct {
		tag  string
		want string
	}{
		{"Drum & Bass", "drum&bass"},
		{"Jungle", "drum&bass"},
		{"Hip-hop & Rap", "hiphop"},
		{"140", "dubstep"},
		{"DnB", "drum&bass"},
		{"Electronic", "electronic"},
		{"Innamind", ""},
		{"UK Garage", ""},
	} {
		if got := library.MatchLibraryGenre(env, tc.tag); got != tc.want {
			t.Errorf("MatchLibraryGenre(%q) = %q, want %q", tc.tag, got, tc.want)
		}
	}
}
