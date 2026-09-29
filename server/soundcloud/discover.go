package soundcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/syncarchive"
	"github.com/sebday/evoplayer/server/tags"
)

const (
	durationSlackSec = 5.0
	relatedLimit     = 50
	previewKeepN     = 20
)

var mixSuffixRe = regexp.MustCompile(`\s*[\(\[][^)\]]*[\)\]]`)

// Seed identifies the track similar results are built from.
type Seed struct {
	Path string
	ID   int64
}

// SimilarTrack is one row in a discover list.
type SimilarTrack struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Artist    string  `json:"artist"`
	Duration  float64 `json:"duration"`
	Artwork   string  `json:"artwork"`
	Permalink string  `json:"permalink"`
}

// SimilarResult is the discover feed for one seed track.
type SimilarResult struct {
	SeedID     int64          `json:"seed_id"`
	SeedTitle  string         `json:"seed_title"`
	SeedArtist string         `json:"seed_artist"`
	Tracks     []SimilarTrack `json:"tracks"`
}

// Similar matches a local file or SoundCloud id, then returns related tracks
// that are not already archived or dismissed.
func Similar(env paths.Env, seed Seed) (SimilarResult, error) {
	opts, err := LoadOptions(env)
	if err != nil {
		return SimilarResult{}, err
	}
	client := NewClient(opts.ClientID, opts.OAuthToken)
	seedTrack, err := resolveSeed(client, seed)
	if err != nil {
		return SimilarResult{}, err
	}
	related, err := client.RelatedTracks(seedTrack.ID, relatedLimit)
	if err != nil {
		return SimilarResult{}, err
	}
	archive, err := syncarchive.Load(opts.ArchivePath)
	if err != nil {
		return SimilarResult{}, err
	}
	dismissed, err := dismissedSet(env.StateDir)
	if err != nil {
		return SimilarResult{}, err
	}
	out := SimilarResult{
		SeedID:     seedTrack.ID,
		SeedTitle:  strings.TrimSpace(seedTrack.Title),
		SeedArtist: strings.TrimSpace(seedTrack.User.Username),
		Tracks:     []SimilarTrack{},
	}
	seen := map[int64]bool{seedTrack.ID: true}
	for _, track := range related {
		if track.ID == 0 || seen[track.ID] || !listable(&track) {
			continue
		}
		if archive.HasSC(track.ID) || dismissed[track.ID] {
			continue
		}
		seen[track.ID] = true
		out.Tracks = append(out.Tracks, similarTrack(track))
		if len(out.Tracks) >= relatedLimit {
			break
		}
	}
	return out, nil
}

func resolveSeed(client *Client, seed Seed) (*Track, error) {
	if seed.ID != 0 {
		return client.Track(seed.ID)
	}
	path := strings.TrimSpace(seed.Path)
	if path == "" {
		return nil, fmt.Errorf("soundcloud: path or id required")
	}
	probed, err := tags.Probe(path)
	if err != nil {
		return nil, err
	}
	if id, err := strconv.ParseInt(strings.TrimSpace(probed.Tag.SoundcloudID), 10, 64); err == nil && id > 0 {
		return client.Track(id)
	}
	return client.MatchTrack(probed.Tag.Artist, probed.Tag.Title, probed.Duration)
}

func similarTrack(track Track) SimilarTrack {
	dur := 0.0
	if track.Duration > 0 {
		dur = float64(track.Duration) / 1000
	}
	return SimilarTrack{
		ID:        track.ID,
		Title:     strings.TrimSpace(track.Title),
		Artist:    strings.TrimSpace(track.User.Username),
		Duration:  dur,
		Artwork:   strings.TrimSpace(track.ArtworkURL),
		Permalink: strings.TrimSpace(track.PermalinkURL),
	}
}

// Track fetches one SoundCloud track by id.
func (c *Client) Track(id int64) (*Track, error) {
	if id == 0 {
		return nil, fmt.Errorf("soundcloud: missing track id")
	}
	body, err := c.getJSONWithClientID(fmt.Sprintf("/tracks/%d", id))
	if err != nil {
		return nil, err
	}
	var track Track
	if err := decodeJSON(body, &track); err != nil {
		return nil, err
	}
	if track.ID == 0 {
		return nil, fmt.Errorf("soundcloud: track %d not found", id)
	}
	return &track, nil
}

