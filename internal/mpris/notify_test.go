package mpris

import (
	"testing"

	"github.com/sebday/evoplayer/internal/playback"
)

func TestPlayerNotifyDeltaIgnoresPosition(t *testing.T) {
	prev := playback.Status{
		State:    "playing",
		Path:     "/music/a.mp3",
		Title:    "A",
		Artist:   "Artist",
		Album:    "Album",
		Art:      "/art/a.jpg",
		Position: 12,
		Volume:   40,
	}
	next := prev
	next.Position = 13
	next.Volume = 41
	next.Duration = 200
	status, metadata, can := playerNotifyDelta(playerNotifyFrom(prev), playerNotifyFrom(next))
	if status || metadata || can {
		t.Fatalf("position/volume-only update emitted status=%v metadata=%v can=%v", status, metadata, can)
	}
}

func TestPlayerNotifyDeltaPlaybackStatus(t *testing.T) {
	prev := playback.Status{State: "playing", Path: "/music/a.mp3"}
	next := prev
	next.State = "paused"
	status, metadata, can := playerNotifyDelta(playerNotifyFrom(prev), playerNotifyFrom(next))
	if !status || metadata || can {
		t.Fatalf("pause should only change status, got status=%v metadata=%v can=%v", status, metadata, can)
	}
}

func TestPlayerNotifyDeltaMetadataAndCanPlay(t *testing.T) {
	prev := playback.Status{State: "stopped"}
	next := playback.Status{
		State:  "playing",
		Path:   "/music/a.mp3",
		Title:  "A",
		Artist: "Artist",
		Art:    "/art/a.jpg",
	}
	status, metadata, can := playerNotifyDelta(playerNotifyFrom(prev), playerNotifyFrom(next))
	if !status || !metadata || !can {
		t.Fatalf("new track should change status, metadata, and canPlay, got status=%v metadata=%v can=%v", status, metadata, can)
	}
}

func TestPlayerNotifyDeltaTitleOnly(t *testing.T) {
	prev := playback.Status{State: "playing", Path: "/music/a.mp3", Title: "A"}
	next := prev
	next.Title = "B"
	status, metadata, can := playerNotifyDelta(playerNotifyFrom(prev), playerNotifyFrom(next))
	if status || !metadata || can {
		t.Fatalf("title change should only update metadata, got status=%v metadata=%v can=%v", status, metadata, can)
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
