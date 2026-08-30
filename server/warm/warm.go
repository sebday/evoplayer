package warm

import (
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/waveform"
)

type Result struct {
	Path     string `json:"path"`
	Art      string `json:"art,omitempty"`
	Thumb    string `json:"thumb,omitempty"`
	Waveform string `json:"waveform,omitempty"`
	ArtBuilt bool   `json:"art_built"`
}

func Track(env paths.Env, trackPath string) (Result, error) {
	return TrackAssets(env, trackPath)
}

// TrackAssets warms album art and thumbs for list rows.
func TrackAssets(env paths.Env, trackPath string) (Result, error) {
	if trackPath == "" {
		return Result{}, os.ErrNotExist
	}
	if st, err := os.Stat(trackPath); err != nil || st.IsDir() {
		return Result{}, os.ErrNotExist
	}
	libEnv := library.EnvFrom(env)
	row, err := library.Meta(libEnv, trackPath, "")
	if err != nil {
		return Result{}, err
	}
	res := Result{
		Path:  trackPath,
		Art:   row.Art,
		Thumb: row.Thumb,
	}
	needArt := res.Art == "" || !assetExists(res.Art)
	if needArt {
		if art, built, err := library.EnsureArt(libEnv, trackPath); err == nil && art != "" {
			res.Art = art
			res.ArtBuilt = built
		}
	}
	needThumb := res.Art != "" && (res.Thumb == "" || !assetExists(res.Thumb))
	if needThumb {
		if thumb, err := library.EnsureThumb(libEnv, res.Art); err == nil {
			res.Thumb = thumb
		}
	}
	return res, nil
}

func WaveformForTrack(env paths.Env, trackPath string) (string, error) {
	if trackPath == "" {
		return "", os.ErrNotExist
	}
	if st, err := os.Stat(trackPath); err != nil || st.IsDir() {
		return "", os.ErrNotExist
	}
	out := filepath.Join(env.WaveformDir, library.CacheKey(env.MusicRoot, trackPath)+".json")
	if waveform.CacheFresh(out) {
		return out, nil
	}
	if err := os.MkdirAll(env.WaveformDir, 0o755); err != nil {
		return "", err
	}
	if err := waveform.Build(trackPath, out); err != nil {
		return "", err
	}
	return out, nil
}

func assetExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}
