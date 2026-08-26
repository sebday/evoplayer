package library

import "github.com/sebday/evoplayer/internal/paths"

type Env struct {
	MusicRoot      string
	StateDir       string
	CacheDir       string
	LikesFile      string
	TracksCacheDir string
	WaveformDir    string
	ArtDir         string
	LibraryDB      string
}

func EnvFrom(p paths.Env) Env {
	return Env{
		MusicRoot:      p.MusicRoot,
		StateDir:       p.StateDir,
		CacheDir:       p.CacheDir,
		LikesFile:      p.LikesFile,
		TracksCacheDir: p.TracksCacheDir,
		WaveformDir:    p.WaveformDir,
		ArtDir:         p.ArtDir,
		LibraryDB:      p.LibraryDB,
	}
}
