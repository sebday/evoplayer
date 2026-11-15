package tags

import (
	"path/filepath"
	"testing"
)

func TestFilenameFallbackKeepsTaggedArtist(t *testing.T) {
	info := TagInfo{Artist: "Dillinja"}
	fillFilenameFallback(&info, "/music/02._dillinja-hands_in_the_air.mp3")
	if info.Artist != "Dillinja" || info.Title != "hands in the air" {
		t.Fatalf("tagged artist was replaced: %#v", info)
	}
	blank := TagInfo{}
	fillFilenameFallback(&blank, "/music/02._dillinja-hands_in_the_air.mp3")
	if blank.Artist != "dillinja" || blank.Title != "hands in the air" {
		t.Fatalf("blank tag fallback: %#v", blank)
	}
}

func TestPathSegmentsIgnoreCase(t *testing.T) {
	root := t.TempDir()
	mix := filepath.Join(root, "Dubstep", "Mixes", "2014", "track.mp3")
	if y := resolvePathYear(root, mix); y != 2014 {
		t.Fatalf("resolvePathYear() = %d, want 2014", y)
	}
	vinyl := filepath.Join(root, "Dubstep", "Vinyl", "dmz", "dmz001-release", "track.mp3")
	got := vinylFromPath(root, vinyl)
	if got == nil || got["publisher"] != "Dmz" {
		t.Fatalf("vinylFromPath() = %#v, want publisher Dmz", got)
	}
	lower := filepath.Join(root, "dubstep", "vinyl", "dmz", "dmz001-release", "track.mp3")
	got = vinylFromPath(root, lower)
	if got == nil || got["publisher"] != "Dmz" {
		t.Fatalf("vinylFromPath() lowercase = %#v, want publisher Dmz", got)
	}
}
