package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/lastfm"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/secrets"
)

const scrobbleDedupeWindow = 3 * time.Second

func warnLastfmCredentialsMissing() {
	if os.Getenv("LASTFM_API_KEY") != "" &&
		os.Getenv("LASTFM_API_SECRET") != "" &&
		os.Getenv("LASTFM_SESSION_KEY") != "" {
		return
	}
	fmt.Fprintf(os.Stderr, "evoplayer: last.fm scrobble disabled (missing pass creds omarchy/lastfm/*)\n")
}

func (d *Daemon) handleScrobble(req ipc.Request) (interface{}, error) {
	switch req.Method {
	case "scrobble.nowplaying":
		return nil, d.scrobbleNowPlaying()
	case "scrobble.submit":
		var p struct {
			Started int64 `json:"started"`
		}
		_ = ipc.DecodeParams(req.Params, &p)
		return nil, d.scrobbleSubmit(p.Started)
	default:
		return nil, fmt.Errorf("unknown scrobble method: %s", req.Method)
	}
}

func (d *Daemon) scrobbleNowPlaying() error {
	st := enrichTrack(d.Env, d.Actor.Snapshot())
	if st.Path == "" || st.Artist == "" || st.Title == "" {
		return nil
	}
	key := scrobbleDedupeKey("track.updateNowPlaying", st, 0)
	if d.scrobbleDuplicate(key) {
		return nil
	}
	if err := scrobbleAPI("track.updateNowPlaying", st, 0); err != nil {
		return err
	}
	return recordScrobble(d.Env.ScrobbleLog, st, "nowplaying", 0)
}

func (d *Daemon) scrobbleSubmit(started int64) error {
	st := enrichTrack(d.Env, d.Actor.Snapshot())
	if st.Path == "" || st.Artist == "" || st.Title == "" {
		return nil
	}
	if st.Duration > 0 && st.Duration <= 30 {
		return nil
	}
	if started <= 0 {
		started = int64(st.Position)
		if started < 0 {
			started = 0
		}
	}
	key := scrobbleDedupeKey("track.scrobble", st, started)
	if d.scrobbleDuplicate(key) {
		return nil
	}
	if err := scrobbleAPI("track.scrobble", st, started); err != nil {
		return err
	}
	return recordScrobble(d.Env.ScrobbleLog, st, "submit", started)
}

func scrobbleDedupeKey(method string, st playback.Status, started int64) string {
	if method == "track.scrobble" {
		return fmt.Sprintf("%s|%s|%d", method, st.Path, started)
	}
	return fmt.Sprintf("%s|%s|%s|%s", method, st.Path, st.Artist, st.Title)
}

func (d *Daemon) scrobbleDuplicate(key string) bool {
	now := time.Now()
	d.scrobbleMu.Lock()
	defer d.scrobbleMu.Unlock()
	if d.scrobbleDedupeKey == key && now.Sub(d.scrobbleDedupeAt) < scrobbleDedupeWindow {
		return true
	}
	d.scrobbleDedupeKey = key
	d.scrobbleDedupeAt = now
	return false
}

func enrichTrack(env paths.Env, st playback.Status) playback.Status {
	if st.Path == "" {
		return st
	}
	row, err := library.Meta(library.EnvFrom(env), st.Path, "")
	if err != nil {
		return st
	}
	if row.Title != "" {
		st.Title = row.Title
	}
	st.Artist = row.Artist
	st.Album = row.Album
	if st.Duration <= 0 {
		st.Duration = row.Duration
	}
	return st
}

func scrobbleAPI(method string, st playback.Status, timestamp int64) error {
	apiKey := os.Getenv("LASTFM_API_KEY")
	secret := os.Getenv("LASTFM_API_SECRET")
	session := os.Getenv("LASTFM_SESSION_KEY")
	if apiKey == "" || secret == "" || session == "" {
		return nil
	}
	return lastfm.APICall(lastfm.ScrobbleParams{
		Method:    method,
		APIKey:    apiKey,
		Secret:    secret,
		Session:   session,
		Artist:    st.Artist,
		Title:     st.Title,
		Album:     st.Album,
		Duration:  fmt.Sprintf("%.0f", st.Duration),
		Timestamp: fmt.Sprintf("%d", timestamp),
	})
}

func recordScrobble(path string, st playback.Status, event string, started int64) error {
	row := map[string]any{
		"event":  event,
		"path":   st.Path,
		"artist": st.Artist,
		"title":  st.Title,
		"album":  st.Album,
		"at":     time.Now().Format(time.RFC3339),
	}
	if started > 0 {
		row["started"] = started
	}
	b, err := json.Marshal(row)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func initScrobbleCredentials() {
	secrets.Load()
	warnLastfmCredentialsMissing()
}

// scrobbleListenThreshold returns seconds of playback required before a scrobble
// may be sent (Last.fm: half the track or 4 minutes, whichever is less). Tracks
// 30 seconds or shorter are not scrobbled.
func scrobbleListenThreshold(durationSec float64) float64 {
	if durationSec <= 30 {
		return -1
	}
	half := durationSec * 0.5
	if half > 240 {
		return 240
	}
	return half
}
