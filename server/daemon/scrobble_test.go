package daemon

import (
	"testing"
	"time"

	"github.com/sebday/evoplayer/server/playback"
)

func TestScrobbleListenThreshold(t *testing.T) {
	cases := []struct {
		dur, want float64
	}{
		{30, -1},
		{45, 22.5},
		{120, 60},
		{180, 90},
		{600, 240},
		{900, 240},
	}
	for _, tc := range cases {
		got := scrobbleListenThreshold(tc.dur)
		if got != tc.want {
			t.Fatalf("duration=%v threshold=%v want %v", tc.dur, got, tc.want)
		}
	}
}

func TestScrobbleSubmitDue(t *testing.T) {
	path := "/music/a.mp3"
	st := playback.Status{State: "playing", Path: path, Duration: 120, Position: 60}
	if due, _ := scrobbleSubmitDue(path, 0, 1000, st); !due {
		t.Fatal("60s of 120s track should be due")
	}
	st.Position = 30
	if due, _ := scrobbleSubmitDue(path, 0, 1000, st); due {
		t.Fatal("30s of 120s track should not be due")
	}
	st.Duration = 20
	if due, _ := scrobbleSubmitDue(path, 0, 1000, st); due {
		t.Fatal("short track should not be due")
	}
}

func TestAutoScrobbleBeginsNewSessionOnTrackChange(t *testing.T) {
	d := &Daemon{}
	prev := playback.Status{State: "playing", Path: "/a.mp3", Duration: 120, Position: 70, Artist: "A", Title: "One"}
	next := playback.Status{State: "playing", Path: "/b.mp3", Duration: 200, Position: 1, Artist: "B", Title: "Two"}

	d.scrobbleMu.Lock()
	d.scrobblePrev = prev
	d.beginScrobbleSessionLocked(prev)
	d.scrobbleMu.Unlock()

	// Last.fm not configured: autoScrobble only updates session bookkeeping.
	d.autoScrobble(next)

	d.scrobbleMu.Lock()
	defer d.scrobbleMu.Unlock()
	if d.scrobblePath != "/b.mp3" {
		t.Fatalf("scrobble path = %q, want /b.mp3", d.scrobblePath)
	}
	if d.scrobbleSubmitted {
		t.Fatal("new track session should not be marked submitted yet")
	}
	if d.scrobbleStartPos != 1 {
		t.Fatalf("start pos = %v, want 1", d.scrobbleStartPos)
	}
}

func TestScrobbleDuplicateWithinWindow(t *testing.T) {
	d := &Daemon{}
	key := "track.updateNowPlaying|/a|Artist|Title"

	if d.scrobbleDuplicate(key) {
		t.Fatal("first call should not be duplicate")
	}
	if !d.scrobbleDuplicate(key) {
		t.Fatal("second call within window should be duplicate")
	}

	d.scrobbleMu.Lock()
	d.scrobbleDedupeAt = time.Now().Add(-scrobbleDedupeWindow - time.Millisecond)
	d.scrobbleMu.Unlock()

	if d.scrobbleDuplicate(key) {
		t.Fatal("call after window should not be duplicate")
	}
}