// MatchTrack searches SoundCloud for artist and title. When durationSec is
// known, the hit must be within a few seconds.
func (c *Client) MatchTrack(artist, title string, durationSec float64) (*Track, error) {
	q := strings.TrimSpace(strings.TrimSpace(artist) + " " + strings.TrimSpace(title))
	if q == "" {
		return nil, fmt.Errorf("soundcloud: nothing to match")
	}
	tracks, err := c.SearchTracks(q, 10)
	if err != nil {
		return nil, err
	}
	return pickMatch(tracks, artist, title, durationSec)
}

// SearchTracks returns SoundCloud track results for a query.
func (c *Client) SearchTracks(query string, limit int) ([]Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("soundcloud: empty search")
	}
	if limit <= 0 {
		limit = 12
	}
	if limit > 25 {
		limit = 25
	}
	path := "/search/tracks?q=" + url.QueryEscape(query) + "&limit=" + strconv.Itoa(limit) + "&linked_partitioning=1"
	body, err := c.getJSONWithClientID(path)
	if err != nil {
		return nil, err
	}
	return decodeTrackList(body)
}

// Search looks up SoundCloud tracks that can be downloaded.
func Search(env paths.Env, query string, limit int) ([]SimilarTrack, error) {
	opts, err := LoadOptions(env)
	if err != nil {
		return nil, err
	}
	tracks, err := NewClient(opts.ClientID, opts.OAuthToken).SearchTracks(query, limit)
	if err != nil {
		return nil, err
	}
	out := make([]SimilarTrack, 0, len(tracks))
	seen := map[int64]bool{}
	for i := range tracks {
		track := tracks[i]
		if seen[track.ID] || !listable(&track) || strings.TrimSpace(track.PermalinkURL) == "" {
			continue
		}
		seen[track.ID] = true
		out = append(out, similarTrack(track))
	}
	return out, nil
}

// RelatedTracks returns SoundCloud's similar-track feed, falling back to the
// station list autoplay uses.
func (c *Client) RelatedTracks(id int64, limit int) ([]Track, error) {
	if id == 0 {
		return nil, fmt.Errorf("soundcloud: missing track id")
	}
	if limit <= 0 {
		limit = relatedLimit
	}
	path := fmt.Sprintf("/tracks/%d/related?limit=%d&linked_partitioning=1", id, limit)
	body, err := c.getJSONWithClientID(path)
	if err == nil {
		tracks, decErr := decodeTrackList(body)
		if decErr == nil && len(tracks) > 0 {
			return tracks, nil
		}
		if decErr != nil {
			err = decErr
		}
	}
	station := fmt.Sprintf("/stations/soundcloud:track-stations:%d/tracks?limit=%d&linked_partitioning=1", id, limit)
	body, serr := c.getJSONWithClientID(station)
	if serr != nil {
		if err != nil {
			return nil, err
		}
		return nil, serr
	}
	return decodeTrackList(body)
}

