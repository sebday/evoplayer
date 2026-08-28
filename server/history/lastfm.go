package history

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const lastfmBase = "https://ws.audioscrobbler.com/2.0/"

var lastfmTextRe = regexp.MustCompile(`['"]#text['"]\s*:\s*['"]([^'"]*)['"]`)

func fetchJSON(rawURL string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "evo-player/1.0 (local history)")
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func lastfm(apiKey, method string, params map[string]string) (map[string]any, error) {
	q := url.Values{}
	q.Set("method", method)
	q.Set("api_key", apiKey)
	q.Set("format", "json")
	for k, v := range params {
		q.Set(k, v)
	}
	return fetchJSON(lastfmBase + "?" + q.Encode())
}

func lastfmFieldText(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case map[string]any:
		for _, key := range []string{"#text", "name"} {
			if text, ok := v[key]; ok {
				s := strings.TrimSpace(fmt.Sprint(text))
				if s != "" {
					return s
				}
			}
		}
		return ""
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		if strings.HasPrefix(text, "{") && strings.Contains(text, "#text") {
			if m := lastfmTextRe.FindStringSubmatch(text); len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
		}
		return text
	}
}

func chartRows(body map[string]any, key, nameKey, countKey string) []NamedCount {
	if countKey == "" {
		countKey = "playcount"
	}
	rowsVal, _ := body[key]
	var rows []map[string]any
	switch v := rowsVal.(type) {
	case map[string]any:
		rows = []map[string]any{v}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				rows = append(rows, m)
			}
		}
	}
	out := make([]NamedCount, 0, len(rows))
	for _, row := range rows {
		name := lastfmFieldText(row[nameKey])
		if name == "" {
			name = lastfmFieldText(row["name"])
		}
		if name == "" {
			continue
		}
		count := 0
		if raw, ok := row[countKey]; ok {
			count = intFromAny(raw)
		} else if attr, ok := row["@attr"].(map[string]any); ok {
			count = intFromAny(attr["count"])
		}
		out = append(out, NamedCount{Name: name, Count: count})
	}
	return out
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return int(f)
	case int:
		return n
	default:
		return 0
	}
}

func lastfmOverallReport(apiKey, user string, limit int) (*Report, error) {
	infoData, err := lastfm(apiKey, "user.getInfo", map[string]string{"user": user})
	if err != nil {
		return nil, err
	}
	userInfo, _ := infoData["user"].(map[string]any)
	playcount := intFromAny(userInfo["playcount"])
	artistCount := intFromAny(userInfo["artist_count"])
	albumCount := intFromAny(userInfo["album_count"])
	trackCount := intFromAny(userInfo["track_count"])

	data, err := lastfm(apiKey, "user.getTopArtists", map[string]string{"user": user, "period": "overall", "limit": strconv.Itoa(limit)})
	if err != nil {
		return nil, err
	}
	topArtistsBody, _ := data["topartists"].(map[string]any)
	artistRows := chartRows(topArtistsBody, "artist", "name", "playcount")

	data, err = lastfm(apiKey, "user.getTopAlbums", map[string]string{"user": user, "period": "overall", "limit": strconv.Itoa(limit)})
	if err != nil {
		return nil, err
	}
	topAlbumsBody, _ := data["topalbums"].(map[string]any)
	albumRows := chartRows(topAlbumsBody, "album", "name", "playcount")

	data, err = lastfm(apiKey, "user.getTopTracks", map[string]string{"user": user, "period": "overall", "limit": strconv.Itoa(limit)})
	if err != nil {
		return nil, err
	}
	topTracksBody, _ := data["toptracks"].(map[string]any)
	rawTracks := asMapSlice(topTracksBody["track"])
	trackRows := make([]TrackCount, 0, len(rawTracks))
	for _, row := range rawTracks {
		artist := lastfmFieldText(row["artist"])
		title := lastfmFieldText(row["name"])
		count := intFromAny(row["playcount"])
		if artist != "" && title != "" {
			trackRows = append(trackRows, TrackCount{Artist: artist, Title: title, Count: count})
		}
	}

	hours := round1(float64(playcount) * 3.5 / 60.0)

	recentLimit := limit
	if recentLimit > 50 {
		recentLimit = 50
	}
	recentData, err := lastfm(apiKey, "user.getRecentTracks", map[string]string{"user": user, "limit": strconv.Itoa(recentLimit)})
	if err != nil {
		return nil, err
	}
	recentTracksBody, _ := recentData["recenttracks"].(map[string]any)
	recentRaw := asMapSlice(recentTracksBody["track"])
	recent := make([]map[string]any, 0, len(recentRaw))
	for _, row := range recentRaw {
		artist := lastfmFieldText(row["artist"])
		title := lastfmFieldText(row["name"])
		album := lastfmFieldText(row["album"])
		if artist == "" || title == "" {
			continue
		}
		at := ""
		if date, ok := row["date"].(map[string]any); ok {
			at = lastfmFieldText(date["#text"])
		}
		recent = append(recent, map[string]any{
			"artist": artist,
			"title":  title,
			"album":  album,
			"at":     at,
		})
	}
	if len(recent) > limit {
		recent = recent[:limit]
	}
	if len(artistRows) > limit {
		artistRows = artistRows[:limit]
	}
	if len(albumRows) > limit {
		albumRows = albumRows[:limit]
	}
	if len(trackRows) > limit {
		trackRows = trackRows[:limit]
	}

	artists := artistCount
	if artists == 0 {
		artists = len(artistRows)
	}
	albums := albumCount
	if albums == 0 {
		albums = len(albumRows)
	}
	tracks := trackCount
	if tracks == 0 {
		tracks = len(trackRows)
	}

	return &Report{
		Source: "lastfm",
		Period: "overall",
		Week:   WeekInfo{From: 0, To: 0, Label: "All time"},
		Totals: Totals{
			Scrobbles: playcount,
			Artists:   artists,
			Albums:    albums,
			Tracks:    tracks,
			Hours:     hours,
		},
		Daily:      []DailyCount{},
		TopArtists: artistRows,
		TopAlbums:  albumRows,
		TopTracks:  trackRows,
		Recent:     recent,
	}, nil
}

