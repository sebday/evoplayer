package soundcloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
)

func TestParseTagList(t *testing.T) {
	got := parseTagList(`dubstep bass "drum & bass" 140bpm`)
	want := []string{"dubstep", "bass", "drum & bass", "140bpm"}
	if len(got) != len(want) {
		t.Fatalf("parseTagList = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseTagList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTrackEmbedGenreMatchesLibraryFolder(t *testing.T) {
	root := t.TempDir()
	opts := DownloadOptions{MusicRoot: root}
	if got := trackEmbedGenre(&Track{Genre: "Dubstep"}, opts); got != "Dubstep" {
		t.Fatalf("raw genre = %q, want Dubstep", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "drum&bass"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := trackEmbedGenre(&Track{Genre: "Drum & Bass"}, opts); got != "drum&bass" {
		t.Fatalf("matched folder = %q, want drum&bass", got)
	}
}

func TestTrackEmbedGenreDoesNotUseDefault(t *testing.T) {
	opts := DownloadOptions{
		MusicRoot:   t.TempDir(),
		MusicConfig: paths.Env{}.MusicConfig,
	}
	track := &Track{Title: "x", User: struct{ Username string `json:"username"` }{Username: "y"}}
	if got := trackEmbedGenre(track, opts); got != "" {
		t.Fatalf("expected no default genre, got %q", got)
	}
}
