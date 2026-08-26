package youtube

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
	"github.com/sebday/evoplayer/internal/config"
	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/tags"
)

type Options struct {
	MusicRoot   string
	MusicConfig string
}

func LoadOptions(env paths.Env) (Options, error) {
	return Options{
		MusicRoot:   env.MusicRoot,
		MusicConfig: env.MusicConfig,
	}, nil
}

func defaultGenre(musicConfig string) string {
	genre, err := config.Get(musicConfig, "download", "youtube_genre", "misc")
	if err != nil || genre == "" {
		return "misc"
	}
	return genre
}

func DownloadURL(env paths.Env, pageURL string) (string, error) {
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
	client := youtube.Client{HTTPClient: &http.Client{Timeout: 10 * time.Minute}}
	ctx := context.Background()
	video, err := client.GetVideoContext(ctx, pageURL)
	if err != nil {
		return "", fmt.Errorf("youtube: %w", err)
	}
	artist := strings.TrimSpace(video.Author)
	title := strings.TrimSpace(video.Title)
	if artist == "" {
		artist = "YouTube"
	}
	if title == "" {
		title = video.ID
	}
	dest := filepath.Join(incoming, tags.SanitizeFilenamePart(artist)+" - "+tags.SanitizeFilenamePart(title)+".mp3")
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	formats := video.Formats.Type("audio").WithAudioChannels()
	if len(formats) == 0 {
		return "", fmt.Errorf("youtube: no audio formats")
	}
	formats.Sort()
	format := formats[len(formats)-1]
	tmp := dest + ".raw"
	if err := downloadStream(ctx, client, video, format, tmp); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := ffmpegToMP3(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", err
	}
	os.Remove(tmp)

	year := tags.YearFromText(title)
	if year == 0 && !video.PublishDate.IsZero() {
		year = video.PublishDate.Year()
	}
	meta := map[string]string{
		"artist":  artist,
		"title":   title,
		"genre":   defaultGenre(opts.MusicConfig),
		"comment": "source:youtube",
	}
	if year > 0 {
		meta["year"] = fmt.Sprintf("%d", year)
	}
	picture, mime := fetchThumbnail(client.HTTPClient, video)
	if err := tags.EmbedMP3(dest, meta, picture, mime); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: youtube tag embed: %v\n", err)
	}
	return dest, nil
}

func downloadStream(ctx context.Context, client youtube.Client, video *youtube.Video, format youtube.Format, dest string) error {
	stream, _, err := client.GetStreamContext(ctx, video, &format)
	if err != nil {
		return err
	}
	defer stream.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, stream)
	closeErr := f.Close()
	if err != nil {
		os.Remove(dest)
		return err
	}
	return closeErr
}

func ffmpegToMP3(src, dest string) error {
	return exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", src, "-codec:a", "libmp3lame", "-q:a", "0", dest).Run()
}

func fetchThumbnail(client *http.Client, video *youtube.Video) ([]byte, string) {
	if client == nil {
		client = http.DefaultClient
	}
	urls := []string{
		fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", video.ID),
		fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", video.ID),
	}
	for _, thumb := range video.Thumbnails {
		if thumb.URL != "" {
			urls = append(urls, thumb.URL)
		}
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
