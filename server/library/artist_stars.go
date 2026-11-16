package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func namedStarsPath(env Env, file string) string {
	return filepath.Join(env.StateDir, file)
}

func loadNamedStars(env Env, file string) ([]string, error) {
	raw, err := os.ReadFile(namedStarsPath(env, file))
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

func saveNamedStars(env Env, file string, names []string) error {
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
	return WriteFileAtomic(namedStarsPath(env, file), append(raw, '\n'), 0o644)
}

func namedStarSet(env Env, file string) map[string]bool {
	names, err := loadNamedStars(env, file)
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

func namedStarred(env Env, file, name string) bool {
	return namedStarSet(env, file)[strings.ToLower(strings.TrimSpace(name))]
}

func toggleNamedStar(env Env, file, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, os.ErrInvalid
	}
	names, err := loadNamedStars(env, file)
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
		return false, saveNamedStars(env, file, kept)
	}
	return true, saveNamedStars(env, file, append(kept, name))
}

func artistStarSet(env Env) map[string]bool {
	return namedStarSet(env, "artist-stars.json")
}

func starNames(env Env, file string) []string {
	names, err := loadNamedStars(env, file)
	if err != nil {
		return nil
	}
	var out []string
	seen := map[string]struct{}{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		key := strings.ToLower(name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

// ArtistStarNames is the saved spelling of each favorited artist.
func ArtistStarNames(env Env) []string {
	return starNames(env, "artist-stars.json")
}

// AlbumStarNames is the saved spelling of each favorited album.
func AlbumStarNames(env Env) []string {
	return starNames(env, "album-stars.json")
}

func artistStarred(env Env, name string) bool {
	return namedStarred(env, "artist-stars.json", name)
}

// ArtistStarred reports whether the artist is favorited.
func ArtistStarred(env Env, name string) bool {
	return artistStarred(env, name)
}

// ToggleArtistStar favorites or unfavorites an artist by name.
func ToggleArtistStar(env Env, name string) (bool, error) {
	return toggleNamedStar(env, "artist-stars.json", name)
}

func albumStarSet(env Env) map[string]bool {
	return namedStarSet(env, "album-stars.json")
}

func albumStarred(env Env, name string) bool {
	return namedStarred(env, "album-stars.json", name)
}

// AlbumStarred reports whether the album is favorited.
func AlbumStarred(env Env, name string) bool {
	return albumStarred(env, name)
}

// ToggleAlbumStar favorites or unfavorites an album by name.
func ToggleAlbumStar(env Env, name string) (bool, error) {
	return toggleNamedStar(env, "album-stars.json", name)
}

func labelStarSet(env Env) map[string]bool {
	return namedStarSet(env, "label-stars.json")
}

func labelStarred(env Env, name string) bool {
	return namedStarred(env, "label-stars.json", name)
}

// LabelStarNames is the saved spelling of each favorited label.
func LabelStarNames(env Env) []string {
	return starNames(env, "label-stars.json")
}

// LabelStarred reports whether the label is favorited.
func LabelStarred(env Env, name string) bool {
	return labelStarred(env, name)
}

// ToggleLabelStar favorites or unfavorites a label by name.
func ToggleLabelStar(env Env, name string) (bool, error) {
	return toggleNamedStar(env, "label-stars.json", name)
}
