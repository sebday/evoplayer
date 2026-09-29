package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

func loadDoc(path string) (map[string]any, error) {
	doc := map[string]any{}
	if path == "" {
		return doc, nil
	}
	if _, err := os.Stat(path); err != nil {
		return doc, nil
	}
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func Load(path string) (map[string]map[string]any, error) {
	doc, err := loadDoc(path)
	if err != nil {
		return nil, err
	}
	data := map[string]map[string]any{}
	for section, val := range doc {
		if m, ok := val.(map[string]any); ok {
			data[section] = m
		}
	}
	return data, nil
}

func docSection(doc map[string]any, section string) map[string]any {
	if m, ok := doc[section].(map[string]any); ok {
		return m
	}
	m := map[string]any{}
	doc[section] = m
	return m
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
	doc, err := loadDoc(path)
	if err != nil {
		return err
	}
	sec := docSection(doc, section)
	if section == "soundcloud" && key == "oauth_token" {
		delete(sec, "oauth_token")
		_ = write(path, doc)
		return fmt.Errorf("evoplayer: soundcloud oauth is not stored in music.toml (browser cookie or pass)")
	}
	sec[key] = value
	if sc, ok := doc["soundcloud"].(map[string]any); ok {
		delete(sc, "likes_url")
		delete(sc, "oauth_token")
	}
	return write(path, doc)
}

// SetAll writes several keys of one section in a single rewrite.
func SetAll(path, section string, fields [][2]string) error {
	doc, err := loadDoc(path)
	if err != nil {
		return err
	}
	sec := docSection(doc, section)
	for _, f := range fields {
		sec[f[0]] = f[1]
	}
	if sc, ok := doc["soundcloud"].(map[string]any); ok {
		delete(sc, "likes_url")
		delete(sc, "oauth_token")
	}
	return write(path, doc)
}

func PruneDerived(path string) error {
	doc, err := loadDoc(path)
	if err != nil {
		return err
	}
	sc, ok := doc["soundcloud"].(map[string]any)
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
	return write(path, doc)
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

var sectionOrder = []string{"paths", "soundcloud", "genres", "genre_aliases", "playlist_folders"}

func sectionRank(name string) int {
	for i, s := range sectionOrder {
		if s == name {
			return i
		}
	}
	return len(sectionOrder)
}

func isTable(v any) bool {
	switch v.(type) {
	case map[string]any, []map[string]any:
		return true
	}
	return false
}

func encodeTOML(v map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func write(path string, doc map[string]any) error {
	top := map[string]any{}
	var sections []string
	for key, val := range doc {
		if isTable(val) {
			sections = append(sections, key)
		} else {
			top[key] = val
		}
	}
	sort.Slice(sections, func(i, j int) bool {
		ri, rj := sectionRank(sections[i]), sectionRank(sections[j])
		if ri != rj {
			return ri < rj
		}
		return sections[i] < sections[j]
	})
	var chunks [][]byte
	if len(top) > 0 {
		chunk, err := encodeTOML(top)
		if err != nil {
			return err
		}
		chunks = append(chunks, chunk)
	}
	for _, section := range sections {
		chunk, err := encodeTOML(map[string]any{section: doc[section]})
		if err != nil {
			return err
		}
		chunks = append(chunks, chunk)
	}
	content := append(bytes.Join(chunks, []byte("\n\n")), '\n')
	return writeAtomic(path, content)
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(name, mode)
	}
	if err == nil {
		err = os.Rename(name, path)
	}
	if err != nil {
		_ = os.Remove(name)
	}
	return err
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
