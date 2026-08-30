package playlist

import (
	"path/filepath"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

type Env struct {
	library.Env
	PlaylistDir   string
	PlaylistStars string
	MusicRoot     string
	MusicConfig   string
}

func EnvFrom(p paths.Env) Env {
	return Env{
		Env:           library.EnvFrom(p),
		PlaylistDir:   p.PlaylistDir,
		PlaylistStars: filepath.Join(p.StateDir, "playlist-stars.json"),
		MusicRoot:     p.MusicRoot,
		MusicConfig:   p.MusicConfig,
	}
}

func (e Env) currentM3U() string {
	return filepath.Join(e.PlaylistDir, "current.m3u")
}

func (e Env) currentTracksJSON() string {
	return filepath.Join(e.PlaylistDir, "current.tracks.json")
}
