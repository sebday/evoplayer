package soundcloud

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/secrets"
	"github.com/sebday/evoplayer/server/tags"
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
	return DownloadReport(opts, nopReporter{})
}

func DownloadReport(opts DownloadOptions, rep Reporter) error {
	if rep == nil {
		rep = nopReporter{}
	}
	if opts.MusicRoot == "" {
		return fmt.Errorf("evoplayer: music root not configured")
	}
	if opts.OAuthSource != "" {
		msg := LogInfof("soundcloud auth from %s", opts.OAuthSource)
		fmt.Fprintf(os.Stderr, "evoplayer: %s\n", msg)
		rep.Line(msg)
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
	rep.Progress(jobs.Progress{Phase: "fetching likes"})
	rep.Line(LogInfo("fetching likes"))
	tracks, err := client.LikesTracksProgress(func(n int) {
		rep.Progress(jobs.Progress{Phase: fmt.Sprintf("fetching likes (%d)", n), Done: n})
		if n > 0 && n%200 == 0 {
			rep.Line(LogInfof("fetched %d likes", n))
		}
	})
	if err != nil {
		rep.Line(LogFail(err.Error()))
		return err
	}
	rep.Line(LogInfof("%d likes", len(tracks)))
	pending := make([]Track, 0, len(tracks))
	for _, track := range tracks {
		if !archive.Has(track.ID) {
			pending = append(pending, track)
		}
	}
	archived := len(tracks) - len(pending)
	rep.Line(LogInfof("%d already archived", archived))
	total := len(pending)
	if total == 0 {
		rep.Line(LogInfo("no new likes to download"))
		return NormalizeIncoming(opts.MusicRoot)
	}
	rep.Line(LogInfof("%d tracks to download", total))
	done := 0
	for _, track := range pending {
		label := strings.TrimSpace(track.User.Username)
		if title := strings.TrimSpace(track.Title); title != "" {
			if label != "" {
				label += " - " + title
			} else {
				label = title
			}
		}
		rep.Progress(jobs.Progress{Phase: label, Done: done, Total: total})
		dest := trackDestPath(incoming, track)
		if _, err := os.Stat(dest); err == nil {
			if err := archive.Add(track.ID); err != nil {
				fmt.Fprintf(os.Stderr, "evoplayer: warn: archive write: %v\n", err)
				rep.Line(LogWarn(fmt.Sprintf("archive write failed: %v", err)))
			}
			done++
			msg := LogSkip(filepath.Base(dest))
			fmt.Fprintf(os.Stderr, "evoplayer: %s\n", msg)
			rep.Line(msg)
			rep.Progress(jobs.Progress{Phase: msg, Done: done, Total: total})
			continue
		}
		if err := downloadTrack(client, &track, dest); err != nil {
			var msg string
			if isDRMError(err) {
				msg = LogSkip(label + " (drm)")
			} else {
				msg = LogFail(fmt.Sprintf("%s (%v)", label, err))
			}
			fmt.Fprintf(os.Stderr, "evoplayer: warn: soundcloud download failed %d: %v\n", track.ID, err)
			rep.Line(msg)
			continue
		}
		if err := archive.Add(track.ID); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: archive write: %v\n", err)
			rep.Line(LogWarn(fmt.Sprintf("archive write failed: %v", err)))
		}
		done++
		msg := LogOK(filepath.Base(dest))
		fmt.Fprintf(os.Stderr, "evoplayer: %s\n", msg)
		rep.Line(msg)
		rep.Progress(jobs.Progress{Phase: msg, Done: done, Total: total})
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
	client := NewClient(opts.ClientID, "")
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
	err := client.DownloadTrackStream(track, dest)
	if err != nil {
		os.Remove(dest)
		if pageURL := strings.TrimSpace(track.PermalinkURL); pageURL != "" {
			if yerr := downloadYtDlp(pageURL, dest, client.OAuthToken, client.ClientID); yerr == nil {
				err = nil
			} else if isDRMError(yerr) || isDRMError(err) {
				return fmt.Errorf("drm protected")
			}
		}
		if err != nil {
			if isDRMError(err) {
				return fmt.Errorf("drm protected")
			}
			return err
		}
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

func DownloadEnvReport(env paths.Env, rep Reporter) error {
	opts, err := LoadOptions(env)
	if err != nil {
		return err
	}
	return DownloadReport(opts, rep)
}
