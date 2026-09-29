package library

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

type SortResult struct {
	Folder string            `json:"folder"`
	Moved  int               `json:"moved"`
	Failed int               `json:"failed"`
	Moves  []MoveTrackResult `json:"moves,omitempty"`
}

func SortFolder(env Env, rel string) (SortResult, error) {
	rel = strings.TrimPrefix(strings.TrimPrefix(rel, "/"), "\\")
	dir := filepath.Join(env.MusicRoot, filepath.FromSlash(rel))
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return SortResult{}, fmt.Errorf("evoplayer: not a folder: %s", rel)
	}
	res := SortResult{Folder: rel}
	var db *sql.DB
	if env.LibraryDB != "" {
		if opened, err := OpenDB(env.LibraryDB); err == nil {
			db = opened
			defer db.Close()
		}
	}
	walkDir := filepath.Clean(dir)
	err = walkLibrary(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != walkDir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !playback.IsSupportedPath(path) {
			return nil
		}
		genre := GenreFromPath(env.MusicRoot, path)
		if genre == "" || !dirExists(filepath.Join(env.MusicRoot, genre)) {
			res.Failed++
			return nil
		}
		probed, _ := tags.Probe(path)
		canon, err := trackDestForFolder(env, path, genre, probed)
		if err != nil {
			res.Failed++
			return nil
		}
		if samePath(path, canon) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(canon), 0o755); err != nil {
			res.Failed++
			return nil
		}
		if _, err := os.Stat(canon); err == nil {
			res.Failed++
			return nil
		}
		if err := os.Rename(path, canon); err != nil {
			res.Failed++
			return nil
		}
		if db != nil {
			if st, err := os.Stat(canon); err == nil {
				_ = replaceTrack(db, env, path, trackFromProbe(canon, genre, probed), st.ModTime().UnixNano(), st.Size())
			}
		}
		appendPlacement(env, "sort", path, canon)
		res.Moved++
		res.Moves = append(res.Moves, MoveTrackResult{From: path, To: canon, Folder: genre})
		return nil
	})
	return res, err
}

func samePath(a, b string) bool {
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	return aa == bb
}

func appendPlacement(env Env, op, from, to string) {
	logPath := filepath.Join(env.StateDir, "placement.jsonl")
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	row := map[string]string{
		"id":   fmt.Sprintf("%s-%d", time.Now().Format(time.RFC3339Nano), os.Getpid()),
		"at":   time.Now().Format(time.RFC3339),
		"op":   op,
		"from": from,
		"to":   to,
	}
	raw, _ := json.Marshal(row)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}
