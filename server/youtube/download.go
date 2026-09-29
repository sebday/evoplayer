package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/syncarchive"
	"github.com/sebday/evoplayer/server/tags"
	"github.com/sebday/evoplayer/server/ytdlp"
)

type ProgressFunc func(phase string, percent int)

type Options struct {
	MusicRoot   string
	MusicConfig string
	StateDir    string
}

type ytdlpInfo struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Uploader   string  `json:"uploader"`
	Channel    string  `json:"channel"`
	Artist     string  `json:"artist"`
	UploadDate string  `json:"upload_date"`
	Thumbnail  string  `json:"thumbnail"`
	Duration   float64 `json:"duration"`
}

func LoadOptions(env paths.Env) (Options, error) {
	return Options{
		MusicRoot:   env.MusicRoot,
		MusicConfig: env.MusicConfig,
		StateDir:    env.StateDir,
	}, nil
}

// VideoID extracts a YouTube video id from a supported URL.
func VideoID(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	switch host {
	case "youtu.be":
		return strings.Trim(u.Path, "/")
	case "youtube.com", "m.youtube.com", "music.youtube.com":
		if id := strings.TrimSpace(u.Query().Get("v")); id != "" {
			return id
		}
	}
	return ""
}

func defaultGenre(musicConfig string) string {
	genre, err := config.Get(musicConfig, "download", "youtube_genre", "")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(genre)
}

func DownloadURLCtx(ctx context.Context, env paths.Env, pageURL string, progress ProgressFunc) (string, error) {
	report := func(phase string, pct int) {
		if progress != nil {
			progress(phase, pct)
		}
	}
	opts, err := LoadOptions(env)
	if err != nil {
		return "", err
	}
	if opts.MusicRoot == "" {
		return "", fmt.Errorf("evoplayer: music root not configured")
	}
	incoming := filepath.Join(opts.MusicRoot, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return "", err
	}
	bin, err := ytdlp.Bin()
	if err != nil {
		return "", fmt.Errorf("youtube: yt-dlp is required")
	}

	report("reading metadata", 0)
	info, browser, err := ytdlpDump(ctx, bin, pageURL)
	if err != nil {
		return "", err
	}
	archive, err := syncarchive.Load(syncarchive.Path(opts.StateDir))
	if err != nil {
		return "", err
	}
	artist := info.artist()
	title := strings.TrimSpace(info.Title)
	if artist == "" {
		artist = "YouTube"
	}
	if title == "" {
		title = info.ID
	}
	label := trackLabel(artist, title)
	if archive.HasYT(info.ID) {
		report(stage("already downloaded", label), 100)
		return "", nil
	}
	dest := filepath.Join(incoming, tags.SanitizeFilenamePart(artist)+" - "+tags.SanitizeFilenamePart(title)+".mp3")
	if _, err := os.Stat(dest); err == nil {
		report(stage("already downloaded", label), 100)
		return dest, nil
	}

	tmpDir, err := os.MkdirTemp("", "evoplayer-yt-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	report(stage("downloading", label), 0)
	raw, err := ytdlpFetch(ctx, bin, pageURL, tmpDir, browser, label, progress)
	if err != nil {
		return "", err
	}
	report(stage("converting", label), 0)
	var onConvert func(float64)
	if progress != nil && info.Duration > 0 {
		onConvert = func(sec float64) {
			pct := min(100, int(sec/info.Duration*100))
			progress(stage(fmt.Sprintf("converting %d%%", pct), label), pct)
		}
	}
	if err := ytdlp.ToMP3(ctx, raw, dest, onConvert); err != nil {
		return "", fmt.Errorf("youtube: convert to mp3: %w", err)
	}

	year := tags.YearFromText(title)
	if year == 0 && len(info.UploadDate) >= 4 {
		if y, err := strconv.Atoi(info.UploadDate[:4]); err == nil {
			year = y
		}
	}
	meta := map[string]string{
		"artist":  artist,
		"title":   title,
		"comment": "source:youtube",
	}
	if genre := defaultGenre(opts.MusicConfig); genre != "" {
		meta["genre"] = genre
	}
	if year > 0 {
		meta["year"] = fmt.Sprintf("%d", year)
	}
	if info.Duration > 0 {
		meta["duration_ms"] = fmt.Sprintf("%.0f", info.Duration*1000)
	}
	report(stage("tagging", label), 0)
	picture, mime := fetchThumbnail(info)
	if err := tags.EmbedMP3(dest, meta, picture, mime); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: youtube tag embed: %v\n", err)
	}
	report(stage("tagging", label), 100)
	if err := archive.AddYT(info.ID); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: archive write: %v\n", err)
	}
	return dest, nil
}

