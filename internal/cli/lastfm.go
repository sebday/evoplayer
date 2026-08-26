package cli

import (
	"fmt"
	"os"

	"github.com/sebday/evoplayer/internal/lastfm"
	"github.com/sebday/evoplayer/internal/paths"
)

func CmdLastfm(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer lastfm <auth-session|scrobble-api|recording-mbid>")
	}
	switch args[0] {
	case "auth-session":
		if len(args) < 4 {
			return fmt.Errorf("usage: evoplayer lastfm auth-session <api_key> <secret> <token>")
		}
		session, err := lastfm.AuthSession(args[1], args[2], args[3])
		if err != nil {
			return err
		}
		fmt.Print(session)
		return nil
	case "scrobble-api":
		if len(args) < 12 {
			return fmt.Errorf("usage: evoplayer lastfm scrobble-api <method> <api_key> <secret> <session> <artist> <title> <album> <duration> <timestamp> <album_artist> <mbid>")
		}
		return lastfm.APICall(lastfm.ScrobbleParams{
			Method:      args[1],
			APIKey:      args[2],
			Secret:      args[3],
			Session:     args[4],
			Artist:      args[5],
			Title:       args[6],
			Album:       args[7],
			Duration:    args[8],
			Timestamp:   args[9],
			AlbumArtist: args[10],
			MBID:        args[11],
		})
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
		return fmt.Errorf("usage: evoplayer lastfm <auth-session|scrobble-api|recording-mbid>")
	}
}

func lastfmScrobbleAPI(method, artist, title, album, duration, timestamp, albumArtist, mbid string) error {
	apiKey := os.Getenv("LASTFM_API_KEY")
	secret := os.Getenv("LASTFM_API_SECRET")
	session := os.Getenv("LASTFM_SESSION_KEY")
	if apiKey == "" || secret == "" || session == "" {
		return fmt.Errorf("evoplayer: last.fm credentials missing in pass (omarchy/lastfm/*)")
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
