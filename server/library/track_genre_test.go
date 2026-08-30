package library

import "testing"

func TestTagGenreForPathFallback(t *testing.T) {
	got := tagGenreForPath("/nonexistent/track.mp3", "dubstep")
	if got != "dubstep" {
		t.Fatalf("fallback = %q", got)
	}
}