func (info ytdlpInfo) artist() string {
	for _, v := range []string{info.Artist, info.Uploader, info.Channel} {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func ytdlpDump(ctx context.Context, bin, pageURL string) (ytdlpInfo, string, error) {
	var last error
	for _, browser := range ytdlp.Browsers() {
		info, err := ytdlpDumpOnce(ctx, bin, pageURL, browser)
		if err == nil {
			return info, browser, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("youtube: yt-dlp failed")
	}
	return ytdlpInfo{}, "", last
}

func ytdlpFetch(ctx context.Context, bin, pageURL, tmpDir, prefer, label string, progress ProgressFunc) (string, error) {
	outTmpl := filepath.Join(tmpDir, "audio.%(ext)s")
	var last error
	for _, browser := range ytdlp.Browsers(prefer) {
		args := append(ytdlpBaseArgs(browser), "-f", "bestaudio/best", "--newline", "-o", outTmpl, "--", pageURL)
		if err := ytdlp.Run(ctx, bin, args, func(line string) {
			if detail, pct, ok := ytDownloadDetail(line, label); ok && progress != nil {
				progress(detail, pct)
			}
		}); err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			last = fmt.Errorf("youtube: %w", err)
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(tmpDir, "audio.*"))
		if len(matches) == 0 {
			last = fmt.Errorf("youtube: yt-dlp wrote no audio")
			continue
		}
		return matches[0], nil
	}
	if last == nil {
		last = fmt.Errorf("youtube: yt-dlp failed")
	}
	return "", last
}

func ytdlpDumpOnce(ctx context.Context, bin, pageURL, browser string) (ytdlpInfo, error) {
	out, err := ytdlp.Output(ctx, bin, append(ytdlpBaseArgs(browser), "-J", "--skip-download", "--", pageURL))
	if err != nil {
		return ytdlpInfo{}, fmt.Errorf("youtube: %w", err)
	}
	var info ytdlpInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return ytdlpInfo{}, fmt.Errorf("youtube: parse yt-dlp metadata: %w", err)
	}
	if strings.TrimSpace(info.ID) == "" && strings.TrimSpace(info.Title) == "" {
		return ytdlpInfo{}, fmt.Errorf("youtube: empty yt-dlp metadata")
	}
	return info, nil
}

func ytdlpBaseArgs(browser string) []string {
	return append([]string{"--no-playlist", "--no-warnings"}, ytdlp.CookieArgs(browser)...)
}

func trackLabel(artist, title string) string {
	s := strings.TrimSpace(artist)
	t := strings.TrimSpace(title)
	switch {
	case s != "" && t != "":
		s = s + " — " + t
	case t != "":
		s = t
	}
	r := []rune(s)
	if len(r) > 80 {
		s = string(r[:79]) + "…"
	}
	return s
}

func stage(verb, label string) string {
	if label == "" {
		return verb
	}
	return verb + " · " + label
}

var ytProgressRE = regexp.MustCompile(`(\d+(?:\.\d+)?)%\s+of\s+~?\s*(\S+)(?:\s+at\s+(\S+))?(?:\s+ETA\s+(\S+))?`)

func ytDownloadDetail(line, label string) (string, int, bool) {
	if !strings.Contains(line, "[download]") {
		return "", 0, false
	}
	pct, ok := parseYtDlpPercent(line)
	if !ok {
		return "", 0, false
	}
	detail := fmt.Sprintf("downloading %d%%", pct)
	if m := ytProgressRE.FindStringSubmatch(line); m != nil {
		if m[2] != "" {
			detail += " of " + m[2]
		}
		if m[3] != "" {
			detail += " at " + m[3]
		}
		if eta := m[4]; eta != "" && !strings.EqualFold(eta, "Unknown") {
			detail += ", " + eta + " left"
		}
	}
	return stage(detail, label), pct, true
}

func parseYtDlpPercent(line string) (int, bool) {
	i := strings.LastIndex(line, "%")
	if i < 1 {
		return 0, false
	}
	start := i - 1
	for start >= 0 && (line[start] == '.' || line[start] >= '0' && line[start] <= '9') {
		start--
	}
	start++
	if start >= i {
		return 0, false
	}
	pct, err := strconv.ParseFloat(line[start:i], 64)
	if err != nil {
		return 0, false
	}
	n := int(pct)
	if n < 0 {
		n = 0
	}
	if n > 100 {
		n = 100
	}
	return n, true
}

func fetchThumbnail(info ytdlpInfo) ([]byte, string) {
	client := &http.Client{Timeout: 20 * time.Second}
	urls := []string{}
	if info.ID != "" {
		urls = append(urls,
			fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", info.ID),
			fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", info.ID),
		)
	}
	if info.Thumbnail != "" {
		urls = append(urls, info.Thumbnail)
	}
	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
		resp.Body.Close()
		if err != nil || len(body) < 1024 {
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}
		mime := resp.Header.Get("Content-Type")
		if i := strings.Index(mime, ";"); i > 0 {
			mime = mime[:i]
		}
		if mime == "" {
			mime = tags.PictureMIME(body)
		}
		body, mime = squareCover(body)
		return body, strings.TrimSpace(mime)
	}
	return nil, ""
}
