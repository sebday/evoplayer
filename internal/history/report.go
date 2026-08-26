package history

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Generate(p Params) (*Report, error) {
	if p.Limit <= 0 {
		p.Limit = 12
	}
	allTime := p.WeekFrom == "" || p.WeekFrom == "0" || p.WeekFrom == "overall"
	if allTime {
		if cached, err := loadCachedOverall(p.CacheDir); err == nil && cached != nil {
			cached.User = p.User
			attachRecap(cached, p.CacheDir)
			return cached, nil
		}
		var report *Report
		if p.APIKey != "" && p.User != "" {
			r, err := lastfmOverallReport(p.APIKey, p.User, p.Limit)
			if err != nil {
				report = scrobbleFallbackAlltime(p.ScrobbleLog, p.Limit)
			} else {
				report = r
			}
		} else {
			report = scrobbleFallbackAlltime(p.ScrobbleLog, p.Limit)
		}
		report.User = p.User
		attachRecap(report, p.CacheDir)
		if err := os.MkdirAll(p.CacheDir, 0o755); err != nil {
			return nil, err
		}
		if err := writeJSON(filepath.Join(p.CacheDir, "overall.json"), report); err != nil {
			return nil, err
		}
		return report, nil
	}

	weeks := []WeekInfo{}
	if p.APIKey != "" && p.User != "" {
		if w, err := availableWeeks(p.APIKey, p.User); err == nil {
			weeks = w
		}
	}

	startTS, _ := strconv.Atoi(p.WeekFrom)
	if startTS == 0 && len(weeks) > 0 {
		startTS = weeks[0].From
	}
	if startTS == 0 {
		now := time.Now()
		offset := int(now.Weekday()) - int(time.Monday)
		if offset < 0 {
			offset += 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -offset)
		startTS = int(start.Unix())
	}
	endTS := startTS + 7*86400

	cachePath := filepath.Join(p.CacheDir, "week-"+strconv.Itoa(startTS)+".json")
	if cached, err := loadCachedReport(p.CacheDir, startTS); err == nil && cached != nil {
		cached.AvailableWeeks = weeks
		attachRecap(cached, p.CacheDir)
		return cached, nil
	}
	_ = cachePath

	var report *Report
	if p.APIKey != "" && p.User != "" {
		r, err := lastfmWeekReport(p.APIKey, p.User, startTS, endTS, p.Limit)
		if err != nil {
			report = scrobbleFallback(p.ScrobbleLog, startTS, endTS, p.Limit)
		} else {
			report = r
		}
	} else {
		report = scrobbleFallback(p.ScrobbleLog, startTS, endTS, p.Limit)
	}
	report.User = p.User
	report.AvailableWeeks = weeks
	attachRecap(report, p.CacheDir)
	if err := os.MkdirAll(p.CacheDir, 0o755); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(p.CacheDir, "week-"+strconv.Itoa(startTS)+".json"), report); err != nil {
		return nil, err
	}
	return report, nil
}

func emptyReportBody(
	startTS, endTS int,
	source string,
	recent []map[string]any,
	daily map[string]int,
	artists map[string]int,
	albums map[string]int,
	tracks map[[2]string]int,
	hours float64,
	overall bool,
) *Report {
	scrobbles := 0
	if overall {
		for _, c := range tracks {
			scrobbles += c
		}
	} else {
		for _, c := range daily {
			scrobbles += c
		}
	}
	dailyOut := make([]DailyCount, 0, 7)
	for _, d := range weekdayOrder {
		dailyOut = append(dailyOut, DailyCount{Day: d, Count: daily[d]})
	}
	topArtists := topNamedCounts(artists, 10)
	topAlbums := topNamedCounts(albums, 10)
	topTracks := topTrackCounts(tracks, 10)
	body := &Report{
		Source:     source,
		Totals:     Totals{Scrobbles: scrobbles, Artists: len(artists), Albums: len(albums), Tracks: len(tracks), Hours: round1(hours)},
		Daily:      dailyOut,
		TopArtists: topArtists,
		TopAlbums:  topAlbums,
		TopTracks:  topTracks,
		Recent:     recent,
	}
	if overall {
		body.Period = "overall"
		body.Week = WeekInfo{From: 0, To: 0, Label: "All time"}
		return body
	}
	body.Week = WeekInfo{From: startTS, To: endTS, Label: weekLabel(startTS, endTS)}
	return body
}