func lastfmWeekReport(apiKey, user string, startTS, endTS, limit int) (*Report, error) {
	fromStr := strconv.Itoa(startTS)
	data, err := lastfm(apiKey, "user.getWeeklyArtistChart", map[string]string{"user": user, "from": fromStr})
	if err != nil {
		return nil, err
	}
	artistsBody, _ := data["weeklyartistchart"].(map[string]any)
	artistRows := chartRows(artistsBody, "artist", "name", "playcount")

	data, err = lastfm(apiKey, "user.getWeeklyAlbumChart", map[string]string{"user": user, "from": fromStr})
	if err != nil {
		return nil, err
	}
	albumsBody, _ := data["weeklyalbumchart"].(map[string]any)
	albumRows := chartRows(albumsBody, "album", "name", "playcount")

	data, err = lastfm(apiKey, "user.getWeeklyTrackChart", map[string]string{"user": user, "from": fromStr})
	if err != nil {
		return nil, err
	}
	trackBody, _ := data["weeklytrackchart"].(map[string]any)
	rawTracks := asMapSlice(trackBody["track"])
	trackRows := make([]TrackCount, 0, len(rawTracks))
	scrobbles := 0
	for _, row := range rawTracks {
		artist := lastfmFieldText(row["artist"])
		title := lastfmFieldText(row["name"])
		count := intFromAny(row["playcount"])
		if artist != "" && title != "" {
			trackRows = append(trackRows, TrackCount{Artist: artist, Title: title, Count: count})
			scrobbles += count
		}
	}

	daily := make([]DailyCount, 0, 7)
	for _, d := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"} {
		daily = append(daily, DailyCount{Day: d, Count: 0})
	}
	hours := round1(float64(scrobbles) * 3.5 / 60.0)

	recentLimit := limit
	if recentLimit > 50 {
		recentLimit = 50
	}
	recentData, err := lastfm(apiKey, "user.getRecentTracks", map[string]string{"user": user, "limit": strconv.Itoa(recentLimit)})
	if err != nil {
		return nil, err
	}
	recentTracksBody, _ := recentData["recenttracks"].(map[string]any)
	recentRaw := asMapSlice(recentTracksBody["track"])
	recent := make([]map[string]any, 0, len(recentRaw))
	for _, row := range recentRaw {
		artist := lastfmFieldText(row["artist"])
		title := lastfmFieldText(row["name"])
		album := lastfmFieldText(row["album"])
		if artist == "" || title == "" {
			continue
		}
		at := ""
		if date, ok := row["date"].(map[string]any); ok {
			at = lastfmFieldText(date["#text"])
		}
		recent = append(recent, map[string]any{
			"artist": artist,
			"title":  title,
			"album":  album,
			"at":     at,
		})
	}
	if len(recent) > limit {
		recent = recent[:limit]
	}
	if len(artistRows) > 10 {
		artistRows = artistRows[:10]
	}
	if len(albumRows) > 10 {
		albumRows = albumRows[:10]
	}
	if len(trackRows) > 10 {
		trackRows = trackRows[:10]
	}

	return &Report{
		Source: "lastfm",
		Week:   WeekInfo{From: startTS, To: endTS, Label: weekLabel(startTS, endTS)},
		Totals: Totals{
			Scrobbles: scrobbles,
			Artists:   len(artistRows),
			Albums:    len(albumRows),
			Tracks:    len(trackRows),
			Hours:     hours,
		},
		Daily:      daily,
		TopArtists: artistRows,
		TopAlbums:  albumRows,
		TopTracks:  trackRows,
		Recent:     recent,
	}, nil
}

func availableWeeks(apiKey, user string) ([]WeekInfo, error) {
	data, err := lastfm(apiKey, "user.getWeeklyChartList", map[string]string{"user": user})
	if err != nil {
		return nil, err
	}
	listBody, _ := data["weeklychartlist"].(map[string]any)
	weeksRaw := asMapSlice(listBody["chart"])
	out := make([]WeekInfo, 0, len(weeksRaw))
	for _, w := range weeksRaw {
		startTS := intFromAny(w["from"])
		endTS := intFromAny(w["to"])
		if startTS <= 0 || endTS <= 0 {
			continue
		}
		out = append(out, WeekInfo{
			From:  startTS,
			To:    endTS,
			Label: weekLabel(startTS, endTS),
		})
	}
	return out, nil
}

func asMapSlice(v any) []map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return []map[string]any{t}
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}
