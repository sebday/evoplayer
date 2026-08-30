package status

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

// Write saves the current track so a later daemon start can restore it paused.
func Write(env paths.Env, st playback.Status) error {
	if st.Path == "" || env.PlayerState == "" {
		return nil
	}
	payload := map[string]interface{}{
		"path":     st.Path,
		"genre":    st.Genre,
		"playlist": st.Playlist,
		"position": st.Position,
		"title":    st.Title,
		"artist":   st.Artist,
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
