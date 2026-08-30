package mpris

import (
	"testing"

	"github.com/sebday/evoplayer/server/playback"
)

func TestPlayerNotifyDeltaPlaybackStatus(t *testing.T) {
	prev := playback.Status{State: "playing", Path: "/music/a.mp3"}
	next := prev
	next.State = "paused"
	status, metadata, can := playerNotifyDelta(playerNotifyFrom(prev), playerNotifyFrom(next))
	if !status || metadata || can {
		t.Fatalf("pause should only change status, got status=%v metadata=%v can=%v", status, metadata, can)
	}
}

func TestPlaybackStatusOf(t *testing.T) {
	if got := playbackStatusOf("playing"); got != "Playing" {
		t.Fatalf("playing = %q", got)
	}
	if got := playbackStatusOf("paused"); got != "Paused" {
		t.Fatalf("paused = %q", got)
	}
	if got := playbackStatusOf("stopped"); got != "Stopped" {
		t.Fatalf("stopped = %q", got)
	}
	if got := playbackStatusOf(""); got != "Stopped" {
		t.Fatalf("empty = %q", got)
	}
}
