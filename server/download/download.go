package download

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/soundcloud"
	"github.com/sebday/evoplayer/server/syncarchive"
	"github.com/sebday/evoplayer/server/youtube"
)

const (
	KindYouTube    = "youtube"
	KindSCTrack    = "sc-track"
	KindSCLikes    = "sc-likes"
	KindSCPlaylist = "sc-playlist"
	KindSCArtist   = "sc-artist"
)

// ClassifyURL returns the download kind for a YouTube or SoundCloud URL.
func ClassifyURL(rawURL string) string {
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
	case "youtube.com", "m.youtube.com", "music.youtube.com":
		path := strings.Trim(u.Path, "/")
		if path == "watch" || strings.HasPrefix(path, "watch") {
			return KindYouTube
		}
	case "youtu.be":
		if strings.Trim(u.Path, "/") != "" {
			return KindYouTube
		}
	case "soundcloud.com", "on.soundcloud.com":
		return classifySoundCloudPath(strings.Trim(u.Path, "/"))
	}
	return ""
}

func classifySoundCloudPath(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	switch {
	case len(parts) == 2 && parts[0] == "you" && parts[1] == "likes":
		return KindSCLikes
	case len(parts) == 2 && parts[1] == "likes":
		return KindSCPlaylist
	case len(parts) >= 3 && parts[1] == "sets":
		return KindSCPlaylist
	case len(parts) == 1:
		return KindSCArtist
	case len(parts) == 2 && parts[1] == "tracks":
		return KindSCArtist
	case len(parts) == 2:
		return KindSCTrack
	default:
		return KindSCTrack
	}
}

// NormalizeCollectionURL maps bare artist profile URLs to their uploads feed.
func NormalizeCollectionURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if ClassifyURL(rawURL) != KindSCArtist {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 1 {
		return rawURL
	}
	u.Path = parts[0] + "/tracks"
	return u.String()
}

// DownloadFromURLCtx downloads content from a supported URL. Multi-track kinds
// return an empty path; single-track kinds return the downloaded file path.
func DownloadFromURLCtx(ctx context.Context, env paths.Env, rawURL string, rep jobs.Reporter, progress youtube.ProgressFunc) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep == nil {
		rep = jobs.NopReporter
	}
	switch ClassifyURL(rawURL) {
	case KindYouTube:
		archive, err := syncarchive.Load(syncarchive.Path(env.StateDir))
		if err != nil {
			return "", err
		}
		if id := youtube.VideoID(rawURL); id != "" && archive.HasYT(id) {
			rep.Line(jobs.LogSkip(id + " (archived)"))
			return "", nil
		}
		path, err := youtube.DownloadURLCtx(ctx, env, rawURL, progress)
		if err != nil {
			return "", err
		}
		if path == "" {
			return "", nil
		}
		rep.Line(jobs.LogOK(filepathBase(path)))
		return path, nil
	case KindSCTrack:
		return soundcloud.DownloadTrackURLCtx(ctx, env, rawURL, rep)
	case KindSCLikes:
		if err := soundcloud.DownloadEnvReportCtx(ctx, env, rep); err != nil {
			return "", err
		}
		return "", nil
	case KindSCPlaylist, KindSCArtist:
		pageURL := NormalizeCollectionURL(rawURL)
		if pageURL != strings.TrimSpace(rawURL) {
			rep.Line(jobs.LogInfof("using uploads feed %s", pageURL))
		}
		if err := soundcloud.DownloadCollectionURLCtx(ctx, env, pageURL, rep); err != nil {
			return "", err
		}
		return "", nil
	default:
		return "", fmt.Errorf("evoplayer: unsupported download url (youtube or soundcloud only)")
	}
}

func filepathBase(path string) string {
	if path == "" {
		return ""
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
