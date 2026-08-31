package status

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

type playerStatePayload struct {
	Path     string  `json:"path,omitempty"`
	Genre    string  `json:"genre,omitempty"`
	Playlist string  `json:"playlist,omitempty"`
	Position float64 `json:"position,omitempty"`
	Title    string  `json:"title,omitempty"`
	Artist   string  `json:"artist,omitempty"`
	Volume   *int    `json:"volume,omitempty"`
}

func readPlayerState(env paths.Env) (playerStatePayload, error) {
	out := playerStatePayload{}
	if env.PlayerState == "" {
		return out, os.ErrNotExist
	}
	b, err := os.ReadFile(env.PlayerState)
	if err != nil {
		return out, err
	}
	if json.Unmarshal(b, &out) != nil {
		return playerStatePayload{}, err
	}
	return out, nil
}

func writePlayerState(env paths.Env, payload playerStatePayload) error {
	if env.PlayerState == "" {
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(env.PlayerState), 0o755); err != nil {
		return err
	}
	return os.WriteFile(env.PlayerState, b, 0o644)
}

func clampVolume(vol int) int {
	if vol < 0 {
		return 0
	}
	if vol > 100 {
		return 100
	}
	return vol
}

// SavedVolume returns the last persisted output level, if any.
func SavedVolume(env paths.Env) (int, bool) {
	payload, err := readPlayerState(env)
	if err != nil || payload.Volume == nil {
		return 0, false
	}
	return clampVolume(*payload.Volume), true
}

// WriteVolume persists the output level without requiring a loaded track.
func WriteVolume(env paths.Env, volume int) error {
	payload, err := readPlayerState(env)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	v := clampVolume(volume)
	payload.Volume = &v
	return writePlayerState(env, payload)
}

// Write saves the current track so a later daemon start can restore it paused.
func Write(env paths.Env, st playback.Status) error {
	if env.PlayerState == "" {
		return nil
	}
	payload, err := readPlayerState(env)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if st.Path == "" {
		return nil
	}
	payload.Path = st.Path
	payload.Genre = st.Genre
	payload.Playlist = st.Playlist
	payload.Position = st.Position
	payload.Title = st.Title
	payload.Artist = st.Artist
	v := clampVolume(st.Volume)
	payload.Volume = &v
	return writePlayerState(env, payload)
}
