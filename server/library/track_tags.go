package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

type TrackTagsPatch struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Year   string `json:"year"`
	Genre  string `json:"genre"`
	Label  string `json:"label"`
}

type TrackTags struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Year   string `json:"year"`
	Genre  string `json:"genre"`
	Label  string `json:"label"`
}

func ReadTrackTags(path string) (TrackTags, error) {
	path = filepath.Clean(path)
	if path == "" {
		return TrackTags{}, fmt.Errorf("path required")
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return TrackTags{}, fmt.Errorf("not a file: %s", path)
	}
	if !playback.IsSupportedPath(path) {
		return TrackTags{}, fmt.Errorf("unsupported file: %s", path)
	}
	tag, err := tags.ReadTags(path)
	if err != nil {
		return TrackTags{}, err
	}
	return TrackTags{
		Title:  strings.TrimSpace(tag.Title),
		Artist: strings.TrimSpace(tag.Artist),
		Year:   strings.TrimSpace(tag.Year),
		Genre:  strings.TrimSpace(tag.Genre),
		Label:  strings.TrimSpace(tag.Label),
	}, nil
}

func UpdateTrackTags(env Env, path string, patch TrackTagsPatch) (Track, error) {
	path = filepath.Clean(path)
	if path == "" {
		return Track{}, fmt.Errorf("path required")
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return Track{}, fmt.Errorf("not a file: %s", path)
	}
	if !playback.IsSupportedPath(path) {
		return Track{}, fmt.Errorf("unsupported file: %s", path)
	}
	targets := map[string]string{
		"title":     strings.TrimSpace(patch.Title),
		"artist":    strings.TrimSpace(patch.Artist),
		"year":      strings.TrimSpace(patch.Year),
		"genre":     strings.TrimSpace(patch.Genre),
		"publisher": strings.TrimSpace(patch.Label),
	}
	if err := tags.Write(path, targets); err != nil {
		return Track{}, err
	}
	probed, _ := tags.Probe(path)
	genre := strings.TrimSpace(probed.Tag.Genre)
	if genre == "" {
		genre = genreFromPath(env.MusicRoot, path)
	}
	item := Track{
		Path:     path,
		Genre:    genre,
		Title:    probed.Tag.Title,
		Artist:   probed.Tag.Artist,
		Album:    probed.Tag.Album,
		Year:     probed.Tag.Year,
		Label:    probed.Tag.Label,
		Duration: probed.Duration,
	}
	if st, err := os.Stat(path); err == nil {
		if db, err := EnsureDB(env); err == nil && db != nil {
			tx, err := db.Begin()
			if err == nil {
				_ = upsertTrack(tx, env, item, st.ModTime().UnixNano(), st.Size())
				_ = tx.Commit()
			}
		}
	}
	return Meta(env, path, "")
}
