package soundcloud

import (
	"context"
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
	"github.com/sebday/evoplayer/server/syncarchive"
	"github.com/sebday/evoplayer/server/tags"
)

const defaultUser = "seb-day"

type DownloadOptions struct {
	User        string
	OAuthToken  string
	OAuthSource string
	ClientID    string
	MusicRoot   string
	MusicConfig string
	StateDir    string
	ArchivePath string
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
		MusicConfig: env.MusicConfig,
		StateDir:    env.StateDir,
		ArchivePath: syncarchive.Path(env.StateDir),
	}, nil
}

func DownloadReportCtx(ctx context.Context, opts DownloadOptions, rep jobs.Reporter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep == nil {
		rep = jobs.NopReporter
	}
	if opts.MusicRoot == "" {
		return fmt.Errorf("evoplayer: music root not configured")
	}
	if opts.OAuthSource != "" {
		msg := jobs.LogInfof("soundcloud auth from %s", opts.OAuthSource)
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
	archive, err := syncarchive.Load(opts.ArchivePath)
	if err != nil {
		return err
	}
	rep.Progress(jobs.Progress{Phase: "fetching likes"})
	rep.Line(jobs.LogInfo("fetching likes"))
	tracks, err := client.LikesTracksProgressCtx(ctx, func(n int) {
		rep.Progress(jobs.Progress{Phase: fmt.Sprintf("fetching likes (%d)", n), Done: n})
		if n > 0 && n%200 == 0 {
			rep.Line(jobs.LogInfof("fetched %d likes", n))
		}
	})
	if err != nil {
		rep.Line(jobs.LogFail(err.Error()))
		return err
	}
	rep.Line(jobs.LogInfof("%d likes", len(tracks)))
	pending := make([]Track, 0, len(tracks))
	for _, track := range tracks {
		if !archive.HasSC(track.ID) {
			pending = append(pending, track)
		}
	}
	archived := len(tracks) - len(pending)
	rep.Line(jobs.LogInfof("%d already archived", archived))
	total := len(pending)
	if total == 0 {
		rep.Line(jobs.LogInfo("no new likes to download"))
		return NormalizeIncoming(ctx, opts.MusicRoot)
	}
	rep.Line(jobs.LogInfof("%d tracks to download", total))
	done := 0
	for _, track := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}
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
			archiveSC(archive, track.ID, rep)
			done++
			msg := jobs.LogSkip(filepath.Base(dest))
			fmt.Fprintf(os.Stderr, "evoplayer: %s\n", msg)
			rep.Line(msg)
			rep.Progress(jobs.Progress{Phase: msg, Done: done, Total: total})
			continue
		}
		if err := downloadTrack(ctx, client, &track, dest, opts); err != nil {
			var msg string
			if isDRMError(err) {
				archiveSC(archive, track.ID, rep)
				done++
				msg = jobs.LogSkip(label + " (drm)")
			} else {
				msg = jobs.LogFail(fmt.Sprintf("%s (%v)", label, err))
			}
			fmt.Fprintf(os.Stderr, "evoplayer: warn: soundcloud download failed %d: %v\n", track.ID, err)
			rep.Line(msg)
			if isDRMError(err) {
				rep.Progress(jobs.Progress{Phase: msg, Done: done, Total: total})
			}
			continue
		}
		archiveSC(archive, track.ID, rep)
		done++
		msg := jobs.LogOK(filepath.Base(dest))
		fmt.Fprintf(os.Stderr, "evoplayer: %s\n", msg)
		rep.Line(msg)
		rep.Progress(jobs.Progress{Phase: msg, Done: done, Total: total})
	}
	return NormalizeIncoming(ctx, opts.MusicRoot)
}

func DownloadTrackURLCtx(ctx context.Context, env paths.Env, pageURL string, rep jobs.Reporter) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep == nil {
		rep = jobs.NopReporter
	}
	opts, err := LoadOptions(env)
	if err != nil {
		return "", err
	}
	if opts.MusicRoot == "" {
		return "", fmt.Errorf("evoplayer: music root not configured")
	}
	if opts.OAuthSource != "" {
		msg := jobs.LogInfof("soundcloud auth from %s", opts.OAuthSource)
		fmt.Fprintf(os.Stderr, "evoplayer: %s\n", msg)
		rep.Line(msg)
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
	archive, err := syncarchive.Load(opts.ArchivePath)
	if err != nil {
		return "", err
	}
	label := strings.TrimSpace(track.User.Username)
	if title := strings.TrimSpace(track.Title); title != "" {
		if label != "" {
			label += " - " + title
		} else {
			label = title
		}
	}
	if archive.HasSC(track.ID) {
		msg := jobs.LogSkip(label + " (archived)")
		rep.Line(msg)
		return "", nil
	}
	dest := trackDestPath(incoming, *track)
	rep.Progress(jobs.Progress{Phase: label, Done: 0, Total: 1})
	if err := downloadTrack(ctx, client, track, dest, opts); err != nil {
		if isDRMError(err) {
			archiveSC(archive, track.ID, rep)
			msg := jobs.LogSkip(label + " (drm)")
			rep.Line(msg)
			return "", nil
		}
		rep.Line(jobs.LogFail(fmt.Sprintf("%s (%v)", label, err)))
		return "", err
	}
	archiveSC(archive, track.ID, rep)
	if err := NormalizeIncoming(ctx, opts.MusicRoot); err != nil {
		return "", err
	}
	rep.Line(jobs.LogOK(filepath.Base(dest)))
	return dest, nil
}

