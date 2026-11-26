package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/tags"
)

func TestFolderForGenreTagKeepsJungleAndLiquidInDrumAndBass(t *testing.T) {
	env := aliasEnv(t)
	if got := FolderForGenreTag(env, "Jungle"); got != "Drum & Bass" {
		t.Fatalf("Jungle folder = %q", got)
	}
	if got := FolderForGenreTag(env, "Liquid"); got != "Drum & Bass" {
		t.Fatalf("Liquid folder = %q", got)
	}
	if got := FolderForGenreTag(env, "jungle"); got != "Drum & Bass" {
		t.Fatalf("jungle folder = %q", got)
	}
	if got := MatchLibraryGenre(env, "Jungle"); got != "" {
		t.Fatalf("exact match filed Jungle into %q", got)
	}
	if got := FolderForGenreTag(env, "Techno"); got != "" {
		t.Fatalf("Techno folder = %q", got)
	}
	if got := FolderForGenreTag(env, "House"); got != "House" {
		t.Fatalf("House folder = %q", got)
	}
}

func TestIncomingDestFilesAliasWithoutRewritingTag(t *testing.T) {
	env := aliasEnv(t)
	src := filepath.Join(env.MusicRoot, ".evoplayer", "incoming", "definite party material - windswept.mp3")
	probed := tags.ProbeResult{Tag: tags.TagInfo{
		Artist: "definite party material",
		Title:  "Windswept",
		Genre:  "Jungle",
	}, Duration: 330}
	dest, err := incomingDest(env, src, probed)
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(env.MusicRoot, "Drum & Bass", "Soundcloud")
	if filepath.Dir(dest) != wantDir {
		t.Fatalf("dest dir = %q, want %q", filepath.Dir(dest), wantDir)
	}
	if probed.Tag.Genre != "Jungle" {
		t.Fatalf("genre tag = %q", probed.Tag.Genre)
	}
}

func aliasEnv(t *testing.T) Env {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"Drum & Bass", "House"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := filepath.Join(t.TempDir(), "music.toml")
	body := strings.TrimSpace(`
[genres]
drumandbass = "Drum & Bass"
house = "House"

[genre_aliases]
jungle = "drumandbass"
liquid = "drumandbass"
`) + "\n"
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return Env{MusicRoot: root, MusicConfig: cfg}
}
