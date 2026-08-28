package tags

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	id3v2 "github.com/bogem/id3v2"
)

const minYear = 1985
const maxYear = 2026

var audioExts = map[string]struct{}{
	".mp3": {}, ".mp2": {}, ".m4a": {},
}

var knownYears = map[string]int{
	"bailey_gq-metalheadz_history_sessions_phonox-05-04-24": 2024,
	"future_beats_radio_show_04-06-15":                      2015,
	"future_beats_radio_show_04-04-13":                      2013,
}

var tagFields = []string{"title", "artist", "genre", "year", "album", "publisher", "catalognumber"}

type TagMap map[string]string

type Change struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type FileResult struct {
	Path    string            `json:"path"`
	Changed bool              `json:"changed"`
	Changes map[string]Change `json:"changes,omitempty"`
	Error   bool              `json:"error,omitempty"`
}

type DirResult struct {
	Root         string `json:"root"`
	ChangedFiles int    `json:"changed_files"`
	Failed       int    `json:"failed"`
}

func StandardizePath(musicRoot, target string) (any, int, error) {
	root := filepath.Clean(musicRoot)
	path := filepath.Clean(target)
	info, err := os.Stat(path)
	if err != nil {
		return nil, 1, err
	}
	if info.IsDir() {
		result := DirResult{Root: path}
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !isAudio(p) {
				return nil
			}
			res := standardizeFile(root, p)
			if res.Error {
				result.Failed++
				return nil
			}
			if res.Changed {
				result.ChangedFiles++
			}
			return nil
		})
		return result, exitCode(result.Failed), err
	}
	if !isAudio(path) {
		return nil, 1, fmt.Errorf("not an audio file")
	}
	res := standardizeFile(root, path)
	if res.Error {
		return res, 1, nil
	}
	return res, 0, nil
}

func exitCode(failed int) int {
	if failed > 0 {
		return 1
	}
	return 0
}

func isAudio(path string) bool {
	_, ok := audioExts[strings.ToLower(filepath.Ext(path))]
	return ok
}

func standardizeFile(musicRoot, path string) FileResult {
	before := readTags(path)
	targets := targetTags(musicRoot, path, before)
	changes := map[string]Change{}
	for _, key := range tagFields {
		newVal := targets[key]
		oldVal := before[key]
		if newVal != "" && newVal != oldVal {
			changes[key] = Change{From: oldVal, To: newVal}
		}
	}
	if len(changes) == 0 {
		return FileResult{Path: path, Changed: false}
	}
	writeMap := map[string]string{}
	for key, diff := range changes {
		writeMap[key] = diff.To
	}
	if !writeTags(path, writeMap) {
		return FileResult{Path: path, Changed: false, Error: true}
	}
	return FileResult{Path: path, Changed: true, Changes: changes}
}

func readTags(path string) TagMap {
	out := TagMap{
		"title": "", "artist": "", "genre": "", "album": "",
		"year": "", "publisher": "", "catalognumber": "",
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3", ".mp2":
		tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
		if err != nil {
			break
		}
		defer tag.Close()
		out["title"] = strings.TrimSpace(tag.Title())
		out["artist"] = strings.TrimSpace(firstNonEmpty(tag.Artist(), textFrame(tag, "TPE2")))
		out["genre"] = strings.TrimSpace(tag.Genre())
		out["album"] = strings.TrimSpace(tag.Album())
		out["publisher"] = strings.TrimSpace(textFrame(tag, "TPUB"))
		out["year"] = yearFromText(firstNonEmpty(textFrame(tag, "TDRC"), textFrame(tag, "TYER")))
		for _, frame := range tag.GetFrames("TXXX") {
			if user, ok := frame.(id3v2.UserDefinedTextFrame); ok {
				desc := strings.ToLower(user.Description)
				if desc == "catalognumber" || desc == "catalog" || desc == "catalogue" {
					out["catalognumber"] = strings.TrimSpace(user.Value)
					break
				}
			}
		}
	case ".m4a":
		readM4ATags(path, out)
	}
	if out["title"] == "" {
		out["title"] = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return out
}

func textFrame(tag *id3v2.Tag, id string) string {
	frame := tag.GetLastFrame(id)
	if frame == nil {
		return ""
	}
	if text, ok := frame.(id3v2.TextFrame); ok {
		return text.Text
	}
	return fmt.Sprint(frame)
}

func readM4ATags(path string, out TagMap) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", path)
	raw, err := cmd.Output()
	if err != nil {
		return
	}
	var payload struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return
	}
	tags := payload.Format.Tags
	lookup := map[string]string{}
	for k, v := range tags {
		lookup[strings.ToLower(k)] = strings.TrimSpace(v)
	}
	pick := func(keys ...string) string {
		for _, key := range keys {
			if val := lookup[strings.ToLower(key)]; val != "" {
				return val
			}
		}
		return ""
	}
	out["title"] = pick("title", "titl", "©nam")
	out["artist"] = pick("artist", "album_artist", "albumartist", "©art")
	out["genre"] = pick("genre", "©gen")
	out["album"] = pick("album", "©alb")
	out["year"] = yearFromText(pick("date", "year", "©day"))
	out["publisher"] = pick("label", "publisher", "organization", "tpub")
	out["catalognumber"] = pick("catalognumber", "catalog", "catalogue")
}

