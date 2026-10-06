package cli

import (
	"fmt"
	"os"

	"github.com/sebday/evoplayer/server/lastfm"
	"github.com/sebday/evoplayer/server/paths"
)

func CmdLastfm(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer lastfm recording-mbid <artist> <title> [album]")
	}
	switch args[0] {
	case "recording-mbid":
		if len(args) < 3 {
			return fmt.Errorf("usage: evoplayer lastfm recording-mbid <artist> <title> [album]")
		}
		album := ""
		if len(args) > 3 {
			album = args[3]
		}
		mbid, err := lastfm.RecordingMBID(args[1], args[2], album)
		if err != nil {
			os.Exit(1)
		}
		fmt.Print(mbid)
		return nil
	default:
		return fmt.Errorf("usage: evoplayer lastfm recording-mbid <artist> <title> [album]")
	}
}

func lastfmScrobbleAPI(method, artist, title, album, duration, timestamp, albumArtist, mbid string) error {
	apiKey := os.Getenv("LASTFM_API_KEY")
	secret := os.Getenv("LASTFM_API_SECRET")
	session := os.Getenv("LASTFM_SESSION_KEY")
	if apiKey == "" || secret == "" || session == "" {
		return fmt.Errorf("evoplayer: last.fm credentials missing in pass (evoshell/lastfm/*)")
	}
	return lastfm.APICall(lastfm.ScrobbleParams{
		Method:      method,
		APIKey:      apiKey,
		Secret:      secret,
		Session:     session,
		Artist:      artist,
		Title:       title,
		Album:       album,
		Duration:    duration,
		Timestamp:   timestamp,
		AlbumArtist: albumArtist,
		MBID:        mbid,
	})
}
