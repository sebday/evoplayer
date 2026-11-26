package soundcloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestExtraSoundCloudTags(t *testing.T) {
	track := &Track{
		Genre:   "Drum & Bass",
		TagList: `"Space Jazz" house "Drum & Bass"`,
	}
	got := extraSoundCloudTags(track, "Drum & Bass")
	if got != "Space Jazz; house" {
		t.Fatalf("got %q", got)
	}
	if extraSoundCloudTags(track, "House") != "Drum & Bass; Space Jazz" {
		t.Fatalf("folder skip failed: %q", extraSoundCloudTags(track, "House"))
	}
}

func TestTrackEmbedGenreKeepsJungle(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Drum & Bass"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(t.TempDir(), "music.toml")
	body := "[genres]\ndrumandbass = \"Drum & Bass\"\n\n[genre_aliases]\njungle = \"drumandbass\"\n"
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := DownloadOptions{MusicRoot: root, MusicConfig: cfg}
	got := trackEmbedGenre(&Track{Genre: "Jungle"}, opts)
	if got != "Jungle" {
		t.Fatalf("embed genre = %q", got)
	}
	env := library.Env{MusicRoot: root, MusicConfig: cfg}
	if folder := FolderFromLikedTrack(env, &Track{Genre: "Jungle"}); folder != "Drum & Bass" {
		t.Fatalf("folder = %q", folder)
	}
}
