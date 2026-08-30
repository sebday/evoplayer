package library

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

func PreviewIncoming(env Env, path string) map[string]any {
	probed, _ := tags.Probe(path)
	title := strings.TrimSpace(probed.Tag.Title)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	overlay := overlayGenre(env, path)
	folder := overlay
	if folder == "" {
		folder = genreFolderFromTags(env, probed.Tag)
	}
	genre := strings.TrimSpace(probed.Tag.Genre)
	if overlay != "" {
		genre = overlay
	}
	art, _, _ := EnsureArt(env, path)
	out := map[string]any{
		"path":           path,
		"title":          title,
		"artist":         strings.TrimSpace(probed.Tag.Artist),
		"album":          strings.TrimSpace(probed.Tag.Album),
		"folder":         folder,
		"genre":          genre,
		"year":           strings.TrimSpace(probed.Tag.Year),
		"duration":       probed.Duration,
		"duration_label": playback.FormatTime(probed.Duration),
		"needs_folder":   folder == "",
		"needs_genre":    folder == "",
	}
	if art != "" {
		out["art"] = art
	}
	return out
}

func ListIncoming(env Env) []map[string]any {
	incoming := filepath.Join(env.MusicRoot, ".incoming")
	entries, err := os.ReadDir(incoming)
	if err != nil {
		return nil
	}
	var out []map[string]any
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(incoming, e.Name())
		if incomingSkipFile(path) || !audio.IsAudio(path) {
			continue
		}
		out = append(out, PreviewIncoming(env, path))
	}
	return out
}
