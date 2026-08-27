package soundcloud

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/internal/audio"
	"github.com/sebday/evoplayer/internal/config"
	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/secrets"
	"github.com/sebday/evoplayer/internal/tags"
)

const defaultUser = "seb-day"

type DownloadOptions struct {
	User        string
	OAuthToken  string
	OAuthSource string
	ClientID    string
	MusicRoot   string
	StateDir    string
	ArchivePath string
}

func LikesURL(user string) string {
	user = strings.TrimSpace(user)
	if user == "" {
		user = defaultUser
	}
	return fmt.Sprintf("https://soundcloud.com/%s/likes", user)
}

func ArchivePath(stateDir string) string {
	return filepath.Join(stateDir, "sync-archive.txt")
}

func LoadOptions(env paths.Env) (DownloadOptions, error) {
	user, err := config.Get(env.MusicConfig, "soundcloud", "user", defaultUser)
	if err != nil {
		return DownloadOptions{}, err
	}
	clientID, err := config.Get(env.MusicConfig, "soundcloud", "client_id", "")
	if err != nil {
		return DownloadOptions{}, err
	}
	tok := secrets.SoundcloudOAuth()
	return DownloadOptions{
		User:        user,
		OAuthToken:  tok.Token,
		OAuthSource: tok.Source,
		ClientID:    clientID,
		MusicRoot:   env.MusicRoot,
		StateDir:    env.StateDir,
		ArchivePath: ArchivePath(env.StateDir),
	}, nil
}

func Download(opts DownloadOptions) error {
	if opts.MusicRoot == "" {
		return fmt.Errorf("evoplayer: music root not configured")
	}
	if opts.OAuthSource != "" {
		fmt.Fprintf(os.Stderr, "evoplayer: soundcloud auth from %s\n", opts.OAuthSource)
	}
	incoming := filepath.Join(opts.MusicRoot, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(opts.StateDir, 0o755); err != nil {
		return err
	}
	client := NewClient(opts.ClientID, opts.OAuthToken)
	archive, err := LoadArchive(opts.ArchivePath)
	if err != nil {
		return err
	}
	tracks, err := client.LikesTracks()
	if err != nil {
		return err
	}
	for _, track := range tracks {
		if archive.Has(track.ID) {
			continue
		}
		dest := trackDestPath(incoming, track)
		if err := downloadTrack(client, &track, dest); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: soundcloud download failed %d: %v\n", track.ID, err)
			continue
		}
		if err := archive.Add(track.ID); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: archive write: %v\n", err)
		}
		fmt.Fprintf(os.Stderr, "evoplayer: downloaded: %s\n", filepath.Base(dest))
	}
	return NormalizeIncoming(opts.MusicRoot)
}

func DownloadTrackURL(env paths.Env, pageURL string) (string, error) {
	opts, err := LoadOptions(env)
	if err != nil {
		return "", err
	}
	if opts.MusicRoot == "" {
		return "", fmt.Errorf("evoplayer: music root not configured")
	}
	if opts.OAuthSource != "" {
		fmt.Fprintf(os.Stderr, "evoplayer: soundcloud auth from %s\n", opts.OAuthSource)
	}
	incoming := filepath.Join(opts.MusicRoot, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return "", err
	}
	client := NewClient(opts.ClientID, opts.OAuthToken)
	track, err := client.ResolveURL(pageURL)
	if err != nil {
		return "", err
	}
	dest := trackDestPath(incoming, *track)
	if err := downloadTrack(client, track, dest); err != nil {
		return "", err
	}
	if err := NormalizeIncoming(opts.MusicRoot); err != nil {
		return "", err
	}
	return dest, nil
}

func trackDestPath(incoming string, track Track) string {
	artist := tags.SanitizeFilenamePart(track.User.Username)
	title := tags.SanitizeFilenamePart(track.Title)
	return filepath.Join(incoming, artist+" - "+title+".mp3")
}

func downloadTrack(client *Client, track *Track, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	raw := dest + ".raw"
	if err := client.DownloadTrackStream(track, raw); err != nil {
		os.Remove(raw)
		return err
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(raw), "."))
	if ext != "mp3" {
		if err := ffmpegToMP3(raw, dest); err != nil {
			os.Remove(raw)
			return err
		}
		os.Remove(raw)
	} else if err := os.Rename(raw, dest); err != nil {
		os.Remove(raw)
		return err
	}
	meta := trackMeta(track)
	var picture []byte
	var mime string
	artURL := tags.ArtworkURLLarge(track.ArtworkURL)
	if artURL != "" {
		if data, m, err := fetchBytes(client.HTTP, artURL); err == nil && len(data) > 0 {
			picture = data
			mime = m
			if mime == "" {
				mime = tags.PictureMIME(data)
			}
		}
	}
	if err := tags.EmbedMP3(dest, meta, picture, mime); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: tag embed failed: %v\n", err)
	}
	return nil
}

func trackMeta(track *Track) map[string]string {
	artist := strings.TrimSpace(track.User.Username)
	title := strings.TrimSpace(track.Title)
	year := ""
	if y := tags.YearFromText(title); y > 0 {
		year = fmt.Sprintf("%d", y)
	} else if track.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, track.CreatedAt); err == nil {
			year = fmt.Sprintf("%d", t.Year())
		}
	}
	meta := map[string]string{
		"artist":  artist,
		"title":   title,
		"comment": "source:soundcloud",
	}
	if year != "" {
		meta["year"] = year
	}
	return meta
}

func ffmpegToMP3(src, dest string) error {
	return exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", src, "-codec:a", "libmp3lame", "-q:a", "0", dest).Run()
}

func NormalizeIncoming(musicRoot string) error {
	incoming := filepath.Join(musicRoot, ".incoming")
	entries, err := os.ReadDir(incoming)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		f := filepath.Join(incoming, e.Name())
		base := strings.TrimSuffix(f, filepath.Ext(f))
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(f), "."))
		switch ext {
		case "mp3":
			continue
		case "part", "ytdl", "temp", "raw", "jpg", "jpeg", "png", "webp", "gif":
			mp3 := base + ".mp3"
			if _, err := os.Stat(mp3); err == nil {
				_ = os.Remove(f)
			}
			continue
		}
		if !audio.IsAudio(f) {
			continue
		}
		mp3 := base + ".mp3"
		if _, err := os.Stat(mp3); err == nil {
			_ = os.Remove(f)
			continue
		}
		if err := ffmpegToMP3(f, mp3); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: mp3 convert failed: %s\n", f)
			continue
		}
		_ = os.Remove(f)
		fmt.Fprintf(os.Stderr, "evoplayer: converted to mp3: %s\n", filepath.Base(mp3))
	}
	return nil
}

func DownloadEnv(env paths.Env) error {
	opts, err := LoadOptions(env)
	if err != nil {
		return err
	}
	return Download(opts)
}
