package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type File struct {
	Paths      map[string]any `toml:"paths"`
	Soundcloud map[string]any `toml:"soundcloud"`
	Skip       []string       `toml:"skip"`
	Extra      map[string]map[string]any
}

type VizView struct {
	Sensitivity    int     `json:"sensitivity"`
	Autosens       int     `json:"autosens"`
	NoiseReduction int     `json:"noise_reduction"`
	Monstercat     float64 `json:"monstercat"`
	FrameRate      int     `json:"frame_rate"`
	LowCutoff      int     `json:"low_cutoff"`
	HighCutoff     int     `json:"high_cutoff"`
}

type JSONView struct {
	Soundcloud struct {
		User        string `json:"user"`
		OAuthSource string `json:"oauth_source,omitempty"`
		ClientID    string `json:"client_id"`
	} `json:"soundcloud"`
	Paths struct {
		Root string `json:"root"`
	} `json:"paths"`
	Viz VizView `json:"viz"`
}

func Load(path string) (map[string]map[string]any, error) {
	data := map[string]map[string]any{}
	if path == "" {
		return data, nil
	}
	if _, err := os.Stat(path); err != nil {
		return data, nil
	}
	raw := map[string]any{}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, err
	}
	for section, val := range raw {
		m, ok := val.(map[string]any)
		if !ok {
			continue
		}
		data[section] = m
	}
	return data, nil
}

func Get(path, section, key, defaultVal string) (string, error) {
	data, err := Load(path)
	if err != nil {
		return defaultVal, err
	}
	sec, ok := data[section]
	if !ok {
		return defaultVal, nil
	}
	val, ok := sec[key]
	if !ok || val == nil {
		return defaultVal, nil
	}
	switch v := val.(type) {
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	default:
		return fmt.Sprint(v), nil
	}
}

func Set(path, section, key, value string) error {
	data, err := Load(path)
	if err != nil {
		return err
	}
	if data[section] == nil {
		data[section] = map[string]any{}
	}
	if section == "soundcloud" && key == "oauth_token" {
		delete(data[section], "oauth_token")
		_ = write(path, data)
		return fmt.Errorf("evoplayer: soundcloud oauth is not stored in music.toml (browser cookie or pass)")
	}
	data[section][key] = value
	if section == "soundcloud" || data["soundcloud"] != nil {
		delete(data["soundcloud"], "likes_url")
		delete(data["soundcloud"], "oauth_token")
	}
	return write(path, data)
}

func PruneDerived(path string) error {
	data, err := Load(path)
	if err != nil {
		return err
	}
	sc, ok := data["soundcloud"]
	if !ok {
		return nil
	}
	_, hasLikes := sc["likes_url"]
	_, hasOAuth := sc["oauth_token"]
	if !hasLikes && !hasOAuth {
		return nil
	}
	delete(sc, "likes_url")
	delete(sc, "oauth_token")
	return write(path, data)
}

func JSON(path, musicRoot string) (JSONView, error) {
	_ = PruneDerived(path)
	data, err := Load(path)
	if err != nil {
		return JSONView{}, err
	}
	var out JSONView
	sc := data["soundcloud"]
	if sc == nil {
		sc = map[string]any{}
	}
	if user, ok := sc["user"].(string); ok && user != "" {
		out.Soundcloud.User = user
	} else {
		out.Soundcloud.User = "seb-day"
	}
	if clientID, ok := sc["client_id"].(string); ok {
		out.Soundcloud.ClientID = clientID
	}
	root := musicRoot
	if root == "" {
		root = ResolveRoot(path)
	}
	out.Paths.Root = root
	vizView, err := VizJSON(path)
	if err != nil {
		return out, err
	}
	out.Viz = vizView
	return out, nil
}

func ReadRoot(paths ...string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		data, err := Load(path)
		if err != nil {
			continue
		}
		pathsSec := data["paths"]
		if pathsSec == nil {
			continue
		}
		if root, ok := pathsSec["root"].(string); ok && root != "" {
			return root
		}
	}
	return ""
}

var skipListRe = regexp.MustCompile(`(?ms)^\s*skip\s*=\s*\[(.*?)\]`)
var skipItemRe = regexp.MustCompile(`"([^"]+)"`)

func SkipDirs(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	text, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	match := skipListRe.FindSubmatch(text)
	if match == nil {
		return nil, nil
	}
	items := skipItemRe.FindAllStringSubmatch(string(match[1]), -1)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if len(item) > 1 {
			out = append(out, item[1])
		}
	}
	return out, nil
}

func esc(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func write(path string, data map[string]map[string]any) error {
	sections := make([]string, 0, len(data))
	for section := range data {
		sections = append(sections, section)
	}
	// stable section order: paths first, then soundcloud, genres, genre_aliases, then rest sorted
	sortSections := func(a, b string) bool {
		order := map[string]int{"paths": 0, "soundcloud": 1, "genres": 2, "genre_aliases": 3}
		oa, ob := order[a], order[b]
		if oa != ob {
			if oa == 0 {
				return true
			}
			if ob == 0 {
				return false
			}
			if oa == 1 {
				return true
			}
			if ob == 1 {
				return false
			}
			if oa == 2 {
				return true
			}
			if ob == 2 {
				return false
			}
			if oa == 3 {
				return true
			}
			if ob == 3 {
				return false
			}
		}
		return a < b
	}
	for i := 0; i < len(sections); i++ {
		for j := i + 1; j < len(sections); j++ {
			if !sortSections(sections[i], sections[j]) {
				sections[i], sections[j] = sections[j], sections[i]
			}
		}
	}

	var lines []string
	for _, section := range sections {
		vals := data[section]
		if vals == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("[%s]", section))
		keys := make([]string, 0, len(vals))
		for key := range vals {
			keys = append(keys, key)
		}
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		for _, key := range keys {
			val := vals[key]
			switch v := val.(type) {
			case string:
				lines = append(lines, fmt.Sprintf("%s = \"%s\"", key, esc(v)))
			case bool:
				if v {
					lines = append(lines, fmt.Sprintf("%s = true", key))
				} else {
					lines = append(lines, fmt.Sprintf("%s = false", key))
				}
			case []any:
				items := make([]string, 0, len(v))
				for _, item := range v {
					items = append(items, fmt.Sprintf("\"%s\"", esc(fmt.Sprint(item))))
				}
				lines = append(lines, fmt.Sprintf("%s = [%s]", key, strings.Join(items, ", ")))
			default:
				lines = append(lines, fmt.Sprintf("%s = %v", key, v))
			}
		}
		lines = append(lines, "")
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	content := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

func ValidateMusicRoot(root string) error {
	root = strings.TrimRight(root, "/")
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("evoplayer: music library not found: %s", root)
	}
	if !info.IsDir() {
		return fmt.Errorf("evoplayer: music library not found: %s", root)
	}
	return nil
}

func DefaultMusicRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	music := filepath.Join(home, "music")
	if dirExists(music) {
		return music
	}
	upper := filepath.Join(home, "Music")
	if dirExists(upper) {
		return upper
	}
	return music
}

func ResolveRoot(paths ...string) string {
	if root := ReadRoot(paths...); root != "" && dirExists(root) {
		return root
	}
	return DefaultMusicRoot()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
