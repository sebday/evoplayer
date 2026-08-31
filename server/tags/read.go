package tags

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	id3v2 "github.com/bogem/id3v2"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var sanitizeRe = regexp.MustCompile(`[/\\:*?"<>|]+`)
var spaceRe = regexp.MustCompile(`\s+`)
var slugNonAlnum = regexp.MustCompile(`[^a-z0-9$]+`)
var slugUnders = regexp.MustCompile(`_+`)
var yearInTag = regexp.MustCompile(`(\d{4})`)

type TagInfo struct {
	Title         string `json:"title"`
	Artist        string `json:"artist"`
	Genre         string `json:"genre"`
	Album         string `json:"album"`
	AlbumArtist   string `json:"album_artist"`
	Year          string `json:"year"`
	Label         string `json:"label"`
	CatalogNumber string `json:"catalog_number"`
	Comment       string `json:"comment"`
}

type ProbeResult struct {
	Tag      TagInfo
	Duration float64
}

func ReadTags(path string) (TagInfo, error) {
	res, err := Probe(path)
	return res.Tag, err
}

// Probe reads tags and duration in one pass. MP3 uses native ID3 (optional
// duration-only ffprobe); other formats use a single ffprobe -show_format.
func Probe(path string) (ProbeResult, error) {
	return probe(path, true)
}

// ProbeImport reads tags for bulk .incoming import. MP3 duration comes from
// ID3 TLEN only (no per-file ffprobe); mix routing still uses path heuristics.
func ProbeImport(path string) (ProbeResult, error) {
	return probe(path, false)
}

func probe(path string, ffprobeMP3Duration bool) (ProbeResult, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3", ".mp2":
		info, dur := readID3(path)
		fillFilenameFallback(&info, path)
		if ffprobeMP3Duration && dur <= 0 {
			dur = ffprobeDuration(path)
		}
		return ProbeResult{Tag: info, Duration: dur}, nil
	default:
		info, dur := readFFProbe(path)
		fillFilenameFallback(&info, path)
		return ProbeResult{Tag: info, Duration: dur}, nil
	}
}

func readID3(path string) (TagInfo, float64) {
	info := TagInfo{}
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return info, 0
	}
	defer tag.Close()
	info.Title = strings.TrimSpace(tag.Title())
	info.Artist = strings.TrimSpace(firstNonEmpty(tag.Artist(), textFrame(tag, "TPE2")))
	info.AlbumArtist = strings.TrimSpace(textFrame(tag, "TPE2"))
	info.Genre = strings.TrimSpace(tag.Genre())
	info.Album = strings.TrimSpace(tag.Album())
	info.Label = strings.TrimSpace(textFrame(tag, "TPUB"))
	info.Year = yearFromText(firstNonEmpty(textFrame(tag, "TDRC"), textFrame(tag, "TYER")))
	if comm, ok := tag.GetLastFrame("COMM").(id3v2.CommentFrame); ok {
		info.Comment = strings.TrimSpace(comm.Text)
	}
	for _, frame := range tag.GetFrames("TXXX") {
		if user, ok := frame.(id3v2.UserDefinedTextFrame); ok {
			desc := strings.ToLower(user.Description)
			if desc == "catalognumber" || desc == "catalog" || desc == "catalogue" {
				info.CatalogNumber = strings.TrimSpace(user.Value)
				break
			}
		}
	}
	dur := 0.0
	if tlen := strings.TrimSpace(textFrame(tag, "TLEN")); tlen != "" {
		if ms, err := strconv.ParseFloat(tlen, 64); err == nil && ms > 0 {
			dur = ms / 1000
		}
	}
	return info, dur
}

func readFFProbe(path string) (TagInfo, float64) {
	info := TagInfo{}
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", path)
	raw, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return info, 0
	}
	var payload struct {
		Format struct {
			Duration string            `json:"duration"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return info, 0
	}
	lookup := map[string]string{}
	for k, v := range payload.Format.Tags {
		if strings.TrimSpace(v) != "" {
			lookup[strings.ToLower(k)] = strings.TrimSpace(v)
		}
	}
	pick := func(keys ...string) string {
		for _, key := range keys {
			if val := lookup[strings.ToLower(key)]; val != "" {
				return val
			}
		}
		return ""
	}
	info.Title = pick("title")
	info.Artist = pick("artist", "album_artist", "albumartist")
	info.Genre = pick("genre")
	info.Album = pick("album")
	info.AlbumArtist = pick("album_artist", "albumartist")
	info.Label = pick("label", "publisher", "organization", "tpub")
	info.CatalogNumber = pick("catalognumber", "catalog", "catalogue", "catno")
	info.Comment = pick("comment", "description")
	year := pick("date", "year", "originaldate", "original_year", "tyer")
	if m := yearInTag.FindStringSubmatch(year); len(m) > 1 {
		info.Year = m[1]
	}
	dur := 0.0
	if payload.Format.Duration != "" {
		dur, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	}
	return info, dur
}

// MediaDuration returns audio length in seconds (ffprobe), or 0 when unknown.
func MediaDuration(path string) float64 {
	return ffprobeDuration(path)
}

func ffprobeDuration(path string) float64 {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", path)
	raw, err := cmd.Output()
	if err != nil {
		return 0
	}
	dur, _ := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	return dur
}

func fillFilenameFallback(info *TagInfo, path string) {
	if info.Title == "" {
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		artist, title := ParseFilenameArtistTitle(stem)
		if title != "" {
			info.Title = title
		}
		if artist != "" {
			info.Artist = artist
		}
		if info.Title == "" {
			info.Title = stem
		}
	} else if info.Artist == "" {
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if artist, _ := ParseFilenameArtistTitle(stem); artist != "" {
			info.Artist = artist
		}
	}
}

func SanitizeFilenamePart(s string) string {
	s = sanitizeRe.ReplaceAllString(s, " ")
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func Slugify(value string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	value, _, _ = transform.String(t, value)
	value = strings.ToLower(value)
	value = slugNonAlnum.ReplaceAllString(value, "_")
	value = slugUnders.ReplaceAllString(value, "_")
	return strings.Trim(value, "_")
}

func ReadTagsJSON(path string) ([]byte, error) {
	info, err := ReadTags(path)
	if err != nil {
		return nil, err
	}
	return json.Marshal(info)
}
