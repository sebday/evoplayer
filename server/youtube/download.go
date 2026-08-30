package youtube

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/tags"
)

type ProgressFunc func(phase string, percent int)

type Options struct {
	MusicRoot   string
	MusicConfig string
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
	}, nil
}

func defaultGenre(musicConfig string) string {
	genre, err := config.Get(musicConfig, "download", "youtube_genre", "")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(genre)
}

func DownloadURL(env paths.Env, pageURL string) (string, error) {
	return DownloadURLCtx(context.Background(), env, pageURL, nil)
}

func DownloadURLCtx(ctx context.Context, env paths.Env, pageURL string, progress ProgressFunc) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	bin, err := lookYtDlp()
	if err != nil {
		return "", fmt.Errorf("youtube: yt-dlp is required")
	}

	report("metadata", 0)
	info, browser, err := ytdlpDump(ctx, bin, pageURL)
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
	dest := filepath.Join(incoming, tags.SanitizeFilenamePart(artist)+" - "+tags.SanitizeFilenamePart(title)+".mp3")
	if _, err := os.Stat(dest); err == nil {
		report("download", 100)
		return dest, nil
	}

	tmpDir, err := os.MkdirTemp("", "evoplayer-yt-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	report("download", 0)
	raw, err := ytdlpFetch(ctx, bin, pageURL, tmpDir, browser, progress)
	if err != nil {
		return "", err
	}
	report("convert", 0)
	if err := ffmpegToMP3(ctx, raw, dest, info.Duration, progress); err != nil {
		os.Remove(dest)
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
	report("tag", 0)
	picture, mime := fetchThumbnail(info)
	if err := tags.EmbedMP3(dest, meta, picture, mime); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: youtube tag embed: %v\n", err)
	}
	report("tag", 100)
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

var lookYtDlp = func() (string, error) {
	return exec.LookPath("yt-dlp")
}

func ytdlpDump(ctx context.Context, bin, pageURL string) (ytdlpInfo, string, error) {
	var last error
	for _, browser := range cookieBrowsers() {
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

func ytdlpFetch(ctx context.Context, bin, pageURL, tmpDir, prefer string, progress ProgressFunc) (string, error) {
	outTmpl := filepath.Join(tmpDir, "audio.%(ext)s")
	var last error
	for _, browser := range cookieBrowserOrder(prefer) {
		args := append(ytdlpBaseArgs(browser), "-f", "bestaudio/best", "--newline", "-o", outTmpl, pageURL)
		if err := ytDlpRun(ctx, bin, args, func(line string) {
			if pct, ok := parseYtDlpPercent(line); ok && progress != nil {
				progress("download", pct)
			}
		}); err != nil {
			last = err
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
	out, err := ytDlpOutput(ctx, bin, append(ytdlpBaseArgs(browser), "-J", "--skip-download", pageURL))
	if err != nil {
		return ytdlpInfo{}, err
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
	args := []string{"--no-playlist", "--no-warnings"}
	if browser != "" {
		args = append(args, "--cookies-from-browser", browser)
	}
	return args
}

func cookieBrowsers() []string {
	return []string{"", "brave", "chromium"}
}

func cookieBrowserOrder(prefer string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 3)
	add := func(b string) {
		if seen[b] {
			return
		}
		seen[b] = true
		out = append(out, b)
	}
	add(prefer)
	for _, b := range cookieBrowsers() {
		add(b)
	}
	return out
}

func ytDlpOutput(ctx context.Context, bin string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	msg := strings.TrimSpace(err.Error())
	if exit, ok := err.(*exec.ExitError); ok {
		if stderr := strings.TrimSpace(string(exit.Stderr)); stderr != "" {
			msg = lastNonEmptyLine(stderr)
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, fmt.Errorf("youtube: %s", msg)
}

func ytDlpRun(ctx context.Context, bin string, args []string, onLine func(string)) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(stderr)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if onLine != nil {
			onLine(sc.Text())
		}
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("youtube: %s", err)
	}
	return nil
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

func ffmpegToMP3(ctx context.Context, src, dest string, duration float64, progress ProgressFunc) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", src, "-codec:a", "libmp3lame", "-q:a", "0",
		"-progress", "pipe:1", "-nostats", dest)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "out_time_ms=") || progress == nil || duration <= 0 {
			continue
		}
		ms, err := strconv.ParseFloat(strings.TrimPrefix(line, "out_time_ms="), 64)
		if err != nil || ms <= 0 {
			continue
		}
		pct := int(ms / (duration * 1000) * 100)
		if pct > 100 {
			pct = 100
		}
		if pct < 0 {
			pct = 0
		}
		progress("convert", pct)
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		os.Remove(dest)
		return ctx.Err()
	}
	return err
}

func lastNonEmptyLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" {
			return t
		}
	}
	return strings.TrimSpace(s)
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
		return body, strings.TrimSpace(mime)
	}
	return nil, ""
}
