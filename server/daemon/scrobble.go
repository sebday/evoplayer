package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/lastfm"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/secrets"
)

const (
	scrobbleDedupeWindow  = 3 * time.Second
	scrobblePendingBatch  = 50
	scrobblePendingMaxAge = 14 * 24 * time.Hour
)

func warnLastfmCredentialsMissing() {
	if os.Getenv("LASTFM_API_KEY") != "" &&
		os.Getenv("LASTFM_API_SECRET") != "" &&
		os.Getenv("LASTFM_SESSION_KEY") != "" {
		return
	}
	fmt.Fprintf(os.Stderr, "evoplayer: last.fm scrobble disabled (missing pass creds evoshell/lastfm/*)\n")
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
		return nil, ipc.ErrUnknownMethod(req.Method)
	}
}

// autoScrobble mirrors the Quickshell panel: now playing on track change, submit
// after min(half duration, 4 minutes) of continuous playback.
func (d *Daemon) autoScrobble(st playback.Status) {
	configured := secrets.LastfmConfigured()

	var (
		nowPlaying *playback.Status
		submit     *playback.Status
		submitAt   int64
	)

	d.scrobbleMu.Lock()
	prev := d.scrobblePrev

	if st.Path == "" {
		d.resetScrobbleSessionLocked()
		d.scrobblePrev = st
		d.scrobblePrevAt = time.Now()
		d.scrobbleMu.Unlock()
		return
	}

	// A seek, or the resume that reports 0 then jumps to the saved position,
	// is not listening time. Start the session again from the new position.
	if st.Path == prev.Path && st.State == "playing" &&
		scrobblePositionJumped(prev.Position, st.Position, scrobbleElapsed(d.scrobblePrevAt)) {
		d.beginScrobbleSessionLocked(st)
	}

	if prev.Path != "" && st.Path != prev.Path &&
		d.scrobblePath == prev.Path && !d.scrobbleSubmitted && prev.State == "playing" {
		if due, started := scrobbleSubmitDue(d.scrobblePath, d.scrobbleStartPos, d.scrobbleStartedAt, prev); due {
			prevCopy := prev
			submit = &prevCopy
			submitAt = started
			d.scrobbleSubmitted = true
		}
	}

	if st.State == "playing" {
		if st.Path != prev.Path {
			stCopy := st
			nowPlaying = &stCopy
			d.beginScrobbleSessionLocked(st)
		}
		if due, started := scrobbleSubmitDue(d.scrobblePath, d.scrobbleStartPos, d.scrobbleStartedAt, st); due && !d.scrobbleSubmitted {
			stCopy := st
			submit = &stCopy
			submitAt = started
			d.scrobbleSubmitted = true
		}
	}

	d.scrobblePrev = st
	d.scrobblePrevAt = time.Now()
	d.scrobbleMu.Unlock()

	if !configured || (nowPlaying == nil && submit == nil) {
		return
	}
	go func() {
		if nowPlaying != nil {
			if err := d.scrobbleNowPlayingStatus(*nowPlaying); err != nil {
				fmt.Fprintf(os.Stderr, "evoplayer: scrobble now playing: %v\n", err)
			}
		}
		if submit != nil {
			if err := d.scrobbleSubmitStatus(*submit, submitAt); err != nil {
				fmt.Fprintf(os.Stderr, "evoplayer: scrobble submit: %v\n", err)
			}
		}
	}()
}

func (d *Daemon) resetScrobbleSessionLocked() {
	d.scrobblePath = ""
	d.scrobbleStartPos = -1
	d.scrobbleStartedAt = 0
	d.scrobbleSubmitted = false
}

func (d *Daemon) beginScrobbleSessionLocked(st playback.Status) {
	if st.Path == "" {
		return
	}
	d.scrobblePath = st.Path
	d.scrobbleStartPos = st.Position
	if d.scrobbleStartPos < 0 {
		d.scrobbleStartPos = 0
	}
	d.scrobbleStartedAt = time.Now().Unix() - int64(d.scrobbleStartPos)
	d.scrobbleSubmitted = false
}

func scrobbleSubmitDue(scrobblePath string, startPos float64, startedAt int64, st playback.Status) (bool, int64) {
	if scrobblePath == "" || st.Path != scrobblePath || st.State != "playing" {
		return false, 0
	}
	if startPos < 0 {
		return false, 0
	}
	dur := st.Duration
	threshold := scrobbleListenThreshold(dur)
	if threshold < 0 || dur <= 0 {
		return false, 0
	}
	pos := st.Position
	if pos < startPos {
		startPos = pos
	}
	listened := pos - startPos
	if listened < threshold {
		return false, 0
	}
	started := startedAt
	if started <= 0 {
		started = time.Now().Unix() - int64(listened)
	}
	return true, started
}

func (d *Daemon) scrobbleNowPlaying() error {
	return d.scrobbleNowPlayingStatus(enrichTrack(d.env(), d.Actor.Snapshot()))
}

