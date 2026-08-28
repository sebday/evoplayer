package playback

import (
	"os/exec"
	"strconv"
	"strings"
)

// DurationForPath returns track length in seconds via ffprobe, or 0 if unknown.
func DurationForPath(path string) float64 {
	if path == "" {
		return 0
	}
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration",
		"-of", "default=nw=1:nk=1", path)
	raw, err := cmd.Output()
	if err != nil {
		return 0
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil || sec <= 0 {
		return 0
	}
	return sec
}

func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}
