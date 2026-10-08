package daemon

import (
	"testing"

	"github.com/sebday/evoplayer/server/playback"
)

func TestScrobblePositionJumped(t *testing.T) {
	if scrobblePositionJumped(10, 10.2, 0.25) {
		t.Fatal("normal playback is not a seek")
	}
	if scrobblePositionJumped(0, 3, 1) {
		t.Fatal("a few seconds of playback is not a seek")
	}
	if !scrobblePositionJumped(0, 800, 1) {
		t.Fatal("resume that jumps from 0 should reset the listen")
	}
	if !scrobblePositionJumped(400, 10, 1) {
		t.Fatal("backward seek should reset the listen")
	}
	if scrobblePositionJumped(100, 112, 10) {
		t.Fatal("position that keeps up with a late sample is not a seek")
	}
	if scrobblePositionJumped(-1, 50, 1) {
		t.Fatal("missing previous position is not a seek")
	}
}

func TestScrobbleSubmitDueIgnoresUnheardGap(t *testing.T) {
	st := playback.Status{State: "playing", Path: "/mix.mp3", Position: 800, Duration: 3600}
	due, _ := scrobbleSubmitDue("/mix.mp3", 800, 0, st)
	if due {
		t.Fatal("a session that starts at the resume point has not been heard yet")
	}
	st.Position = 800 + 241
	due, _ = scrobbleSubmitDue("/mix.mp3", 800, 0, st)
	if !due {
		t.Fatal("four minutes after the resume point should scrobble")
	}
}