func yearFromText(raw string) string {
	if m := regexp.MustCompile(`(\d{4})`).FindStringSubmatch(raw); len(m) > 1 {
		return m[1]
	}
	return ""
}

func writeTags(path string, targets map[string]string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3", ".mp2":
		return writeMP3Tags(path, targets)
	case ".m4a":
		return writeM4ATags(path, targets)
	}
	return false
}

func writeMP3Tags(path string, targets map[string]string) bool {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		tag = id3v2.NewEmptyTag()
	}
	if v, ok := targets["title"]; ok {
		tag.SetTitle(v)
	}
	if v, ok := targets["artist"]; ok {
		tag.SetArtist(v)
	}
	if v, ok := targets["genre"]; ok {
		tag.SetGenre(v)
	}
	if v, ok := targets["year"]; ok {
		tag.AddTextFrame("TDRC", tag.DefaultEncoding(), v)
	}
	if v, ok := targets["album"]; ok {
		tag.SetAlbum(v)
	}
	if v, ok := targets["publisher"]; ok {
		tag.AddTextFrame("TPUB", tag.DefaultEncoding(), v)
	}
	if v, ok := targets["catalognumber"]; ok {
		keep := tag.GetFrames("TXXX")
		tag.DeleteFrames("TXXX")
		for _, frame := range keep {
			if user, ok := frame.(id3v2.UserDefinedTextFrame); ok {
				desc := strings.ToLower(user.Description)
				if desc == "catalognumber" {
					continue
				}
				tag.AddUserDefinedTextFrame(user)
			}
		}
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Description: "CATALOGNUMBER",
			Value:       v,
		})
	}
	return tag.Save() == nil
}

func writeM4ATags(path string, targets map[string]string) bool {
	args := []string{"-y", "-hide_banner", "-loglevel", "error", "-i", path, "-c", "copy"}
	add := func(key, val string) {
		if val != "" {
			args = append(args, "-metadata", key+"="+val)
		}
	}
	if v, ok := targets["title"]; ok {
		add("title", v)
	}
	if v, ok := targets["artist"]; ok {
		add("artist", v)
	}
	if v, ok := targets["genre"]; ok {
		add("genre", v)
	}
	if v, ok := targets["year"]; ok {
		add("date", v)
	}
	if v, ok := targets["album"]; ok {
		add("album", v)
	}
	if v, ok := targets["publisher"]; ok {
		add("label", v)
	}
	if v, ok := targets["catalognumber"]; ok {
		add("CATALOGNUMBER", v)
	}
	tmp, err := os.CreateTemp("", "evo-tag-*"+filepath.Ext(path))
	if err != nil {
		return false
	}
	tmpPath := tmp.Name()
	tmp.Close()
	args = append(args, tmpPath)
	if err := exec.Command("ffmpeg", args...).Run(); err != nil {
		os.Remove(tmpPath)
		return false
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return false
	}
	return true
}

