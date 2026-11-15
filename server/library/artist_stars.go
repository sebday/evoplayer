package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func artistStarsPath(env Env) string {
	return filepath.Join(env.StateDir, "artist-stars.json")
}

func loadArtistStars(env Env) ([]string, error) {
	raw, err := os.ReadFile(artistStarsPath(env))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return []string{}, nil
	}
	return names, nil
}

func saveArtistStars(env Env, names []string) error {
	if env.StateDir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(env.StateDir, 0o755); err != nil {
		return err
	}
	if names == nil {
		names = []string{}
	}
	raw, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return WriteFileAtomic(artistStarsPath(env), append(raw, '\n'), 0o644)
}

func artistStarSet(env Env) map[string]bool {
	names, err := loadArtistStars(env)
	if err != nil {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(names))
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if key != "" {
			out[key] = true
		}
	}
	return out
}

func artistStarred(env Env, name string) bool {
	return artistStarSet(env)[strings.ToLower(strings.TrimSpace(name))]
}

// ToggleArtistStar favorites or unfavorites an artist by name.
func ToggleArtistStar(env Env, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, os.ErrInvalid
	}
	names, err := loadArtistStars(env)
	if err != nil {
		return false, err
	}
	key := strings.ToLower(name)
	kept := make([]string, 0, len(names))
	found := false
	for _, cur := range names {
		if strings.ToLower(strings.TrimSpace(cur)) == key {
			found = true
			continue
		}
		kept = append(kept, cur)
	}
	if found {
		return false, saveArtistStars(env, kept)
	}
	return true, saveArtistStars(env, append(kept, name))
}
