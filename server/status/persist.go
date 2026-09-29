package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

var writeMu sync.Mutex

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
	if err := json.Unmarshal(b, &out); err != nil {
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
	dir := filepath.Dir(env.PlayerState)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".player-*.json")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmp.Name(), 0o644)
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), env.PlayerState)
	}
	if werr != nil {
		_ = os.Remove(tmp.Name())
	}
	return werr
}

// updatePlayerState serializes read-modify-write of player.json; unreadable state is replaced.
func updatePlayerState(env paths.Env, fn func(*playerStatePayload)) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	payload, _ := readPlayerState(env)
	fn(&payload)
	return writePlayerState(env, payload)
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
	return updatePlayerState(env, func(payload *playerStatePayload) {
		v := clampVolume(volume)
		payload.Volume = &v
	})
}

// Write saves the current track so a later daemon start can restore it paused.
func Write(env paths.Env, st playback.Status) error {
	if env.PlayerState == "" || st.Path == "" {
		return nil
	}
	return updatePlayerState(env, func(payload *playerStatePayload) {
		payload.Path = st.Path
		payload.Genre = st.Genre
		payload.Playlist = st.Playlist
		payload.Position = st.Position
		payload.Title = st.Title
		payload.Artist = st.Artist
		v := clampVolume(st.Volume)
		payload.Volume = &v
	})
}
