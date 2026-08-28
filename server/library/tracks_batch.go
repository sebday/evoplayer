package library

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

const tracksBatchSize = 400

// TracksForPaths returns metadata for paths in order, using one DB query per batch.
func TracksForPaths(env Env, paths []string, playlistName string) []Track {
	if len(paths) == 0 {
		return nil
	}
	byPath := make(map[string]Track, len(paths))
	db, dbErr := EnsureDB(env)
	if dbErr == nil && db != nil {
		for i := 0; i < len(paths); i += tracksBatchSize {
			end := i + tracksBatchSize
			if end > len(paths) {
				end = len(paths)
			}
			chunk, err := queryTracksByPaths(db, env, paths[i:end])
			if err != nil {
				continue
			}
			for p, row := range chunk {
				byPath[p] = row
			}
		}
	}
	out := make([]Track, 0, len(paths))
	for _, p := range paths {
		if row, ok := byPath[p]; ok {
			row.Playlist = playlistName
			row.DurationLabel = playback.FormatTime(row.Duration)
			out = append(out, row)
			continue
		}
		row := fallbackTrack(env, p)
		row.Playlist = playlistName
		row.DurationLabel = playback.FormatTime(row.Duration)
		out = append(out, row)
	}
	return out
}

func queryTracksByPaths(db *sql.DB, env Env, paths []string) (map[string]Track, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	likesForEnv(env).preload()
	placeholders := make([]string, len(paths))
	args := make([]interface{}, len(paths))
	for i, p := range paths {
		placeholders[i] = "?"
		args[i] = p
	}
	query := fmt.Sprintf(`
SELECT path, genre, title, artist, album, year, label, duration, art, waveform, liked
FROM tracks WHERE path IN (%s)`, strings.Join(placeholders, ","))
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]Track, len(paths))
	for rows.Next() {
		var row Track
		var liked int
		if err := rows.Scan(
			&row.Path, &row.Genre, &row.Title, &row.Artist, &row.Album, &row.Year, &row.Label,
			&row.Duration, &row.Art, &row.Waveform, &liked,
		); err != nil {
			return nil, err
		}
		row.Liked = liked == 1
		if !row.Liked {
			row.Liked = isLiked(env, row.Path)
		}
		enrichTrackAssets(env, &row)
		applyTagGenreIfEmpty(&row)
		out[row.Path] = row
	}
	return out, rows.Err()
}

func fallbackTrack(env Env, path string) Track {
	if path == "" {
		return Track{}
	}
	if _, err := os.Stat(path); err != nil {
		return Track{Path: path}
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
	if row.Duration <= 0 {
		row.Duration = playback.DurationForPath(path)
	}
	enrichTrackAssets(env, &row)
	return row
}
