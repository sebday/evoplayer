package library

import (
	"database/sql"
	"encoding/json"
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

func tagGenreForPath(path, fallback string) string {
	tag, err := tags.ReadTags(path)
	if err == nil {
		if g := strings.TrimSpace(tag.Genre); g != "" {
			return g
		}
	}
	return fallback
}

func applyTagGenreIfEmpty(row *Track) {
	if row == nil || row.Path == "" {
		return
	}
	if strings.TrimSpace(row.Genre) != "" {
		return
	}
	row.Genre = tagGenreForPath(row.Path, row.Genre)
}

func applyTagGenre(row *Track) {
	applyTagGenreIfEmpty(row)
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
			enrichTrackAssets(env, &row)
			row.Playlist = playlist
			row.DurationLabel = playback.FormatTime(row.Duration)
			return row, nil
		}
	}
	row := trackFromTagsCache(env, path)
	if row.Title == "" {
		if tag, err := tags.ReadTags(path); err == nil {
			row.Title = tag.Title
			row.Artist = tag.Artist
			row.Album = tag.Album
			row.Year = tag.Year
			row.Label = tag.Label
		}
	}
	applyTagGenre(&row)
	if row.Title == "" {
		row.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	enrichTrackAssets(env, &row)
	row.Playlist = playlist
	row.DurationLabel = playback.FormatTime(row.Duration)
	return row, nil
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

func trackFromTagsCache(env Env, path string) Track {
	matches, _ := filepath.Glob(filepath.Join(env.TracksCacheDir, "*.tags.json"))
	for _, cache := range matches {
		raw, err := os.ReadFile(cache)
		if err != nil {
			continue
		}
		var items []Track
		if json.Unmarshal(raw, &items) != nil {
			continue
		}
		for _, item := range items {
			if item.Path == path {
				enrichTrackAssets(env, &item)
				if isLiked(env, path) {
					item.Liked = true
				}
				applyTagGenre(&item)
				return item
			}
		}
	}
	return Track{Path: path}
}

func isLiked(env Env, path string) bool {
	return likesForEnv(env).has(path)
}
