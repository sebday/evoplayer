package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/library"
)

func testGenreEnv(t *testing.T) library.Env {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"drum&bass", "dubstep", "grime", "hiphop", "house", "electronic"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return library.Env{MusicRoot: root}
}

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

func TestMatchLibraryGenreAliases(t *testing.T) {
	env := testGenreEnv(t)
	tests := []struct {
		tag  string
		want string
	}{
		{"Jungle", "drum&bass"},
		{"Hip-hop & Rap", "hiphop"},
		{"140", "dubstep"},
		{"DnB", "drum&bass"},
		{"Electronic", "electronic"},
		{"Innamind", ""},
		{"Drum & Bass", "drum&bass"},
		{"Rap/Hip Hop", "hiphop"},
		{"PHONK", "hiphop"},
		{"Dance & EDM", "electronic"},
		{"UK Garage", ""},
	}
	for _, tc := range tests {
		got := library.MatchLibraryGenre(env, tc.tag)
		if got != tc.want {
			t.Errorf("MatchLibraryGenre(%q) = %q, want %q", tc.tag, got, tc.want)
		}
	}
}

func TestResolveCanonicalGenre(t *testing.T) {
	env := library.Env{}
	if got := library.ResolveCanonicalGenre(env, "Jungle"); got != config.GenreDrumAndBass {
		t.Fatalf("ResolveCanonicalGenre(Jungle) = %q", got)
	}
	if got := library.ResolveCanonicalGenre(env, "nope"); got != "nope" {
		t.Fatalf("ResolveCanonicalGenre(nope) = %q, want nope", got)
	}
}

func TestNormalizeGenreKey(t *testing.T) {
	if got := library.NormalizeGenreKey("Drum & Bass"); got != "drumandbass" {
		t.Fatalf("NormalizeGenreKey = %q", got)
	}
}
