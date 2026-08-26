package download

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/soundcloud"
	"github.com/sebday/evoplayer/internal/youtube"
)

func DetectSource(rawURL string) string {
	rawURL = strings.TrimSpace(strings.ToLower(rawURL))
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.TrimPrefix(u.Host, "www.")
	switch host {
	case "youtube.com", "m.youtube.com", "music.youtube.com":
		if strings.Trim(u.Path, "/") == "watch" || strings.HasPrefix(u.Path, "/watch") {
			return "youtube"
		}
	case "youtu.be":
		return "youtube"
	case "soundcloud.com":
		return "soundcloud"
	}
	return ""
}

func DownloadURL(env paths.Env, rawURL string) (string, error) {
	switch DetectSource(rawURL) {
	case "youtube":
		return youtube.DownloadURL(env, rawURL)
	case "soundcloud":
		return soundcloud.DownloadTrackURL(env, rawURL)
	default:
		return "", fmt.Errorf("evoplayer: unsupported download url (youtube or soundcloud only)")
	}
}
