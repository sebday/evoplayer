package playlist

import (
	"encoding/json"
	"os"
)

func loadStars(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var stars []string
	if err := json.Unmarshal(raw, &stars); err != nil {
		return []string{}, nil
	}
	return stars, nil
}

func saveStars(path string, stars []string) error {
	if stars == nil {
		stars = []string{}
	}
	raw, err := json.Marshal(stars)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func isStarred(stars []string, name string) bool {
	for _, s := range stars {
		if s == name {
			return true
		}
	}
	return false
}

func toggleStar(stars []string, name string) ([]string, bool) {
	if isStarred(stars, name) {
		out := make([]string, 0, len(stars))
		for _, s := range stars {
			if s != name {
				out = append(out, s)
			}
		}
		return out, false
	}
	out := append(uniqueSlice(stars), name)
	return out, true
}

func renameStar(stars []string, old, new string) []string {
	if !isStarred(stars, old) {
		return stars
	}
	out := make([]string, 0, len(stars))
	for _, s := range stars {
		if s == old {
			out = append(out, new)
		} else {
			out = append(out, s)
		}
	}
	return uniqueSlice(out)
}

func removeStar(stars []string, name string) []string {
	out := make([]string, 0, len(stars))
	for _, s := range stars {
		if s != name {
			out = append(out, s)
		}
	}
	return out
}

func uniqueSlice(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, s := range items {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
