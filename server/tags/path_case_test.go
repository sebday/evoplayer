package tags

import (
	"path/filepath"
	"testing"
)

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