func decodeTrackList(body []byte) ([]Track, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("soundcloud: empty track list")
	}
	if body[0] == '[' {
		var tracks []Track
		if err := json.Unmarshal(body, &tracks); err != nil {
			return nil, fmt.Errorf("soundcloud: decode: %w", err)
		}
		return tracks, nil
	}
	var page struct {
		Collection []json.RawMessage `json:"collection"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("soundcloud: decode: %w", err)
	}
	out := make([]Track, 0, len(page.Collection))
	for _, raw := range page.Collection {
		if track, ok := unwrapTrack(raw); ok {
			out = append(out, track)
		}
	}
	return out, nil
}

func unwrapTrack(raw json.RawMessage) (Track, bool) {
	var track Track
	if json.Unmarshal(raw, &track) == nil && track.ID != 0 {
		return track, true
	}
	var wrap struct {
		Track Track `json:"track"`
	}
	if json.Unmarshal(raw, &wrap) == nil && wrap.Track.ID != 0 {
		return wrap.Track, true
	}
	return Track{}, false
}

func listable(track *Track) bool {
	if track == nil || track.ID == 0 {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(track.Policy)) {
	case "BLOCK", "SNIP":
		return false
	}
	if track.Streamable != nil && !*track.Streamable {
		return false
	}
	if len(track.Media.Transcodings) == 0 {
		return true
	}
	for _, tc := range track.Media.Transcodings {
		if !strings.Contains(tc.Format.Protocol, "encrypted") {
			return true
		}
	}
	return false
}

func pickMatch(tracks []Track, artist, title string, durationSec float64) (*Track, error) {
	wantArtist := tags.Slugify(artist)
	wantTitle := tags.Slugify(stripMix(title))
	if wantArtist == "" && wantTitle == "" {
		return nil, fmt.Errorf("soundcloud: nothing to match")
	}
	var best *Track
	bestScore := 0
	for i := range tracks {
		track := &tracks[i]
		if !listable(track) {
			continue
		}
		score, ok := scoreMatch(track, wantArtist, wantTitle, durationSec)
		if !ok || score <= bestScore {
			continue
		}
		bestScore = score
		cp := *track
		best = &cp
	}
	if best == nil {
		label := strings.TrimSpace(strings.TrimSpace(artist) + " - " + strings.TrimSpace(title))
		return nil, fmt.Errorf("soundcloud: no matching track for %s", label)
	}
	return best, nil
}

func scoreMatch(track *Track, wantArtist, wantTitle string, durationSec float64) (int, bool) {
	gotArtist := tags.Slugify(track.User.Username)
	gotTitle := tags.Slugify(stripMix(track.Title))
	titleScore := overlapScore(wantTitle, gotTitle)
	artistScore := overlapScore(wantArtist, gotArtist)
	durationKnown := durationSec > 1 && track.Duration > 0
	durationOK := false
	if durationKnown {
		delta := math.Abs(durationSec - float64(track.Duration)/1000)
		if delta > durationSlackSec {
			return 0, false
		}
		durationOK = true
	}
	if wantTitle != "" && titleScore < 2 {
		return 0, false
	}
	if wantArtist != "" && artistScore < 2 {
		return 0, false
	}
	if titleScore < 2 && artistScore < 2 {
		return 0, false
	}
	score := titleScore + artistScore
	if durationOK {
		score += 3
	}
	return score, true
}

func overlapScore(want, got string) int {
	if want == "" || got == "" {
		return 0
	}
	if want == got {
		return 4
	}
	if strings.Contains(got, want) || strings.Contains(want, got) {
		return 2
	}
	return 0
}

func stripMix(s string) string {
	return strings.TrimSpace(mixSuffixRe.ReplaceAllString(s, " "))
}

// PreviewDir is the cache folder for discover previews.
func PreviewDir(cacheDir string) string {
	return filepath.Join(cacheDir, "discover")
}

// PreviewPath is the cached mp3 for a SoundCloud track id.
func PreviewPath(cacheDir string, id int64) string {
	return filepath.Join(PreviewDir(cacheDir), fmt.Sprintf("%d.mp3", id))
}

// IsPreviewPath reports whether path is a file inside the discover cache.
func IsPreviewPath(cacheDir, path string) bool {
	dir := filepath.Clean(PreviewDir(cacheDir))
	path = filepath.Clean(path)
	if dir == "" || path == "" {
		return false
	}
	return path == dir || strings.HasPrefix(path, dir+string(os.PathSeparator))
}

// DownloadPreview saves a SoundCloud track into the discover cache.
func DownloadPreview(ctx context.Context, env paths.Env, id int64, rep jobs.Reporter) (string, error) {
	if rep == nil {
		rep = jobs.NopReporter
	}
	if id == 0 {
		return "", fmt.Errorf("soundcloud: missing track id")
	}
	opts, err := LoadOptions(env)
	if err != nil {
		return "", err
	}
	dest := PreviewPath(env.CacheDir, id)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if st, err := os.Stat(dest); err == nil && !st.IsDir() && st.Size() > 0 {
		touchNow(dest)
		prunePreviews(filepath.Dir(dest), previewKeepN)
		rep.Line(jobs.LogSkip(filepath.Base(dest)))
		return dest, nil
	}
	client := NewClient(opts.ClientID, opts.OAuthToken)
	track, err := client.Track(id)
	if err != nil {
		return "", err
	}
	if !listable(track) {
		return "", fmt.Errorf("soundcloud: track %d is not playable", id)
	}
	label := trackLabel(*track)
	rep.Progress(jobs.Progress{Phase: label, Done: 0, Total: 1})
	rep.Line(jobs.LogInfof("preview %s", label))
	if err := downloadTrack(ctx, client, track, dest, opts); err != nil {
		os.Remove(dest)
		return "", err
	}
	touchNow(dest)
	prunePreviews(filepath.Dir(dest), previewKeepN)
	rep.Line(jobs.LogOK(filepath.Base(dest)))
	return dest, nil
}

func trackLabel(track Track) string {
	label := strings.TrimSpace(track.User.Username)
	title := strings.TrimSpace(track.Title)
	if label != "" && title != "" {
		return label + " - " + title
	}
	if title != "" {
		return title
	}
	return label
}

func touchNow(path string) {
	now := time.Now()
	_ = os.Chtimes(path, now, now)
}

func prunePreviews(dir string, keep int) {
	if keep < 1 {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type item struct {
		path string
		mod  time.Time
	}
	var files []item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".mp3") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, item{path: filepath.Join(dir, e.Name()), mod: info.ModTime()})
	}
	if len(files) <= keep {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	for _, f := range files[keep:] {
		_ = os.Remove(f.path)
	}
}

// Keep moves a cached preview into .incoming, or downloads the permalink when
// there is no preview.
func Keep(ctx context.Context, env paths.Env, id int64, rep jobs.Reporter) (string, error) {
	if rep == nil {
		rep = jobs.NopReporter
	}
	if id == 0 {
		return "", fmt.Errorf("soundcloud: missing track id")
	}
	preview := PreviewPath(env.CacheDir, id)
	if st, err := os.Stat(preview); err == nil && !st.IsDir() && st.Size() > 0 {
		return keepFile(ctx, env, id, preview, rep)
	}
	opts, err := LoadOptions(env)
	if err != nil {
		return "", err
	}
	client := NewClient(opts.ClientID, opts.OAuthToken)
	track, err := client.Track(id)
	if err != nil {
		return "", err
	}
	pageURL := strings.TrimSpace(track.PermalinkURL)
	if pageURL == "" {
		return "", fmt.Errorf("soundcloud: track %d has no permalink", id)
	}
	return DownloadTrackURLCtx(ctx, env, pageURL, rep)
}

func keepFile(ctx context.Context, env paths.Env, id int64, src string, rep jobs.Reporter) (string, error) {
	opts, err := LoadOptions(env)
	if err != nil {
		return "", err
	}
	if opts.MusicRoot == "" {
		return "", fmt.Errorf("evoplayer: music root not configured")
	}
	incoming := filepath.Join(opts.MusicRoot, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return "", err
	}
	info, _ := tags.ReadTags(src)
	artist := tags.SanitizeFilenamePart(info.Artist)
	title := tags.SanitizeFilenamePart(info.Title)
	name := strings.TrimSpace(artist + " - " + title)
	name = strings.Trim(name, "- ")
	if name == "" {
		name = fmt.Sprintf("soundcloud-%d", id)
	}
	dest := filepath.Join(incoming, name+".mp3")
	if _, err := os.Stat(dest); err == nil {
		_ = os.Remove(src)
	} else if err := moveFile(src, dest); err != nil {
		return "", err
	}
	archive, err := syncarchive.Load(opts.ArchivePath)
	if err != nil {
		return "", err
	}
	archiveSC(archive, id, rep)
	if err := NormalizeIncoming(ctx, opts.MusicRoot); err != nil {
		return "", err
	}
	rep.Line(jobs.LogOK(filepath.Base(dest)))
	return dest, nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr == nil {
		copyErr = os.Rename(tmp, dst)
	}
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	return os.Remove(src)
}

var dismissedMu sync.Mutex

func dismissedPath(stateDir string) string {
	return filepath.Join(stateDir, "discover-dismissed.json")
}

// DismissID records a SoundCloud id so later discover lists skip it.
func DismissID(stateDir string, id int64) error {
	if id == 0 {
		return fmt.Errorf("soundcloud: missing track id")
	}
	dismissedMu.Lock()
	defer dismissedMu.Unlock()
	set, err := readDismissed(stateDir)
	if err != nil {
		return err
	}
	set[id] = true
	return writeDismissed(stateDir, set)
}

func dismissedSet(stateDir string) (map[int64]bool, error) {
	dismissedMu.Lock()
	defer dismissedMu.Unlock()
	return readDismissed(stateDir)
}

func readDismissed(stateDir string) (map[int64]bool, error) {
	set := map[int64]bool{}
	b, err := os.ReadFile(dismissedPath(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return set, nil
		}
		return nil, err
	}
	var file struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.Unmarshal(b, &file); err != nil {
		return nil, fmt.Errorf("soundcloud: discover dismissed: %w", err)
	}
	for _, id := range file.IDs {
		if id != 0 {
			set[id] = true
		}
	}
	return set, nil
}

func writeDismissed(stateDir string, set map[int64]bool) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return err
	}
	ids := make([]int64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	b, err := json.Marshal(struct {
		IDs []int64 `json:"ids"`
	}{IDs: ids})
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := dismissedPath(stateDir) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dismissedPath(stateDir))
}
