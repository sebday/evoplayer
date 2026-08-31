package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

type MoveTrackResult struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Folder string `json:"folder"`
}

func MoveTrackToFolder(env Env, path, folder string) (MoveTrackResult, error) {
	path = filepath.Clean(path)
	if path == "" {
		return MoveTrackResult{}, fmt.Errorf("path required")
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return MoveTrackResult{}, fmt.Errorf("not a file: %s", path)
	}
	if !playback.IsSupportedPath(path) {
		return MoveTrackResult{}, fmt.Errorf("unsupported file: %s", path)
	}
	root := filepath.Clean(env.MusicRoot)
	if root == "" {
		return MoveTrackResult{}, fmt.Errorf("music root not set")
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return MoveTrackResult{}, fmt.Errorf("not under music root: %s", path)
	}
	folder = matchLibraryFolder(env, folder)
	if folder == "" {
		return MoveTrackResult{}, fmt.Errorf("unknown library folder: %s", folder)
	}
	probed, _ := tags.Probe(path)
	dest, err := trackDestForFolder(env, path, folder, probed)
	if err != nil {
		return MoveTrackResult{}, err
	}
	res := MoveTrackResult{From: path, To: dest, Folder: folder}
	if samePath(path, dest) {
		if err := embedGenreTag(dest, folder); err != nil {
			return res, err
		}
		if db, err := EnsureDB(env); err == nil && db != nil {
			_, _ = db.Exec(`UPDATE tracks SET genre=? WHERE path=?`, folder, path)
		}
		return res, nil
	}
	if _, err := os.Stat(dest); err == nil {
		return MoveTrackResult{}, fmt.Errorf("destination exists: %s", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return MoveTrackResult{}, err
	}
	if err := os.Rename(path, dest); err != nil {
		return MoveTrackResult{}, err
	}
	if err := embedGenreTag(dest, folder); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: move genre embed: %v\n", err)
	}
	if db, err := EnsureDB(env); err == nil && db != nil {
		if st, err := os.Stat(dest); err == nil {
			item := Track{
				Path:     dest,
				Genre:    folder,
				Title:    probed.Tag.Title,
				Artist:   probed.Tag.Artist,
				Album:    probed.Tag.Album,
				Year:     probed.Tag.Year,
				Label:    probed.Tag.Label,
				Duration: probed.Duration,
			}
			tx, err := db.Begin()
			if err == nil {
				_, _ = tx.Exec(`DELETE FROM tracks WHERE path=?`, path)
				_ = upsertTrack(tx, env, item, st.ModTime().UnixNano(), st.Size())
				_ = tx.Commit()
			}
		}
	}
	appendPlacement(env, "move", path, dest)
	res.To = dest
	return res, nil
}

func trackDestForFolder(env Env, path, folder string, probed tags.ProbeResult) (string, error) {
	tag := probed.Tag
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "" {
		ext = "mp3"
	}
	base := trackFilename(tag.Artist, tag.Title, path, ext)
	if base == "" {
		return "", fmt.Errorf("cannot name %s", path)
	}
	dir := filepath.Join(env.MusicRoot, folder, "soundcloud")
	if incomingIsMix(path, base, probed.Duration) {
		year := mixYear(tag.Year, path, base)
		dir = filepath.Join(env.MusicRoot, folder, "mixes", year)
	} else if isYouTubeSource(path, tag) {
		year := mixYear(tag.Year, path, base)
		dir = filepath.Join(env.MusicRoot, folder, "youtube", year)
	}
	return filepath.Join(dir, base), nil
}

func embedGenreTag(path, genre string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".mp3" && ext != ".mp2" {
		return nil
	}
	return tags.EmbedMP3(path, map[string]string{"genre": genre}, nil, "")
}
