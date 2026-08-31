package soundcloud

import (
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/syncarchive"
)

func TestLikesURL(t *testing.T) {
	if got := LikesURL("seb-day"); got != "https://soundcloud.com/seb-day/likes" {
		t.Fatalf("LikesURL = %q", got)
	}
	if got := LikesURL(""); got != "https://soundcloud.com/seb-day/likes" {
		t.Fatalf("LikesURL empty = %q", got)
	}
}

func TestArchivePath(t *testing.T) {
	got := syncarchive.Path("/state/player")
	want := filepath.Join("/state/player", "sync-archive.txt")
	if got != want {
		t.Fatalf("ArchivePath = %q, want %q", got, want)
	}
}