func topNamedCounts(counts map[string]int, limit int) []NamedCount {
	type pair struct {
		name  string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for name, count := range counts {
		pairs = append(pairs, pair{name, count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].name < pairs[j].name
		}
		return pairs[i].count > pairs[j].count
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]NamedCount, len(pairs))
	for i, p := range pairs {
		out[i] = NamedCount{Name: p.name, Count: p.count}
	}
	return out
}

func topTrackCounts(counts map[[2]string]int, limit int) []TrackCount {
	type pair struct {
		artist string
		title  string
		count  int
	}
	pairs := make([]pair, 0, len(counts))
	for key, count := range counts {
		pairs = append(pairs, pair{key[0], key[1], count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			if pairs[i].artist == pairs[j].artist {
				return pairs[i].title < pairs[j].title
			}
			return pairs[i].artist < pairs[j].artist
		}
		return pairs[i].count > pairs[j].count
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]TrackCount, len(pairs))
	for i, p := range pairs {
		out[i] = TrackCount{Artist: p.artist, Title: p.title, Count: p.count}
	}
	return out
}

func weekLabel(startTS, endTS int) string {
	start := time.Unix(int64(startTS), 0)
	end := time.Unix(int64(endTS-1), 0)
	if start.Year() == end.Year() {
		return strconv.Itoa(start.Day()) + " " + start.Format("Jan") + " – " + strconv.Itoa(end.Day()) + " " + end.Format("Jan 2006")
	}
	return strconv.Itoa(start.Day()) + " " + start.Format("Jan 2006") + " – " + strconv.Itoa(end.Day()) + " " + end.Format("Jan 2006")
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func sanitizeTrackRows(rows []TrackCount) []TrackCount {
	out := make([]TrackCount, 0, len(rows))
	for _, row := range rows {
		artist := strings.TrimSpace(row.Artist)
		title := strings.TrimSpace(row.Title)
		if artist != "" && title != "" {
			out = append(out, TrackCount{Artist: artist, Title: title, Count: row.Count})
		}
	}
	return out
}

func loadCachedReport(cacheDir string, startTS int) (*Report, error) {
	if startTS <= 0 {
		return nil, os.ErrNotExist
	}
	path := filepath.Join(cacheDir, "week-"+strconv.Itoa(startTS)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	if report.Week.From == 0 && report.Week.Label == "" {
		return nil, errors.New("invalid cached report")
	}
	report.TopTracks = sanitizeTrackRows(report.TopTracks)
	return &report, nil
}

func loadCachedOverall(cacheDir string) (*Report, error) {
	path := filepath.Join(cacheDir, "overall.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	if report.Period != "overall" {
		return nil, errors.New("invalid overall cache")
	}
	report.TopTracks = sanitizeTrackRows(report.TopTracks)
	return &report, nil
}

func deltaPct(current, prev int) int {
	if prev <= 0 {
		if current > 0 {
			return 100
		}
		return 0
	}
	return int(math.Round(float64(current-prev) / float64(prev) * 100))
}

func peakForMetric(cacheDir, metric string, current, currentStart, scanLimit int) bool {
	if current <= 0 {
		return false
	}
	best := current
	info, err := os.ReadDir(cacheDir)
	if err != nil {
		return true
	}
	names := make([]string, 0)
	for _, entry := range info {
		name := entry.Name()
		if strings.HasPrefix(name, "week-") && strings.HasSuffix(name, ".json") {
			names = append(names, name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	seen := 0
	for _, name := range names {
		ts, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "week-"), ".json"))
		if err != nil {
			continue
		}
		if ts == currentStart {
			continue
		}
		cached, err := loadCachedReport(cacheDir, ts)
		if err != nil || cached == nil {
			continue
		}
		val := 0
		switch metric {
		case "artists":
			val = cached.Totals.Artists
		case "albums":
			val = cached.Totals.Albums
		case "tracks":
			val = cached.Totals.Tracks
		}
		if val > best {
			return false
		}
		seen++
		if seen >= scanLimit {
			break
		}
	}
	return true
}

func recapTopItem(report *Report, key string) RecapTop {
	var rows any
	switch key {
	case "top_artists":
		rows = report.TopArtists
	case "top_albums":
		rows = report.TopAlbums
	case "top_tracks":
		rows = report.TopTracks
	}
	switch key {
	case "top_tracks":
		if len(report.TopTracks) == 0 {
			return RecapTop{}
		}
		row := report.TopTracks[0]
		artist := strings.TrimSpace(row.Artist)
		title := strings.TrimSpace(row.Title)
		name := strings.Trim(strings.TrimSpace(artist+" — "+title), "—")
		name = strings.TrimSpace(name)
		return RecapTop{Name: name, Artist: artist, Title: title, Count: row.Count}
	default:
		var row NamedCount
		switch v := rows.(type) {
		case []NamedCount:
			if len(v) == 0 {
				return RecapTop{}
			}
			row = v[0]
		}
		return RecapTop{Name: strings.TrimSpace(row.Name), Count: row.Count}
	}
}

func buildOverallRecap(report *Report) map[string]RecapItem {
	recap := map[string]RecapItem{}
	for _, spec := range []struct {
		kind, metric, topKey string
	}{
		{"artists", "artists", "top_artists"},
		{"albums", "albums", "top_albums"},
		{"tracks", "tracks", "top_tracks"},
	} {
		count := 0
		switch spec.metric {
		case "artists":
			count = report.Totals.Artists
		case "albums":
			count = report.Totals.Albums
		case "tracks":
			count = report.Totals.Tracks
		}
		recap[spec.kind] = RecapItem{
			Count:    count,
			Prev:     0,
			DeltaPct: 0,
			Up:       true,
			Peak:     false,
			AllTime:  true,
			Top:      recapTopItem(report, spec.topKey),
		}
	}
	return recap
}

func buildRecap(report *Report, cacheDir string, prevReport *Report) map[string]RecapItem {
	prevTotals := Totals{}
	if prevReport != nil {
		prevTotals = prevReport.Totals
	}
	startTS := report.Week.From
	recap := map[string]RecapItem{}
	for _, spec := range []struct {
		kind, metric, topKey string
	}{
		{"artists", "artists", "top_artists"},
		{"albums", "albums", "top_albums"},
		{"tracks", "tracks", "top_tracks"},
	} {
		count, prev := 0, 0
		switch spec.metric {
		case "artists":
			count = report.Totals.Artists
			prev = prevTotals.Artists
		case "albums":
			count = report.Totals.Albums
			prev = prevTotals.Albums
		case "tracks":
			count = report.Totals.Tracks
			prev = prevTotals.Tracks
		}
		delta := deltaPct(count, prev)
		recap[spec.kind] = RecapItem{
			Count:    count,
			Prev:     prev,
			DeltaPct: delta,
			Up:       delta >= 0,
			Peak:     peakForMetric(cacheDir, spec.metric, count, startTS, 12),
			Top:      recapTopItem(report, spec.topKey),
		}
	}
	return recap
}

func attachRecap(report *Report, cacheDir string) {
	report.TopTracks = sanitizeTrackRows(report.TopTracks)
	if report.Period == "overall" {
		report.Recap = buildOverallRecap(report)
		return
	}
	startTS := report.Week.From
	prevReport, _ := loadCachedReport(cacheDir, startTS-7*86400)
	report.Recap = buildRecap(report, cacheDir, prevReport)
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
