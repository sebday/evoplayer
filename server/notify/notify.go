package notify

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

const AppID = "evo.evoplayer"

// Match evo.monitors Service.qml low popup duration default.
const LowTimeoutMs = 3000

func Enabled() bool {
	if os.Getenv("EVOPLAYER_NOTIFY") == "0" {
		return false
	}
	_, err := exec.LookPath("omarchy")
	return err == nil
}

func send(icon, summary, body, urgency string, timeoutMs int) {
	if !Enabled() {
		return
	}
	if urgency == "" {
		urgency = "low"
	}
	if timeoutMs <= 0 {
		timeoutMs = LowTimeoutMs
	}
	args := []string{
		"notification", "send",
		"--app-name", AppID,
		"-u", urgency,
		"-t", strconv.Itoa(timeoutMs),
	}
	if icon != "" {
		if st, err := os.Stat(icon); err == nil && !st.IsDir() {
			args = append(args, "--image", icon)
		}
	}
	args = append(args, summary)
	if body != "" {
		args = append(args, body)
	}
	cmd := exec.Command("omarchy", args...)
	_ = cmd.Start()
}

func trackSummary(st playback.Status) string {
	if t := strings.TrimSpace(st.Title); t != "" {
		return t
	}
	if st.Path != "" {
		return strings.TrimSuffix(filepath.Base(st.Path), filepath.Ext(st.Path))
	}
	return "Unknown"
}

// trackDetail is the secondary line when summary is already the track title.
func trackDetail(st playback.Status) string {
	if artist := strings.TrimSpace(st.Artist); artist != "" {
		return artist
	}
	if album := strings.TrimSpace(st.Album); album != "" {
		return album
	}
	return ""
}

// trackLine is a single-line track label when summary is an action (Playing, Liked, …).
func trackLine(st playback.Status) string {
	title := trackSummary(st)
	if artist := strings.TrimSpace(st.Artist); artist != "" {
		return artist + " — " + title
	}
	return title
}

func iconPath(st playback.Status) string {
	art := strings.TrimSpace(st.Art)
	if art == "" {
		return ""
	}
	if st, err := os.Stat(art); err == nil && !st.IsDir() {
		return art
	}
	return ""
}

func NowPlaying(st playback.Status) {
	send(iconPath(st), trackSummary(st), trackDetail(st), "low", LowTimeoutMs)
}

func Transport(state string, st playback.Status) {
	summary := "Playing"
	if state == "paused" {
		summary = "Paused"
	}
	send(iconPath(st), summary, trackLine(st), "low", LowTimeoutMs)
}

func Favorite(env paths.Env, liked bool, path string) {
	libEnv := library.EnvFrom(env)
	track, err := library.Meta(libEnv, path, "")
	st := playback.Status{Path: path}
	if err == nil {
		st.Title = track.Title
		st.Artist = track.Artist
		st.Art = track.Art
	}
	summary := "Liked"
	if !liked {
		summary = "Unliked"
	}
	send(iconPath(st), summary, trackLine(st), "low", LowTimeoutMs)
}