func targetTags(musicRoot, path string, tags TagMap) TagMap {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	year := resolveYear(musicRoot, path, tags)
	folder := folderFromPath(musicRoot, path)

	artist := strings.TrimSpace(tags["artist"])
	title := strings.TrimSpace(tags["title"])
	fnArtist, fnTitle := ParseFilenameArtistTitle(stem)
	if fnArtist != "" {
		if artist == "" || strings.EqualFold(artist, fnTitle) {
			artist = fnArtist
		}
		if title == "" || title == stem {
			title = fnTitle
		}
	}
	if artist == "" && fnArtist != "" {
		artist = fnArtist
	}
	if title == "" {
		if fnTitle != "" {
			title = fnTitle
		} else {
			title = stem
		}
	}
	out := TagMap{"title": title, "artist": artist}
	if folder != "" {
		out["genre"] = folder
	}
	if year > 0 {
		out["year"] = strconv.Itoa(year)
	}
	if vinyl := vinylFromPath(musicRoot, path); vinyl != nil {
		for k, v := range vinyl {
			out[k] = v
		}
	}
	return out
}

func validYear(y int) bool {
	return y >= minYear && y <= maxYear
}

func yearFromFilename(stem string) int {
	patterns := []struct {
		re   *regexp.Regexp
		conv func([]string) (int, bool)
	}{
		{regexp.MustCompile(`^(20\d{2})-(\d{2})-(\d{2})`), func(m []string) (int, bool) {
			y, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[3])
			return y, validYear(y) && mo >= 1 && mo <= 12 && d >= 1 && d <= 31
		}},
		{regexp.MustCompile(`(?:^|[-_])(\d{4})(\d{2})(\d{2})$`), func(m []string) (int, bool) {
			y, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[3])
			return y, validYear(y) && mo >= 1 && mo <= 12 && d >= 1 && d <= 31
		}},
		{regexp.MustCompile(`(\d{2})[.-](\d{2})[.-](20\d{2})`), func(m []string) (int, bool) {
			y, _ := strconv.Atoi(m[3])
			mo, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[1])
			return y, validYear(y) && mo >= 1 && mo <= 12 && d >= 1 && d <= 31
		}},
		{regexp.MustCompile(`(\d{2})\.(\d{2})\.(\d{2})$`), func(m []string) (int, bool) {
			d, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			yy, _ := strconv.Atoi(m[3])
			y := 2000 + yy
			if yy >= 70 {
				y = 1900 + yy
			}
			return y, validYear(y) && mo >= 1 && mo <= 12 && d >= 0 && d <= 31
		}},
		{regexp.MustCompile(`(?:^|[-_])(\d{2})[.-](\d{2})[.-](\d{2})(?:\D|$)`), func(m []string) (int, bool) {
			yy, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[3])
			y := 2000 + yy
			if yy >= 70 {
				y = 1900 + yy
			}
			return y, validYear(y) && mo >= 1 && mo <= 12 && d >= 0 && d <= 31
		}},
		{regexp.MustCompile(`(\d{2})(\d{2})(\d{4})$`), func(m []string) (int, bool) {
			y, _ := strconv.Atoi(m[3])
			mo, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[1])
			return y, validYear(y) && mo >= 1 && mo <= 12 && d >= 1 && d <= 31
		}},
	}
	for _, p := range patterns {
		if m := p.re.FindStringSubmatch(stem); m != nil {
			if y, ok := p.conv(m); ok {
				return y
			}
		}
	}
	return 0
}

