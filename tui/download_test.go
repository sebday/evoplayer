package tui

import "testing"

func TestLastJobLogLine(t *testing.T) {
	got := lastJobLogLine("· first\n✓ artist - track.mp3\n")
	if got != "artist - track.mp3" {
		t.Fatalf("got %q", got)
	}
}
