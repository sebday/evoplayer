package warm

import (
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/waveform"
)

type Result struct {
	Path          string `json:"path"`
	Art           string `json:"art,omitempty"`
	Thumb         string `json:"thumb,omitempty"`
	Waveform      string `json:"waveform,omitempty"`
	ArtBuilt      bool   `json:"art_built"`
	WaveformBuilt bool   `json:"waveform_built"`
}

func Track(env paths.Env, trackPath string) (Result, error) {
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
		Path:     trackPath,
		Art:      row.Art,
		Thumb:    row.Thumb,
		Waveform: row.Waveform,
	}
	needArt := res.Art == "" || !assetExists(res.Art)
	needWaveform := res.Waveform == "" || !assetExists(res.Waveform)
	if needArt {
		if art, built, err := library.EnsureArt(libEnv, trackPath); err == nil && art != "" {
			res.Art = art
			res.ArtBuilt = built
		}
	}
	if needWaveform {
		out := waveformPath(libEnv, trackPath)
		if err := waveform.Build(trackPath, out); err == nil && assetExists(out) {
			res.Waveform = out
			res.WaveformBuilt = true
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

// TrackAssets warms album art and thumbs for list rows without building waveforms.
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

// WaveformForTrack builds only the waveform cache file when missing.
func WaveformForTrack(env paths.Env, trackPath string) (Result, error) {
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
		Path:     trackPath,
		Waveform: row.Waveform,
	}
	if res.Waveform != "" && assetExists(res.Waveform) {
		return res, nil
	}
	out := waveformPath(libEnv, trackPath)
	if err := waveform.Build(trackPath, out); err == nil && assetExists(out) {
		res.Waveform = out
		res.WaveformBuilt = true
	}
	return res, nil
}

func waveformPath(env library.Env, path string) string {
	key := library.CacheKey(env.MusicRoot, path)
	return filepath.Join(env.WaveformDir, key+".json")
}

func assetExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}
