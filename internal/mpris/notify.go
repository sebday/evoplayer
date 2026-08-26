package mpris

import "github.com/sebday/evoplayer/internal/playback"

type playerNotify struct {
	status         string
	path           string
	title          string
	artist         string
	album          string
	art            string
	canPlayOrPause bool
}

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

func playerNotifyFrom(st playback.Status) playerNotify {
	return playerNotify{
		status:         playbackStatusOf(st.State),
		path:           st.Path,
		title:          st.Title,
		artist:         st.Artist,
		album:          st.Album,
		art:            st.Art,
		canPlayOrPause: st.Path != "",
	}
}

func playerNotifyDelta(prev, next playerNotify) (status, metadata, canPlayPause bool) {
	status = prev.status != next.status
	metadata = prev.path != next.path ||
		prev.title != next.title ||
		prev.artist != next.artist ||
		prev.album != next.album ||
		prev.art != next.art
	canPlayPause = prev.canPlayOrPause != next.canPlayOrPause
	return
}