func (d *Daemon) scrobbleNowPlayingStatus(st playback.Status) error {
	st = enrichTrack(d.env(), st)
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
	return recordScrobble(d.env().ScrobbleLog, st, "nowplaying", 0)
}

func (d *Daemon) scrobbleSubmit(started int64) error {
	return d.scrobbleSubmitStatus(enrichTrack(d.env(), d.Actor.Snapshot()), started)
}

func (d *Daemon) scrobbleSubmitStatus(st playback.Status, started int64) error {
	st = enrichTrack(d.env(), st)
	if st.Path == "" || st.Artist == "" || st.Title == "" {
		return nil
	}
	if st.Duration > 0 && st.Duration <= 30 {
		return nil
	}
	if started <= 0 {
		started = time.Now().Unix()
	}
	key := scrobbleDedupeKey("track.scrobble", st, started)
	if d.scrobbleDuplicate(key) {
		return nil
	}
	if err := scrobbleAPI("track.scrobble", st, started); err != nil {
		if !strings.Contains(err.Error(), "ignored scrobble") {
			if qerr := d.queuePendingScrobble(st, started); qerr != nil {
				fmt.Fprintf(os.Stderr, "evoplayer: scrobble queue: %v\n", qerr)
			}
		}
		return err
	}
	if err := recordScrobble(d.env().ScrobbleLog, st, "submit", started); err != nil {
		return err
	}
	go d.flushPendingScrobbles()
	return nil
}

type pendingScrobble struct {
	Path     string  `json:"path"`
	Artist   string  `json:"artist"`
	Title    string  `json:"title"`
	Album    string  `json:"album,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Started  int64   `json:"started"`
}

func (p pendingScrobble) status() playback.Status {
	return playback.Status{Path: p.Path, Artist: p.Artist, Title: p.Title, Album: p.Album, Duration: p.Duration}
}

func (d *Daemon) queuePendingScrobble(st playback.Status, started int64) error {
	d.scrobblePendingMu.Lock()
	defer d.scrobblePendingMu.Unlock()
	path := d.env().ScrobblePending
	pending := readPendingScrobbles(path)
	pending = append(pending, pendingScrobble{
		Path:     st.Path,
		Artist:   st.Artist,
		Title:    st.Title,
		Album:    st.Album,
		Duration: st.Duration,
		Started:  started,
	})
	return writePendingScrobbles(path, pending)
}

// flushPendingScrobbles resubmits queued scrobbles oldest first; Last.fm rejects entries older than 14 days.
func (d *Daemon) flushPendingScrobbles() {
	if !secrets.LastfmConfigured() {
		return
	}
	d.scrobblePendingMu.Lock()
	defer d.scrobblePendingMu.Unlock()
	env := d.env()
	pending := readPendingScrobbles(env.ScrobblePending)
	if len(pending) == 0 {
		return
	}
	cutoff := time.Now().Add(-scrobblePendingMaxAge).Unix()
	keep := pending[:0]
	for _, p := range pending {
		if p.Started >= cutoff {
			keep = append(keep, p)
		}
	}
	sent := 0
	for _, p := range keep {
		if sent >= scrobblePendingBatch {
			break
		}
		st := p.status()
		if err := scrobbleAPI("track.scrobble", st, p.Started); err != nil {
			if !strings.Contains(err.Error(), "ignored scrobble") {
				fmt.Fprintf(os.Stderr, "evoplayer: scrobble pending: %v\n", err)
				break
			}
		} else {
			_ = recordScrobble(env.ScrobbleLog, st, "submit", p.Started)
		}
		sent++
	}
	if err := writePendingScrobbles(env.ScrobblePending, keep[sent:]); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: scrobble queue: %v\n", err)
	}
}

func readPendingScrobbles(path string) []pendingScrobble {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []pendingScrobble
	if err := json.Unmarshal(b, &out); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: scrobble queue: %v\n", err)
		return nil
	}
	return out
}

func writePendingScrobbles(path string, pending []pendingScrobble) error {
	if len(pending) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	b, err := json.Marshal(pending)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

const scrobbleSeekSlack = 5.0

// scrobbleElapsed is how long since the previous playback sample. A zero time
// means there is no previous sample, so a forward jump cannot be judged yet.
func scrobbleElapsed(prev time.Time) float64 {
	if prev.IsZero() {
		return 0
	}
	elapsed := time.Since(prev).Seconds()
	if elapsed < 0 {
		return 0
	}
	return elapsed
}

// scrobblePositionJumped reports a seek. Playback moves about one second per
// second; anything further than that, in either direction, was not heard.
func scrobblePositionJumped(prevPos, pos, elapsed float64) bool {
	if prevPos < 0 || pos < 0 {
		return false
	}
	if pos < prevPos-scrobbleSeekSlack {
		return true
	}
	return pos > prevPos+elapsed+scrobbleSeekSlack
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
