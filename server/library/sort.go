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
	Folder string `json:"folder"`
	Moved  int    `json:"moved"`
	Failed int    `json:"failed"`
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
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !playback.IsSupportedPath(path) {
			return nil
		}
		probed, _ := tags.Probe(path)
		canon, err := incomingDest(env, path, probed)
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
				item := Track{
					Path:     canon,
					Genre:    genreFromPath(env.MusicRoot, canon),
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
		appendPlacement(env, "sort", path, canon)
		res.Moved++
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
