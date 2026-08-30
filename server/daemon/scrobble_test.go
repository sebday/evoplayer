package daemon

import (
	"testing"
	"time"
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
