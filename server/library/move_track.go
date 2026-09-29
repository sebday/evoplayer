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
	matched := MatchLibraryGenre(env, folder)
	if matched == "" {
		return MoveTrackResult{}, fmt.Errorf("unknown library folder: %s", folder)
	}
	folder = matched
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
			_ = replaceTrack(db, env, path, trackFromProbe(dest, folder, probed), st.ModTime().UnixNano(), st.Size())
		}
	}
	appendPlacement(env, "move", path, dest)
	return res, nil
}

func embedGenreTag(path, genre string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".mp3" && ext != ".mp2" {
		return nil
	}
	return tags.EmbedMP3(path, map[string]string{"genre": genre}, nil, "")
}
