package library

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

type Track struct {
	Path          string  `json:"path"`
	Genre         string  `json:"genre,omitempty"`
	Title         string  `json:"title"`
	Artist        string  `json:"artist,omitempty"`
	Album         string  `json:"album,omitempty"`
	Year          string  `json:"year,omitempty"`
	Label         string  `json:"label,omitempty"`
	Duration      float64 `json:"duration"`
	Art           string  `json:"art,omitempty"`
	Thumb         string  `json:"thumb,omitempty"`
	Waveform      string  `json:"waveform,omitempty"`
	Liked         bool    `json:"liked,omitempty"`
	Type          string  `json:"type,omitempty"`
	Playlist      string  `json:"playlist,omitempty"`
	DurationLabel string  `json:"duration_label,omitempty"`
}

func trackFromProbe(path, genre string, probed tags.ProbeResult) Track {
	return Track{
		Path:     path,
		Genre:    genre,
		Title:    probed.Tag.Title,
		Artist:   probed.Tag.Artist,
		Album:    probed.Tag.Album,
		Year:     probed.Tag.Year,
		Label:    probed.Tag.Label,
		Duration: probed.Duration,
	}
}

func applyTagGenre(row *Track) {
	if row == nil || row.Path == "" || strings.TrimSpace(row.Genre) != "" {
		return
	}
	if tag, err := tags.ReadTags(row.Path); err == nil {
		row.Genre = strings.TrimSpace(tag.Genre)
	}
}

func Meta(env Env, path string, playlist string) (Track, error) {
	if path == "" {
		return Track{}, os.ErrNotExist
	}
	if _, err := os.Stat(path); err != nil {
		return Track{}, err
	}
	db, err := EnsureDB(env)
	if err == nil {
		if row, err := trackByPath(db, env, path); err == nil {
			row.Playlist = playlist
			row.DurationLabel = playback.FormatTime(row.Duration)
			return row, nil
		}
	}
	row := trackFromFileTags(env, path)
	enrichTrackAssets(env, &row)
	row.Playlist = playlist
	row.DurationLabel = playback.FormatTime(row.Duration)
	return row, nil
}

func trackFromFileTags(env Env, path string) Track {
	row := Track{Path: path, Liked: isLiked(env, path)}
	if tag, err := tags.ReadTags(path); err == nil {
		row.Title = tag.Title
		row.Artist = tag.Artist
		row.Album = tag.Album
		row.Year = tag.Year
		row.Label = tag.Label
		row.Genre = strings.TrimSpace(tag.Genre)
	}
	if row.Title == "" {
		row.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return row
}

func trackByPath(db *sql.DB, env Env, path string) (Track, error) {
	var row Track
	var liked int
	err := db.QueryRow(`
SELECT path, genre, title, artist, album, year, label, duration, art, waveform, liked
FROM tracks WHERE path=? LIMIT 1`, path).Scan(
		&row.Path, &row.Genre, &row.Title, &row.Artist, &row.Album, &row.Year, &row.Label,
		&row.Duration, &row.Art, &row.Waveform, &liked,
	)
	if err != nil {
		return Track{}, err
	}
	row.Liked = liked == 1
	if !row.Liked && isLiked(env, path) {
		row.Liked = true
	}
	enrichTrackAssets(env, &row)
	applyTagGenre(&row)
	return row, nil
}

func isLiked(env Env, path string) bool {
	return likesForEnv(env).has(path)
}