func resolvePathYear(musicRoot, path string) int {
	rel, err := filepath.Rel(filepath.Clean(musicRoot), filepath.Clean(path))
	if err != nil {
		rel = filepath.Base(path)
	}
	if y := yearFromFilename(strings.TrimSuffix(rel, filepath.Ext(rel))); y > 0 {
		return y
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	for i, part := range parts {
		if part == "mixes" && i+1 < len(parts) {
			if year, err := strconv.Atoi(parts[i+1]); err == nil && validYear(year) {
				return year
			}
		}
	}
	for _, part := range parts {
		if m := regexp.MustCompile(`^(19\d{2}|20\d{2})_`).FindStringSubmatch(part); len(m) > 1 {
			if y, _ := strconv.Atoi(m[1]); validYear(y) {
				return y
			}
		}
		if m := regexp.MustCompile(`(?:^|[-_])(19\d{2}|20\d{2})(?:\D|$)`).FindStringSubmatch(part); len(m) > 1 {
			if y, _ := strconv.Atoi(m[1]); validYear(y) {
				return y
			}
		}
	}
	return 0
}

func resolveYear(musicRoot, path string, tags TagMap) int {
	name := filepath.Base(path)
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	if y, ok := knownYears[name]; ok {
		return y
	}
	if y, ok := knownYears[stem]; ok {
		return y
	}
	if y := resolvePathYear(musicRoot, path); validYear(y) {
		return y
	}
	if m := regexp.MustCompile(`\((19\d{2}|20\d{2})\)`).FindStringSubmatch(stem); len(m) > 1 {
		if y, _ := strconv.Atoi(m[1]); validYear(y) {
			return y
		}
	}
	if tags["year"] != "" {
		if y, err := strconv.Atoi(tags["year"]); err == nil && validYear(y) {
			return y
		}
	}
	return 0
}

func folderFromPath(musicRoot, path string) string {
	rel, err := filepath.Rel(filepath.Clean(musicRoot), filepath.Clean(path))
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) == 0 || parts[0] == "." || strings.HasPrefix(parts[0], ".") {
		return ""
	}
	return parts[0]
}

func vinylFromPath(musicRoot, path string) TagMap {
	rel, err := filepath.Rel(filepath.Clean(musicRoot), filepath.Clean(path))
	if err != nil {
		return nil
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) < 4 || parts[0] == "" || strings.HasPrefix(parts[0], ".") || parts[1] != "vinyl" {
		return nil
	}
	if parts[2] == "_misc" {
		return nil
	}
	label := vinylLabelName(parts[2])
	if label == "" {
		return nil
	}
	out := TagMap{"publisher": label}
	if cat := vinylCatalogFromFolder(parts[3]); cat != "" {
		out["album"] = cat
		out["catalognumber"] = cat
	}
	return out
}

func vinylLabelName(folder string) string {
	name := strings.ReplaceAll(folder, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	words := strings.Fields(strings.ToLower(name))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func vinylCatalogFromFolder(releaseFolder string) string {
	isYearToken := func(tok string) bool {
		matched, _ := regexp.MatchString(`^(19|20)\d{2}$`, tok)
		return matched
	}
	isCatToken := func(tok string) bool {
		tok = strings.ToUpper(tok)
		matched, _ := regexp.MatchString(`^[A-Z0-9]{3,16}$`, tok)
		if !matched || isYearToken(tok) {
			return false
		}
		hasLetter, _ := regexp.MatchString(`[A-Z]`, tok)
		hasDigit, _ := regexp.MatchString(`\d`, tok)
		return hasLetter && hasDigit
	}
	score := func(tok string) int {
		digits := len(regexp.MustCompile(`\d`).FindAllString(tok, -1))
		return digits*10 + len(tok)
	}
	first := releaseFolder
	if idx := strings.Index(releaseFolder, "-"); idx >= 0 {
		first = releaseFolder[:idx]
	}
	if isCatToken(first) {
		return strings.ToUpper(first)
	}
	candidates := []string{}
	if strings.Contains(first, "_") {
		tail := first[strings.LastIndex(first, "_")+1:]
		digits := len(regexp.MustCompile(`\d`).FindAllString(tail, -1))
		if isCatToken(tail) && digits >= 2 {
			candidates = append(candidates, strings.ToUpper(tail))
		}
	}
	for _, tok := range regexp.MustCompile(`[-_]`).Split(releaseFolder, -1) {
		if isCatToken(tok) {
			candidates = append(candidates, strings.ToUpper(tok))
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	best := candidates[0]
	bestScore := score(best)
	for _, c := range candidates[1:] {
		if s := score(c); s > bestScore {
			best = c
			bestScore = s
		}
	}
	return best
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func PrintFileResult(res FileResult) {
	b, _ := json.Marshal(res)
	fmt.Println(string(b))
	if res.Changed {
		for key, diff := range res.Changes {
			from := diff.From
			if from == "" {
				from = "∅"
			}
			fmt.Fprintf(os.Stderr, "tag %s: %s -> %s\n", key, from, diff.To)
		}
	}
}
