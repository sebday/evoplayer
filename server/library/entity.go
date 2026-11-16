package library

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

const (
	entityTrackLimit   = 2000
	entityReleaseLimit = 60
)

type EntityRef struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Art     string `json:"art,omitempty"`
	Thumb   string `json:"thumb,omitempty"`
	Starred bool   `json:"starred,omitempty"`
}

type EntityRelease struct {
	Name   string `json:"name"`
	Year   string `json:"year,omitempty"`
	Dir    string `json:"dir,omitempty"`
	Artist string `json:"artist,omitempty"`
	Label  string `json:"label,omitempty"`
	Count  int    `json:"count"`
	Art    string `json:"art,omitempty"`
	Thumb  string `json:"thumb,omitempty"`
}

type EntityPage struct {
	Kind     string          `json:"kind"`
	Name     string          `json:"name"`
	Dir      string          `json:"dir,omitempty"`
	Total    int             `json:"total"`
	Duration float64         `json:"duration"`
	Liked    int             `json:"liked"`
	Starred  bool            `json:"starred"`
	Years    string          `json:"years,omitempty"`
	Credits  []string        `json:"credits,omitempty"`
	Art      string          `json:"art,omitempty"`
	Thumb    string          `json:"thumb,omitempty"`
	Tracks   []Track         `json:"tracks"`
	Releases []EntityRelease `json:"releases"`
	Artists  []EntityRef     `json:"artists"`
	Labels   []EntityRef     `json:"labels"`
	Genres   []EntityRef     `json:"genres"`
}

var creditSeparators = []string{
	" (featuring ", " (feat. ", " (ft. ",
	" featuring ", " feat. ", " feat ", " ft. ", " ft ",
	" vs. ", " vs ", " x ", " & ", " + ", ", ", " / ",
}

// ArtistCredits splits a collaboration credit into its names.
// The whole credit stays first so a duo name still matches itself.
func ArtistCredits(artist string) []string {
	whole := strings.TrimSpace(artist)
	if whole == "" {
		return nil
	}
	parts := []string{whole}
	for _, sep := range creditSeparators {
		var next []string
		for _, p := range parts {
			next = append(next, splitFold(p, sep)...)
		}
		parts = next
	}
	out := []string{whole}
	seen := map[string]bool{strings.ToLower(whole): true}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Count(p, "(") != strings.Count(p, ")") {
			p = strings.Trim(p, "() ")
		}
		key := strings.ToLower(p)
		if p == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

func splitFold(s, sep string) []string {
	lower := strings.ToLower(s)
	var out []string
	for {
		i := strings.Index(lower, sep)
		if i < 0 {
			return append(out, s)
		}
		out = append(out, s[:i])
		s = s[i+len(sep):]
		lower = lower[i+len(sep):]
	}
}

func creditedTo(artist, lname string) bool {
	for _, c := range ArtistCredits(artist) {
		if strings.ToLower(c) == lname {
			return true
		}
	}
	return false
}

func likeEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}

// Entity collects the tracks credited to an artist, label, album, or genre.
func Entity(env Env, kind, name, dir string) (EntityPage, error) {
	name = strings.TrimSpace(name)
	page := EntityPage{Kind: kind, Name: name, Dir: dir}
	if name == "" {
		return page, fmt.Errorf("name required")
	}
	db, err := EnsureDB(env)
	if err != nil {
		return page, err
	}
	lname := strings.ToLower(name)
	base := `SELECT path, COALESCE(genre,''), COALESCE(title,''), COALESCE(artist,''), COALESCE(album,''),
COALESCE(year,''), COALESCE(label,''), COALESCE(duration,0), COALESCE(art,''), COALESCE(waveform,''), COALESCE(liked,0)
FROM tracks WHERE `
	var query string
	var args []any
	keep := func(Track) bool { return true }
	switch kind {
	case "artist":
		query = base + `lower(artist) LIKE ? ESCAPE '\'`
		args = []any{likeEscape(lname)}
		keep = func(t Track) bool { return creditedTo(t.Artist, lname) }
	case "label":
		query = base + `lower(trim(label)) = ?`
		args = []any{lname}
	case "album":
		query = base + `lower(trim(album)) = ?`
		args = []any{lname}
		if dir != "" {
			query += ` AND parent_dir = ?`
			args = append(args, filepath.Clean(dir))
		}
	case "genre":
		want := strings.TrimSpace(name)
		if want == "" {
			return page, fmt.Errorf("genre required")
		}
		query = base + `replace(trim(genre), char(0), '') = ?`
		args = []any{tags.TrimFrame(want)}
	case "year":
		want := yearGroup(name)
		if want == "" {
			return page, fmt.Errorf("year required")
		}
		query = base + `year != ''`
		keep = func(t Track) bool { return yearGroup(t.Year) == want }
	default:
		return page, fmt.Errorf("unknown kind: %s", kind)
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return page, err
	}
	matched, err := scanEntityRows(rows, keep)
	if err != nil {
		return page, err
	}
	switch kind {
	case "genre":
		page.Name = mostCommonSpelling(matched, func(t Track) string { return t.Genre }, name)
	case "label":
		page.Name = mostCommonSpelling(matched, func(t Track) string { return t.Label }, name)
	case "album":
		page.Name = mostCommonSpelling(matched, func(t Track) string { return t.Album }, name)
	case "artist":
		page.Name = mostCommonSpelling(matched, func(t Track) string {
			if strings.ToLower(strings.TrimSpace(t.Artist)) == lname {
				return t.Artist
			}
			return ""
		}, name)
	case "year":
		page.Name = yearGroup(name)
	}
	if kind == "artist" {
		if credits := ArtistCredits(name); len(credits) > 1 {
			page.Credits = credits[1:]
		}
		page.Starred = artistStarred(env, page.Name)
	}
	if kind == "album" {
		page.Starred = albumStarred(env, page.Name)
	}
	if kind == "label" {
		page.Starred = labelStarred(env, page.Name)
	}
	sortEntityTracks(matched)
	summarizeEntity(env, &page, matched)
	return page, nil
}

