package playback

import "testing"

func TestUpNextPathsLinear(t *testing.T) {
	a := NewActor(nil)
	a.queue = []string{"/a.mp3", "/b.mp3", "/c.mp3", "/d.mp3"}
	a.index = 1
	got := a.UpNextPaths(3)
	if len(got) != 2 {
		t.Fatalf("expected 2 tracks after index 1, got %d: %v", len(got), got)
	}
	if got[0] != "/c.mp3" || got[1] != "/d.mp3" {
		t.Fatalf("unexpected order: %v", got)
	}
}

func TestUpNextPathsShuffle(t *testing.T) {
	a := NewActor(nil)
	a.queue = []string{"/a.mp3", "/b.mp3", "/c.mp3", "/d.mp3"}
	a.index = 2
	a.shuffle = true
	a.shuffleOrd = []int{2, 0, 3, 1}
	got := a.UpNextPaths(2)
	if len(got) != 2 {
		t.Fatalf("expected 2 tracks, got %d: %v", len(got), got)
	}
	if got[0] != "/a.mp3" || got[1] != "/d.mp3" {
		t.Fatalf("unexpected shuffle order: %v", got)
	}
}

func TestUpNextPathsAtEnd(t *testing.T) {
	a := NewActor(nil)
	a.queue = []string{"/a.mp3", "/b.mp3"}
	a.index = 1
	a.path = "/b.mp3"
	got := a.UpNextPaths(3)
	if got != nil {
		t.Fatalf("expected nil at end of queue, got %v", got)
	}
}
