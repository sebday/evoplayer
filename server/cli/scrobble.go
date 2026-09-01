package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/sebday/evoplayer/server/jsonlog"
	"github.com/sebday/evoplayer/server/lastfm"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/secrets"
)

func CmdScrobble(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer scrobble <auth|token|nowplaying|submit|recent|touch>")
	}
	switch args[0] {
	case "auth":
		return scrobbleAuth()
	case "token":
		if len(args) < 2 {
			return fmt.Errorf("usage: evoplayer scrobble token <token>")
		}
		return scrobbleToken(args[1])
	case "nowplaying":
		return scrobbleNowPlaying(env)
	case "submit":
		started := int64(0)
		for i := 1; i < len(args); i++ {
			if args[i] == "--started" && i+1 < len(args) {
				started, _ = strconv.ParseInt(args[i+1], 10, 64)
			}
		}
		return scrobbleSubmit(env, started)
	case "touch":
		return scrobbleTouch(env)
	case "recent":
		limit := 12
		jsonOut := false
		for _, a := range args[1:] {
			switch a {
			case "--json":
				jsonOut = true
			default:
				if n, err := strconv.Atoi(a); err == nil && n > 0 {
					limit = n
				}
			}
		}
		return scrobbleRecent(env, limit, jsonOut)
	default:
		return fmt.Errorf("usage: evoplayer scrobble <auth|token|nowplaying|submit|recent|touch>")
	}
}

func scrobbleAuth() error {
	secrets.Load()
	apiKey := os.Getenv("LASTFM_API_KEY")
	secret := os.Getenv("LASTFM_API_SECRET")
	if apiKey == "" || secret == "" {
		return fmt.Errorf("evoplayer: set LASTFM_API_KEY and LASTFM_API_SECRET in pass (omarchy/lastfm/*)")
	}
	if os.Getenv("LASTFM_SESSION_KEY") != "" {
		fmt.Println("evoplayer: LASTFM_SESSION_KEY already set in pass")
		return nil
	}
	url := fmt.Sprintf("https://www.last.fm/api/auth/?api_key=%s&cb=http%%3A%%2F%%2Flocalhost", apiKey)
	fmt.Println("open:", url)
	fmt.Println("after approving, copy the token from the redirect URL and run:")
	fmt.Println("  evoplayer scrobble token <token>")
	_ = exec.Command("xdg-open", url).Run()
	return nil
}

func scrobbleToken(token string) error {
	secrets.Load()
	apiKey := os.Getenv("LASTFM_API_KEY")
	secret := os.Getenv("LASTFM_API_SECRET")
	if token == "" || apiKey == "" || secret == "" {
		return fmt.Errorf("evoplayer: usage: evoplayer scrobble token <token>")
	}
	session, err := lastfm.AuthSession(apiKey, secret, token)
	if err != nil {
		return err
	}
	fmt.Println("evoplayer: session key:", session)
	return nil
}

func scrobbleNowPlaying(env paths.Env) error {
	secrets.Load()
	if DaemonUp(env) {
		_, err := IPC(env, "scrobble.nowplaying", nil)
		return err
	}
	st, err := currentTrack(env)
	if err != nil || st.Artist == "" || st.Title == "" {
		return nil
	}
	return lastfmScrobbleAPI("track.updateNowPlaying", st.Artist, st.Title, st.Album, fmt.Sprintf("%.0f", st.Duration), "", "", "")
}

func scrobbleSubmit(env paths.Env, started int64) error {
	secrets.Load()
	if DaemonUp(env) {
		_, err := IPC(env, "scrobble.submit", map[string]int64{"started": started})
		return err
	}
	st, err := currentTrack(env)
	if err != nil || st.Artist == "" || st.Title == "" {
		return nil
	}
	if started <= 0 {
		started = time.Now().Unix()
	}
	mbid, _ := lastfm.RecordingMBID(st.Artist, st.Title, st.Album)
	return lastfmScrobbleAPI("track.scrobble", st.Artist, st.Title, st.Album, fmt.Sprintf("%.0f", st.Duration), fmt.Sprintf("%d", started), "", mbid)
}

func scrobbleTouch(env paths.Env) error {
	st, err := currentTrack(env)
	if err != nil || st.Path == "" {
		return nil
	}
	row := map[string]any{
		"event": "nowplaying", "path": st.Path,
		"artist": st.Artist, "title": st.Title, "album": st.Album,
		"at": time.Now().Format(time.RFC3339),
	}
	b, _ := json.Marshal(row)
	f, err := os.OpenFile(env.ScrobbleLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func scrobbleRecent(env paths.Env, limit int, jsonOut bool) error {
	rows, err := jsonlog.ScrobbleRecent(env.ScrobbleLog, limit)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSONRows(rows)
	}
	for _, row := range rows {
		artist, _ := row["artist"].(string)
		title, _ := row["title"].(string)
		fmt.Printf("%s — %s\n", artist, title)
	}
	return nil
}

func currentTrack(env paths.Env) (library.Track, error) {
	path := ""
	if DaemonUp(env) {
		st, err := PlaybackStatus(env)
		if err == nil {
			path = st.Path
		}
	}
	if path == "" {
		b, err := os.ReadFile(env.PlayerState)
		if err != nil {
			return library.Track{}, err
		}
		var saved struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(b, &saved) != nil || saved.Path == "" {
			return library.Track{}, fmt.Errorf("no track")
		}
		path = saved.Path
	}
	return library.Meta(library.EnvFrom(env), path, "")
}
