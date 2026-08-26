package playback

import (
	"testing"
)

func TestPlayPathInQueue(t *testing.T) {
	path := writeTestWAV(t, 48000, 2)
	other := writeTestWAV(t, 48000, 1)

	a := NewActor(nil)
	if err := a.ReplaceQueue([]string{other, path}, other); err != nil {
		t.Fatal(err)
	}

	if err := a.PlayPathInQueue(path); err != nil {
		t.Fatal(err)
	}
	st := a.Snapshot()
	if st.Path != path {
		t.Fatalf("path = %q, want %q", st.Path, path)
	}
	if st.State != "playing" {
		t.Fatalf("state = %q, want playing", st.State)
	}

	if err := a.PlayPathInQueue("/nonexistent/track.mp3"); err == nil {
		t.Fatal("expected error for missing path")
	}

	third := writeTestWAV(t, 48000, 1)
	expanded := []string{path, third, other}
	if err := a.SetQueue(expanded, path); err != nil {
		t.Fatal(err)
	}
	st = a.Snapshot()
	if st.Path != path {
		t.Fatalf("SetQueue should not change path, got %q", st.Path)
	}
	if st.PlaylistCount != len(expanded) {
		t.Fatalf("expected %d tracks, got %d", len(expanded), st.PlaylistCount)
	}
}

func TestReplaceQueueNoOpWhenUnchanged(t *testing.T) {
	path := writeTestWAV(t, 48000, 2)
	other := writeTestWAV(t, 48000, 1)
	paths := []string{other, path}

	a := NewActor(nil)
	if err := a.ReplaceQueue(paths, path); err != nil {
		t.Fatal(err)
	}
	gen := a.TrackGeneration()
	st := a.Snapshot()
	if st.Path != path || st.State != "playing" {
		t.Fatalf("expected playing %q, got path=%q state=%q", path, st.Path, st.State)
	}

	if err := a.ReplaceQueue(paths, path); err != nil {
		t.Fatal(err)
	}
	if a.TrackGeneration() != gen {
		t.Fatal("expected unchanged track generation on identical replace")
	}
}
