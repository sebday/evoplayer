package status

import (
	"encoding/json"
	"os"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

func enrichOnce(env paths.Env, st playback.Status) playback.Status {
	if st.Path == "" {
		return st
	}
	meta, err := library.Meta(library.EnvFrom(env), st.Path, "")
	if err != nil {
		return st
	}
	st.Title = pick(meta.Title, st.Title)
	st.Artist = meta.Artist
	st.Genre = meta.Genre
	st.Album = meta.Album
	st.Year = meta.Year
	st.Label = meta.Label
	st.Art = meta.Art
	st.Waveform = meta.Waveform
	st.Liked = meta.Liked
	if meta.Duration > 0 && st.Duration <= 0 {
		st.Duration = meta.Duration
		st = st.WithLabels()
	}
	return st
}

func Saved(env paths.Env) playback.Status {
	b, err := os.ReadFile(env.PlayerState)
	if err != nil {
		return playback.StoppedStatus()
	}
	var saved struct {
		Path     string  `json:"path"`
		Genre    string  `json:"genre"`
		Position float64 `json:"position"`
		Title    string  `json:"title"`
		Artist   string  `json:"artist"`
		Volume   *int    `json:"volume"`
	}
	if json.Unmarshal(b, &saved) != nil || saved.Path == "" {
		return playback.StoppedStatus()
	}
	st := playback.Status{
		State:    "stopped",
		Path:     saved.Path,
		Genre:    saved.Genre,
		Position: saved.Position,
		Title:    saved.Title,
		Artist:   saved.Artist,
	}
	if saved.Volume != nil {
		st.Volume = clampVolume(*saved.Volume)
	}
	return EnrichFull(env, st.WithLabels())
}

func pick(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