func scanEntityRows(rows *sql.Rows, keep func(Track) bool) ([]Track, error) {
	defer rows.Close()
	var out []Track
	for rows.Next() {
		var t Track
		var liked int
		if err := rows.Scan(&t.Path, &t.Genre, &t.Title, &t.Artist, &t.Album, &t.Year, &t.Label,
			&t.Duration, &t.Art, &t.Waveform, &liked); err != nil {
			return nil, err
		}
		t.Liked = liked == 1
		cleanTrackText(&t)
		if keep(t) {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

// Newest release first, then album order on disk.
func sortEntityTracks(tracks []Track) {
	sort.SliceStable(tracks, func(i, j int) bool {
		a, b := tracks[i], tracks[j]
		if a.Year != b.Year {
			return a.Year > b.Year
		}
		da, db := filepath.Dir(a.Path), filepath.Dir(b.Path)
		if da != db {
			return da < db
		}
		return a.Path < b.Path
	})
}

func mostCommonSpelling(tracks []Track, field func(Track) string, fallback string) string {
	counts := map[string]int{}
	best, bestN := fallback, 0
	for _, t := range tracks {
		v := strings.TrimSpace(field(t))
		if v == "" {
			continue
		}
		counts[v]++
		if counts[v] > bestN {
			best, bestN = v, counts[v]
		}
	}
	return best
}

type refTally struct {
	order []string
	names map[string]string
	count map[string]int
	path  map[string]string
}

func newRefTally() *refTally {
	return &refTally{names: map[string]string{}, count: map[string]int{}, path: map[string]string{}}
}

func (r *refTally) add(name, path string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	key := strings.ToLower(name)
	if _, ok := r.count[key]; !ok {
		r.order = append(r.order, key)
		r.names[key] = name
		r.path[key] = path
	}
	r.count[key]++
}

func (r *refTally) addExact(name, path string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if _, ok := r.count[name]; !ok {
		r.order = append(r.order, name)
		r.names[name] = name
		r.path[name] = path
	}
	r.count[name]++
}

func (r *refTally) refs(skip string, limit int) []EntityRef {
	out := make([]EntityRef, 0, len(r.order))
	for _, key := range r.order {
		if key == skip {
			continue
		}
		out = append(out, EntityRef{Name: r.names[key], Count: r.count[key]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func summarizeEntity(env Env, page *EntityPage, tracks []Track) {
	page.Total = len(tracks)
	artists, labels, genres := newRefTally(), newRefTally(), newRefTally()
	type releaseKey struct{ dir, album string }
	releases := map[releaseKey]*EntityRelease{}
	var releaseOrder []releaseKey
	minYear, maxYear := "", ""
	for _, t := range tracks {
		page.Duration += t.Duration
		if t.Liked {
			page.Liked++
		}
		artists.add(t.Artist, t.Path)
		labels.add(t.Label, t.Path)
		genres.addExact(t.Genre, t.Path)
		if y := strings.TrimSpace(t.Year); len(y) == 4 {
			if minYear == "" || y < minYear {
				minYear = y
			}
			if y > maxYear {
				maxYear = y
			}
		}
		album := strings.TrimSpace(t.Album)
		if album == "" {
			continue
		}
		key := releaseKey{filepath.Dir(t.Path), strings.ToLower(album)}
		rel, ok := releases[key]
		if !ok {
			rel = &EntityRelease{Name: album, Year: t.Year, Dir: key.dir, Artist: t.Artist, Label: t.Label}
			releases[key] = rel
			releaseOrder = append(releaseOrder, key)
		}
		if rel.Artist != t.Artist {
			rel.Artist = "Various"
		}
		rel.Count++
	}
	switch {
	case minYear != "" && minYear != maxYear:
		page.Years = minYear + "–" + maxYear
	case minYear != "":
		page.Years = minYear
	}

	if len(tracks) > entityTrackLimit {
		tracks = tracks[:entityTrackLimit]
	}
	page.Tracks = make([]Track, len(tracks))
	firstInDir := map[string]int{}
	for i, t := range tracks {
		t.Art, t.Thumb = ListAssets(env, t.Path, t.Art)
		t.Waveform = resolveWaveform(env, t.Path, t.Waveform)
		t.DurationLabel = playback.FormatTime(t.Duration)
		page.Tracks[i] = t
		if page.Art == "" && t.Art != "" {
			page.Art, page.Thumb = t.Art, t.Thumb
		}
		if _, ok := firstInDir[filepath.Dir(t.Path)]; !ok && t.Art != "" {
			firstInDir[filepath.Dir(t.Path)] = i
		}
	}

	if len(releaseOrder) > entityReleaseLimit {
		releaseOrder = releaseOrder[:entityReleaseLimit]
	}
	page.Releases = make([]EntityRelease, 0, len(releaseOrder))
	for _, key := range releaseOrder {
		rel := releases[key]
		if i, ok := firstInDir[key.dir]; ok {
			rel.Art, rel.Thumb = page.Tracks[i].Art, page.Tracks[i].Thumb
		}
		page.Releases = append(page.Releases, *rel)
	}

	skipArtist, skipLabel := "", ""
	switch page.Kind {
	case "artist":
		skipArtist = strings.ToLower(page.Name)
	case "label":
		skipLabel = strings.ToLower(page.Name)
	}
	page.Artists = artists.refs(skipArtist, 24)
	page.Labels = labels.refs(skipLabel, 24)
	page.Genres = genres.refs("", 8)
}

var yearDigits = regexp.MustCompile(`\d{4}`)

// yearGroup keeps the first four-digit year in a tag such as "2005-06-01".
func yearGroup(raw string) string {
	return yearDigits.FindString(raw)
}

// Index lists every artist, album, label, genre, or year with a track count and a cover.
func Index(env Env, kind string) ([]EntityRef, error) {
	var col string
	switch kind {
	case "artist":
		col = "artist"
	case "album":
		col = "album"
	case "label":
		col = "label"
	case "genre":
		col = "genre"
	case "year":
		col = "year"
	default:
		return nil, fmt.Errorf("unknown kind: %s", kind)
	}
	db, err := EnsureDB(env)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT path, COALESCE(art,''), trim(COALESCE(` + col + `,'')) FROM tracks
WHERE trim(COALESCE(` + col + `,'')) != '' ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tally := newRefTally()
	stored := map[string]string{}
	exact := kind == "genre"
	for rows.Next() {
		var path, art, name string
		if err := rows.Scan(&path, &art, &name); err != nil {
			return nil, err
		}
		name = tags.TrimFrame(name)
		if name == "" {
			continue
		}
		if kind == "year" {
			name = yearGroup(name)
			if name == "" {
				continue
			}
		}
		if exact {
			tally.addExact(name, path)
		} else {
			tally.add(name, path)
		}
		if art != "" {
			stored[path] = art
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := tally.refs("", 0)
	var missing []string
	for i := range out {
		key := out[i].Name
		if !exact {
			key = strings.ToLower(key)
		}
		path := tally.path[key]
		art, thumb := ListAssets(env, path, stored[path])
		if art != "" {
			if cover := resolveCoverThumb(env, art); cover != "" {
				thumb = cover
			} else {
				missing = append(missing, art)
			}
		}
		out[i].Art, out[i].Thumb = art, thumb
	}
	queueCoverThumbs(env, missing)
	if kind == "year" {
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].Name > out[j].Name
		})
	} else if kind == "artist" || kind == "album" || kind == "label" {
		stars := artistStarSet(env)
		if kind == "album" {
			stars = albumStarSet(env)
		}
		if kind == "label" {
			stars = labelStarSet(env)
		}
		for i := range out {
			out[i].Starred = stars[strings.ToLower(out[i].Name)]
		}
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Starred != out[j].Starred {
				return out[i].Starred
			}
			return indexSortKey(out[i].Name) < indexSortKey(out[j].Name)
		})
	} else {
		sort.SliceStable(out, func(i, j int) bool {
			return indexSortKey(out[i].Name) < indexSortKey(out[j].Name)
		})
	}
	return out, nil
}

// indexSortKey files "$AW" under A and "The Prodigy" under P.
func indexSortKey(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	key = strings.TrimPrefix(key, "the ")
	key = strings.TrimLeftFunc(key, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if key == "" {
		return "\uffff" + strings.ToLower(name)
	}
	return key
}
