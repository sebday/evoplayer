package soundcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	seedTrack, artist, title, err := resolveSeed(client, seed)
	if err != nil {
		if errors.Is(err, errNoMatch) {
			if res, ok := catalogSimilar(client, opts, env.StateDir, artist, title); ok {
				return res, nil
			}
		}
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

func resolveSeed(client *Client, seed Seed) (*Track, string, string, error) {
	if seed.ID != 0 {
		track, err := client.Track(seed.ID)
		if err != nil {
			return nil, "", "", err
		}
		return track, track.User.Username, track.Title, nil
	}
	path := strings.TrimSpace(seed.Path)
	if path == "" {
		return nil, "", "", fmt.Errorf("soundcloud: path or id required")
	}
	probed, err := tags.Probe(path)
	if err != nil {
		return nil, "", "", err
	}
	artist := strings.TrimSpace(probed.Tag.Artist)
	title := strings.TrimSpace(probed.Tag.Title)
	if id, err := strconv.ParseInt(strings.TrimSpace(probed.Tag.SoundcloudID), 10, 64); err == nil && id > 0 {
		track, err := client.Track(id)
		return track, artist, title, err
	}
	track, err := client.MatchTrack(artist, title, probed.Duration)
	return track, artist, title, err
}

func artworkThumb(track Track) string {
	raw := strings.TrimSpace(track.ArtworkURL)
	if raw == "" {
		raw = strings.TrimSpace(track.User.AvatarURL)
	}
	raw = strings.NewReplacer("-large", "-t300x300", "-t500x500", "-t300x300", "-original", "-t300x300").Replace(raw)
	// SoundCloud labels some JPEG artwork with a .png path. Qt then refuses to decode it.
	lower := strings.ToLower(raw)
	if strings.Contains(lower, ".sndcdn.com/") && strings.Contains(lower, "/artworks-") && strings.HasSuffix(lower, ".png") {
		return raw[:len(raw)-4] + ".jpg"
	}
	return raw
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
		Artwork:   artworkThumb(track),
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

// errNoMatch is returned when search ran and no upload fit the artist and title.
var errNoMatch = errors.New("soundcloud: no matching track")

func matchMiss(artist, title string) error {
	label := strings.Trim(strings.TrimSpace(artist)+" - "+strings.TrimSpace(title), " -")
	return fmt.Errorf("%w for %s", errNoMatch, label)
}

// MatchTrack searches SoundCloud for artist and title. When durationSec is
// known, a hit within a few seconds wins. A strict uploader match wins;
// otherwise label and repost uploads that name the artist are accepted.
// Narrower queries run next, then the artist's own uploads, and if those
// still miss, the closest titled upload is used even when its length is off.
func (c *Client) MatchTrack(artist, title string, durationSec float64) (*Track, error) {
	artist = strings.TrimSpace(artist)
	title = strings.TrimSpace(title)
	full := strings.TrimSpace(artist + " " + title)
	if full == "" {
		return nil, fmt.Errorf("soundcloud: nothing to match")
	}
	tokens := artistTokens(artist)
	queries := []string{full}
	if artist != "" && title != "" {
		queries = append(queries, artist+" - "+title)
	}
	if title != "" {
		if len(tokens) > 0 {
			queries = append(queries, title+" "+tokens[0])
		}
		plain := strings.Join(strings.Fields(strings.ReplaceAll(artist, ".", " ")), " ")
		if plain != "" && plain != artist {
			queries = append(queries, strings.TrimSpace(plain+" "+title))
		}
		queries = append(queries, title)
	}
	var pool []Track
	var searchErr error
	tried := map[string]bool{}
	for i, q := range queries {
		if tried[q] {
			continue
		}
		tried[q] = true
		tracks, err := c.SearchTracks(q, 20)
		if err != nil {
			searchErr = err
			continue
		}
		if i == 0 {
			if best, err := pickMatch(tracks, artist, title, durationSec); err == nil {
				return best, nil
			}
		}
		pool = append(pool, tracks...)
		if best := pickLooseMatch(pool, tokens, title, durationSec); best != nil {
			return best, nil
		}
	}
	if best := pickFallbackMatch(pool, tokens, title, durationSec); best != nil {
		return best, nil
	}
	if best := c.matchArtistUploads(artist, title, tokens, durationSec); best != nil {
		return best, nil
	}
	if len(pool) == 0 && searchErr != nil {
		return nil, searchErr
	}
	return nil, matchMiss(artist, title)
}

type scUser struct {
	ID             int64  `json:"id"`
	Username       string `json:"username"`
	FollowersCount int64  `json:"followers_count"`
}

// pickArtistUser chooses the account whose name is the artist, including
// glued forms such as DJHatcha for "DJ Hatcha". A name that only contains
// the artist as a substring does not qualify.
func pickArtistUser(users []scUser, artist string) *scUser {
	want := tags.Slugify(artist)
	var best *scUser
	bestScore := 0
	for i := range users {
		score := overlapScore(want, tags.Slugify(users[i].Username))
		if score < 4 {
			continue
		}
		if best == nil || score > bestScore || (score == bestScore && users[i].FollowersCount > best.FollowersCount) {
			bestScore = score
			cp := users[i]
			best = &cp
		}
	}
	return best
}

func (c *Client) searchUsers(query string, limit int) ([]scUser, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("soundcloud: empty search")
	}
	if limit <= 0 || limit > 10 {
		limit = 10
	}
	path := "/search/users?q=" + url.QueryEscape(query) + "&limit=" + strconv.Itoa(limit) + "&linked_partitioning=1"
	body, err := c.getJSONWithClientID(path)
	if err != nil {
		return nil, err
	}
	var page struct {
		Collection []scUser `json:"collection"`
	}
	if err := decodeJSON(body, &page); err != nil {
		return nil, err
	}
	return page.Collection, nil
}

func (c *Client) userTracks(id int64, query string, limit int) ([]Track, error) {
	if id == 0 {
		return nil, fmt.Errorf("soundcloud: missing user id")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	path := fmt.Sprintf("/users/%d/tracks?limit=%d&linked_partitioning=1", id, limit)
	if q := strings.TrimSpace(query); q != "" {
		path = fmt.Sprintf("/users/%d/tracks?q=%s&limit=%d&linked_partitioning=1", id, url.QueryEscape(q), limit)
	}
	body, err := c.getJSONWithClientID(path)
	if err != nil {
		return nil, err
	}
	return decodeTrackList(body)
}

// matchArtistUploads searches the artist's account when global search never
// surfaces the title.
func (c *Client) matchArtistUploads(artist, title string, tokens []string, durationSec float64) *Track {
	users, err := c.searchUsers(artist, 8)
	if err != nil || len(users) == 0 {
		return nil
	}
	user := pickArtistUser(users, artist)
	if user == nil {
		return nil
	}
	tracks, err := c.userTracks(user.ID, title, 20)
	if err != nil || len(tracks) == 0 {
		return nil
	}
	if best, err := pickMatch(tracks, artist, title, durationSec); err == nil {
		return best
	}
	if best := pickLooseMatch(tracks, tokens, title, durationSec); best != nil {
		return best
	}
	return pickFallbackMatch(tracks, tokens, title, durationSec)
}

// catalogSimilar lists the artist's own short uploads when SoundCloud has
// no track under this title.
func catalogSimilar(client *Client, opts DownloadOptions, stateDir, artist, title string) (SimilarResult, bool) {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return SimilarResult{}, false
	}
	users, err := client.searchUsers(artist, 8)
	if err != nil {
		return SimilarResult{}, false
	}
	user := pickArtistUser(users, artist)
	if user == nil {
		return SimilarResult{}, false
	}
	tracks, err := client.userTracks(user.ID, "", 50)
	if err != nil {
		return SimilarResult{}, false
	}
	archive, err := syncarchive.Load(opts.ArchivePath)
	if err != nil {
		return SimilarResult{}, false
	}
	dismissed, err := dismissedSet(stateDir)
	if err != nil {
		return SimilarResult{}, false
	}
	out := SimilarResult{
		SeedTitle:  strings.TrimSpace(title),
		SeedArtist: artist,
		Tracks:     []SimilarTrack{},
	}
	seen := map[int64]bool{}
	for i := range tracks {
		track := tracks[i]
		if seen[track.ID] || !catalogTrack(&track) {
			continue
		}
		if archive.HasSC(track.ID) || dismissed[track.ID] {
			continue
		}
		seen[track.ID] = true
		out.Tracks = append(out.Tracks, similarTrack(track))
	}
	if len(out.Tracks) == 0 {
		return SimilarResult{}, false
	}
	return out, true
}

func catalogTrack(track *Track) bool {
	if !listable(track) {
		return false
	}
	return track.Duration >= 90*1000 && track.Duration <= 15*60*1000
}

var artistStopwords = map[string]bool{
	"and": true, "the": true, "feat": true, "featuring": true, "vs": true, "with": true,
	"productions": true, "records": true, "recordings": true, "record": true,
	"music": true, "sound": true, "sounds": true, "audio": true, "digital": true,
	"official": true, "presents": true, "label": true,
}

// artistTokens splits an artist tag like "Noisia Maldini And Vegas" into the
// names SoundCloud may credit on their own.
func artistTokens(artist string) []string {
	var out []string
	seen := map[string]bool{}
	for _, tok := range strings.Split(tags.Slugify(artist), "_") {
		if len(tok) < 3 || artistStopwords[tok] || seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out
}

// pickFallbackMatch chooses a titled upload that names the artist when nothing
// landed inside the duration window. Cleaner titles win, then the closer length.
func pickFallbackMatch(tracks []Track, tokens []string, title string, durationSec float64) *Track {
	wantTitle := tags.Slugify(stripMix(title))
	if wantTitle == "" || len(tokens) == 0 {
		return nil
	}
	var best *Track
	bestScore := -1
	bestDelta := math.MaxFloat64
	for i := range tracks {
		track := &tracks[i]
		if !listable(track) {
			continue
		}
		gotTitle := tags.Slugify(stripMix(track.Title))
		if !strings.Contains("_"+gotTitle+"_", "_"+wantTitle+"_") {
			continue
		}
		uploader := tags.Slugify(track.User.Username)
		credited := false
		for _, tok := range tokens {
			if tokenInSlug(uploader, tok) || tokenInSlug(gotTitle, tok) {
				credited = true
				break
			}
		}
		if !credited {
			continue
		}
		extra := 0
		for _, part := range strings.Split(gotTitle, "_") {
			if part == "" || len(part) < 3 || strings.Contains("_"+wantTitle+"_", "_"+part+"_") {
				continue
			}
			known := false
			for _, tok := range tokens {
				if part == tok {
					known = true
					break
				}
			}
			if !known {
				extra++
			}
		}
		score := 20 - extra
		delta := math.MaxFloat64
		if durationSec > 1 && track.Duration > 0 {
			delta = math.Abs(durationSec - float64(track.Duration)/1000)
		}
		if score > bestScore || (score == bestScore && delta < bestDelta) {
			bestScore = score
			bestDelta = delta
			cp := *track
			best = &cp
		}
	}
	return best
}

func pickLooseMatch(tracks []Track, tokens []string, title string, durationSec float64) *Track {
	wantTitle := tags.Slugify(stripMix(title))
	if wantTitle == "" || len(tokens) == 0 || durationSec <= 1 {
		return nil
	}
	var best *Track
	bestScore := 0
	for i := range tracks {
		track := &tracks[i]
		if !listable(track) {
			continue
		}
		score, ok := scoreLooseMatch(track, tokens, wantTitle, durationSec)
		if !ok || score <= bestScore {
			continue
		}
		bestScore = score
		cp := *track
		best = &cp
	}
	return best
}

func scoreLooseMatch(track *Track, tokens []string, wantTitle string, durationSec float64) (int, bool) {
	if track.Duration <= 0 || math.Abs(durationSec-float64(track.Duration)/1000) > durationSlackSec {
		return 0, false
	}
	gotTitle := tags.Slugify(stripMix(track.Title))
	score := 0
	switch {
	case gotTitle == wantTitle:
		score = 4
	case strings.Contains("_"+gotTitle+"_", "_"+wantTitle+"_"):
		score = 2
	default:
		return 0, false
	}
	uploader := tags.Slugify(track.User.Username)
	credited := false
	for _, tok := range tokens {
		switch {
		case tokenInSlug(uploader, tok):
			score += 2
			credited = true
		case tokenInSlug(gotTitle, tok):
			score++
			credited = true
		}
	}
	if !credited {
		return 0, false
	}
	return score, true
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
		return nil, matchMiss(artist, title)
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
	if score := overlapText(want, got); score > 0 {
		return score
	}
	return overlapText(strings.ReplaceAll(want, "_", ""), strings.ReplaceAll(got, "_", ""))
}

func overlapText(want, got string) int {
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

var gluedAffixes = map[string]bool{
	"dj": true, "mc": true, "the": true, "vs": true, "and": true,
}

// tokenInSlug reports whether tok is a word in slug, or glued to a short
// prefix such as "dj" in DJHatcha.
func tokenInSlug(slug, tok string) bool {
	if slug == "" || len(tok) < 3 {
		return false
	}
	if strings.Contains("_"+slug+"_", "_"+tok+"_") {
		return true
	}
	flat := strings.ReplaceAll(slug, "_", "")
	from := 0
	for from < len(flat) {
		i := strings.Index(flat[from:], tok)
		if i < 0 {
			return false
		}
		i += from
		if affixOnly(flat[:i]) && affixOnly(flat[i+len(tok):]) {
			return true
		}
		from = i + 1
	}
	return false
}

func affixOnly(s string) bool {
	return s == "" || gluedAffixes[s]
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
