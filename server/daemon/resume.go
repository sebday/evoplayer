package daemon

import (
	"encoding/json"
	"os"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playlist"
)

func readSavedPlayback(env paths.Env) (path string, position float64, ok bool) {
	b, err := os.ReadFile(env.PlayerState)
	if err != nil {
		return "", 0, false
	}
	var saved struct {
		Path     string  `json:"path"`
		Position float64 `json:"position"`
	}
	if json.Unmarshal(b, &saved) != nil || saved.Path == "" {
		return "", 0, false
	}
	if _, err := os.Stat(saved.Path); err != nil {
		return "", 0, false
	}
	return saved.Path, saved.Position, true
}

func (d *Daemon) resumePlayback(play bool) error {
	st := d.Actor.Snapshot()
	if st.Path != "" {
		if play && st.State == "paused" {
			return d.Actor.Toggle()
		}
		if !play && st.State == "playing" {
			return d.Actor.Toggle()
		}
		return nil
	}

	savedPath, savedPos, ok := readSavedPlayback(d.Env)
	if !ok {
		return nil
	}

	paths, err := playlist.ReadCurrentPaths(playlist.EnvFrom(d.Env))
	if err != nil {
		paths = nil
	}
	if len(paths) == 0 {
		paths = []string{savedPath}
	} else if !pathInList(paths, savedPath) {
		paths = append([]string{savedPath}, paths...)
	}

	if err := d.Actor.ReplaceQueue(paths, savedPath); err != nil {
		return err
	}
	if savedPos > 0 {
		_ = d.Actor.Seek(savedPos)
	}
	st = d.Actor.Snapshot()
	if play && st.State == "paused" {
		return d.Actor.Toggle()
	}
	if !play && st.State == "playing" {
		return d.Actor.Toggle()
	}
	return nil
}

func pathInList(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
