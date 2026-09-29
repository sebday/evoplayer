package mpris

import (
	"math"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/sebday/evoplayer/server/playback"
)

// seekedToleranceSec absorbs position tick jitter before reporting a Seeked jump.
const seekedToleranceSec = 1.0

func playbackStatusOf(state string) string {
	switch state {
	case "playing":
		return "Playing"
	case "paused":
		return "Paused"
	default:
		return "Stopped"
	}
}

func loopStatusOf(st playback.Status) string {
	if st.Repeat {
		return "Track"
	}
	return "Playlist"
}

func volumeOf(st playback.Status) float64 {
	return math.Min(math.Max(float64(st.Volume)/100, 0), 1)
}

func metadataChanged(prev, next playback.Status) bool {
	return prev.Path != next.Path ||
		prev.Title != next.Title ||
		prev.Artist != next.Artist ||
		prev.Album != next.Album ||
		prev.Art != next.Art ||
		int64(prev.Duration*1e6) != int64(next.Duration*1e6)
}

func playerChanges(prev, next playback.Status) map[string]dbus.Variant {
	changed := map[string]dbus.Variant{}
	if playbackStatusOf(prev.State) != playbackStatusOf(next.State) {
		changed["PlaybackStatus"] = dbus.MakeVariant(playbackStatusOf(next.State))
	}
	if metadataChanged(prev, next) {
		changed["Metadata"] = dbus.MakeVariant(metadataFrom(next))
	}
	if (prev.Path != "") != (next.Path != "") {
		can := next.Path != ""
		changed["CanPlay"] = dbus.MakeVariant(can)
		changed["CanPause"] = dbus.MakeVariant(can)
		changed["CanSeek"] = dbus.MakeVariant(can)
	}
	if prev.Shuffle != next.Shuffle {
		changed["Shuffle"] = dbus.MakeVariant(next.Shuffle)
	}
	if prev.Repeat != next.Repeat {
		changed["LoopStatus"] = dbus.MakeVariant(loopStatusOf(next))
	}
	if prev.Volume != next.Volume {
		changed["Volume"] = dbus.MakeVariant(volumeOf(next))
	}
	return changed
}

func seeked(prev, next playback.Status, elapsed time.Duration) bool {
	if prev.Path == "" || prev.Path != next.Path {
		return false
	}
	expected := prev.Position
	if prev.State == "playing" {
		expected += elapsed.Seconds()
	}
	return math.Abs(next.Position-expected) > seekedToleranceSec
}
