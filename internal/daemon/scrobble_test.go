package daemon

import (
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/playback"
)

// Manual verification after deploy:
// 1. evoplayer stop && evoplayer start (daemon loads pass creds)
// 2. Play until min(half track, 4 min), skip — one submit in scrobble.jsonl / Last.fm
// 3. Rapid next/prev — one nowplaying per track, no duplicate submits
// 4. Seek backward >5s — listen window resets; no nowplaying spam

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

func TestScrobbleDedupeKey(t *testing.T) {
	st := playback.Status{
		Path:   "/music/track.flac",
		Artist: "Artist",
		Title:  "Title",
	}
	nowPlaying := scrobbleDedupeKey("track.updateNowPlaying", st, 0)
	wantNP := "track.updateNowPlaying|/music/track.flac|Artist|Title"
	if nowPlaying != wantNP {
		t.Fatalf("nowplaying key = %q want %q", nowPlaying, wantNP)
	}
	submit := scrobbleDedupeKey("track.scrobble", st, 1700000000)
	if submit != "track.scrobble|/music/track.flac|1700000000" {
		t.Fatalf("submit key = %q", submit)
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

func TestScrobbleDuplicateDifferentKeys(t *testing.T) {
	d := &Daemon{}
	st := playback.Status{Path: "/a.flac", Artist: "A", Title: "T"}

	k1 := scrobbleDedupeKey("track.updateNowPlaying", st, 0)
	if d.scrobbleDuplicate(k1) {
		t.Fatal("first nowplaying should not duplicate")
	}

	k2 := scrobbleDedupeKey("track.scrobble", st, 100)
	if d.scrobbleDuplicate(k2) {
		t.Fatal("submit with different key should not duplicate")
	}
}
