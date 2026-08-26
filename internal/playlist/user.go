package playlist

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type ActionResult struct {
	OK     bool   `json:"ok"`
	Name   string `json:"name,omitempty"`
	Old    string `json:"old,omitempty"`
	New    string `json:"new,omitempty"`
	Action string `json:"action,omitempty"`
}

func CreateUser(env Env, name string) (ActionResult, error) {
	if err := validateUserName(env, name); err != nil {
		return ActionResult{}, err
	}
	list := filepath.Join(env.PlaylistDir, name+".m3u")
	if _, err := os.Stat(list); err == nil {
		return ActionResult{}, fmt.Errorf("playlist already exists: %s", name)
	}
	if err := os.MkdirAll(env.PlaylistDir, 0o755); err != nil {
		return ActionResult{}, err
	}
	if err := os.WriteFile(list, []byte("#EXTM3U\n"), 0o644); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{OK: true, Name: name, Action: "create"}, nil
}

func RenameUser(env Env, old, new string) (ActionResult, error) {
	if !isUserEditable(env, old) {
		return ActionResult{}, fmt.Errorf("cannot rename playlist: %s", old)
	}
	if err := validateUserName(env, new); err != nil {
		return ActionResult{}, err
	}
	from := filepath.Join(env.PlaylistDir, old+".m3u")
	to := filepath.Join(env.PlaylistDir, new+".m3u")
	if _, err := os.Stat(from); err != nil {
		return ActionResult{}, err
	}
	if _, err := os.Stat(to); err == nil {
		return ActionResult{}, fmt.Errorf("playlist already exists: %s", new)
	}
	if err := os.Rename(from, to); err != nil {
		return ActionResult{}, err
	}
	stars, _ := loadStars(env.PlaylistStars)
	stars = renameStar(stars, old, new)
	_ = saveStars(env.PlaylistStars, stars)
	return ActionResult{OK: true, Old: old, New: new, Action: "rename"}, nil
}

func DeleteUser(env Env, name string) (ActionResult, error) {
	if !isUserEditable(env, name) {
		return ActionResult{}, fmt.Errorf("cannot delete playlist: %s", name)
	}
	if err := os.Remove(filepath.Join(env.PlaylistDir, name+".m3u")); err != nil {
		return ActionResult{}, err
	}
	stars, _ := loadStars(env.PlaylistStars)
	stars = removeStar(stars, name)
	_ = saveStars(env.PlaylistStars, stars)
	return ActionResult{OK: true, Name: name, Action: "delete"}, nil
}

func isReservedName(name string) bool {
	switch name {
	case "all", "favorites", "current", "":
		return true
	default:
		return false
	}
}

func isUserEditable(env Env, name string) bool {
	if isReservedName(name) {
		return false
	}
	if isGenreDir(env, name) {
		return false
	}
	_, err := os.Stat(filepath.Join(env.PlaylistDir, name+".m3u"))
	return err == nil
}

func validateUserName(env Env, name string) error {
	if name == "" {
		return fmt.Errorf("invalid playlist name: %s", name)
	}
	if isReservedName(name) {
		return fmt.Errorf("invalid playlist name: %s", name)
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid playlist name: %s", name)
	}
	r := []rune(name)
	if len(r) == 0 || !unicode.IsLetter(r[0]) && !unicode.IsDigit(r[0]) {
		return fmt.Errorf("invalid playlist name: %s", name)
	}
	for _, ch := range r {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-' || ch == ' ' {
			continue
		}
		return fmt.Errorf("invalid playlist name: %s", name)
	}
	if isGenreDir(env, name) {
		return fmt.Errorf("invalid playlist name: %s", name)
	}
	return nil
}
