package library

import (
	"database/sql"
	"encoding/json"
	"os"
	"sort"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/tags"
)

// MixMinDurationSec is the duration at which a track is treated as a mix
// (imports, dest folders, and the generated mixes playlist).
const MixMinDurationSec = 25 * 60

func IsMix(path string, dur float64) bool {
	return dur >= float64(MixMinDurationSec)
}

func LikedMixPaths(db *sql.DB, env Env) ([]string, error) {
	if db != nil && Ready(db) {
		rows, err := db.Query(`
SELECT path, duration FROM tracks WHERE liked=1 AND duration >= ? ORDER BY path`, MixMinDurationSec)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var paths []string
		for rows.Next() {
			var path string
			var dur float64
			if err := rows.Scan(&path, &dur); err != nil {
				return nil, err
			}
			if !keepLikedMixPath(path, dur) {
				continue
			}
			paths = append(paths, path)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return paths, nil
	}
	return likedMixPathsFromFile(env)
}

func likedMixPathsFromFile(env Env) ([]string, error) {
	raw, err := os.ReadFile(env.LikesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var likes map[string]json.RawMessage
	if len(raw) > 0 {
		if json.Unmarshal(raw, &likes) != nil {
			return nil, nil
		}
	}
	paths := make([]string, 0, len(likes))
	for p := range likes {
		dur := 0.0
		if probed, err := tags.Probe(p); err == nil {
			dur = probed.Duration
		}
		if !keepLikedMixPath(p, dur) {
			continue
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

func keepLikedMixPath(path string, dur float64) bool {
	if path == "" || dur < float64(MixMinDurationSec) {
		return false
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	return audio.IsAudio(path)
}