func DownloadCollectionURLCtx(ctx context.Context, env paths.Env, pageURL string, rep jobs.Reporter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep == nil {
		rep = jobs.NopReporter
	}
	opts, err := LoadOptions(env)
	if err != nil {
		return err
	}
	if opts.MusicRoot == "" {
		return fmt.Errorf("evoplayer: music root not configured")
	}
	if opts.OAuthSource != "" {
		msg := jobs.LogInfof("soundcloud auth from %s", opts.OAuthSource)
		fmt.Fprintf(os.Stderr, "evoplayer: %s\n", msg)
		rep.Line(msg)
	}
	incoming := filepath.Join(opts.MusicRoot, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return err
	}
	rep.Progress(jobs.Progress{Phase: "downloading collection"})
	if strings.Contains(pageURL, "/likes") {
		rep.Line(jobs.LogInfo("downloading likes page"))
	} else {
		rep.Line(jobs.LogInfo("downloading collection"))
	}
	archive, err := syncarchive.Load(opts.ArchivePath)
	if err != nil {
		return err
	}
	added, err := downloadYtDlpCollection(ctx, pageURL, incoming, opts.OAuthToken, opts.ClientID, archive, rep)
	if err != nil {
		rep.Line(jobs.LogFail(err.Error()))
		return err
	}
	if len(added) == 0 {
		rep.Line(jobs.LogInfo("no new tracks downloaded"))
	}
	return NormalizeIncoming(ctx, opts.MusicRoot)
}

func archiveSC(archive *syncarchive.Archive, id int64, rep jobs.Reporter) {
	if archive == nil || id == 0 {
		return
	}
	if err := archive.AddSC(id); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: archive write: %v\n", err)
		if rep != nil {
			rep.Line(jobs.LogWarn(fmt.Sprintf("archive write failed: %v", err)))
		}
	}
}

func trackDestPath(incoming string, track Track) string {
	artist := tags.SanitizeFilenamePart(track.User.Username)
	title := tags.SanitizeFilenamePart(track.Title)
	return filepath.Join(incoming, artist+" - "+title+".mp3")
}

func downloadTrack(ctx context.Context, client *Client, track *Track, dest string, opts DownloadOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	err := client.DownloadTrackStream(ctx, track, dest)
	if err != nil {
		os.Remove(dest)
		if pageURL := strings.TrimSpace(track.PermalinkURL); pageURL != "" {
			if yerr := downloadYtDlp(ctx, pageURL, dest, client.OAuthToken, client.ClientID); yerr == nil {
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
	meta := trackMeta(track, opts)
	if dur := tags.MediaDuration(dest); dur > 0 {
		meta["duration_ms"] = fmt.Sprintf("%.0f", dur*1000)
	}
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

func trackMeta(track *Track, opts DownloadOptions) map[string]string {
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
	if genre := trackEmbedGenre(track, opts); genre != "" {
		meta["genre"] = genre
	}
	return meta
}

func ffmpegToMP3(ctx context.Context, src, dest string) error {
	return exec.CommandContext(ctx, "ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", src, "-codec:a", "libmp3lame", "-q:a", "0", dest).Run()
}

func NormalizeIncoming(ctx context.Context, musicRoot string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	incoming := filepath.Join(musicRoot, ".incoming")
	entries, err := os.ReadDir(incoming)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
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
		if err := ffmpegToMP3(ctx, f, mp3); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: mp3 convert failed: %s\n", f)
			continue
		}
		_ = os.Remove(f)
		fmt.Fprintf(os.Stderr, "evoplayer: converted to mp3: %s\n", filepath.Base(mp3))
	}
	return nil
}

func DownloadEnvReportCtx(ctx context.Context, env paths.Env, rep jobs.Reporter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	opts, err := LoadOptions(env)
	if err != nil {
		return err
	}
	return DownloadReportCtx(ctx, opts, rep)
}
