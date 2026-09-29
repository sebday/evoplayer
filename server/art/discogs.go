package art

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const PreviewSize = 600

func discogsToken() string {
	return strings.TrimSpace(os.Getenv("DISCOGS_TOKEN"))
}

var (
	discogsFitInRe      = regexp.MustCompile(`/fit-in/[0-9]+x[0-9]+/`)
	discogsReleaseIDRe  = regexp.MustCompile(`discogs\.com/release/([0-9]+)`)
	discogsReleaseAPIRe = regexp.MustCompile(`api\.discogs\.com/releases/([0-9]+)`)
	discogsArtistIDRe   = regexp.MustCompile(`discogs\.com/artists?/([0-9]+)`)
	discogsArtistAPIRe  = regexp.MustCompile(`api\.discogs\.com/artists/([0-9]+)`)
)

// sizedDiscogsURL only upgrades legacy /fit-in/WxH/ thumbs. Signed i.discogs.com
// URLs (rs:fit/...) must not be rewritten or the CDN returns 403.
func sizedDiscogsURL(raw string, size int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return ""
	}
	if !discogsFitInRe.MatchString(raw) {
		return raw
	}
	fit := fmt.Sprintf("/fit-in/%dx%d/", size, size)
	return discogsFitInRe.ReplaceAllString(raw, fit)
}

type discogsSearchRow struct {
	Title       string `json:"title"`
	Year        string `json:"year"`
	Catno       string `json:"catno"`
	Thumb       string `json:"thumb"`
	ResourceURL string `json:"resource_url"`
}

func discogsArtistIDFromQuery(q string) string {
	for _, re := range []*regexp.Regexp{discogsArtistIDRe, discogsArtistAPIRe} {
		if m := re.FindStringSubmatch(q); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func discogsReleaseIDFromQuery(q string) string {
	for _, re := range []*regexp.Regexp{discogsReleaseIDRe, discogsReleaseAPIRe} {
		if m := re.FindStringSubmatch(q); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func searchDiscogsAll(query string) []Result {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if id := discogsReleaseIDFromQuery(query); id != "" {
		return searchDiscogsReleaseID(id)
	}
	if id := discogsArtistIDFromQuery(query); id != "" {
		return searchDiscogsArtistID(id)
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("per_page", "100")
	body, err := discogsGet("https://api.discogs.com/database/search?" + params.Encode())
	if err != nil {
		return nil
	}
	var payload struct {
		Results []struct {
			Title       string `json:"title"`
			Year        any    `json:"year"`
			Catno       any    `json:"catno"`
			Thumb       string `json:"thumb"`
			CoverImage  string `json:"cover_image"`
			ResourceURL string `json:"resource_url"`
			Type        string `json:"type"`
		} `json:"results"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	direct := make([]Result, 0, len(payload.Results))
	follow := make([]discogsSearchRow, 0)
	for _, r := range payload.Results {
		row := discogsSearchRow{
			Title:       strings.TrimSpace(r.Title),
			Year:        discogsText(r.Year),
			Catno:       discogsText(r.Catno),
			Thumb:       strings.TrimSpace(r.Thumb),
			ResourceURL: strings.TrimSpace(r.ResourceURL),
		}
		if row.Title == "" {
			row.Title = strings.TrimSpace(r.Type)
		}
		artURL := usableArtURL(r.CoverImage)
		if artURL == "" {
			artURL = usableArtURL(r.Thumb)
		}
		if artURL != "" {
			thumb := usableArtURL(r.Thumb)
			if thumb == "" {
				thumb = artURL
			}
			direct = append(direct, Result{
				URL:    artURL,
				Thumb:  thumb,
				Label:  row.Title,
				Source: "discogs",
				Year:   row.Year,
				Catno:  row.Catno,
			})
			continue
		}
		if row.ResourceURL != "" {
			follow = append(follow, row)
		}
	}
	if len(direct) > 0 {
		return append(direct, fetchDiscogsImages(follow)...)
	}
	return fetchDiscogsImages(follow)
}

// Discogs hides images on an unauthenticated search, so those hits are loaded
// from the result page. The cap stays inside the public 25 requests/minute limit.
const openSearchFetch = 16

func fetchDiscogsImages(rows []discogsSearchRow) []Result {
	if len(rows) > openSearchFetch {
		rows = rows[:openSearchFetch]
	}
	if len(rows) == 0 {
		return nil
	}
	parts := make([][]Result, len(rows))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, row := range rows {
		wg.Add(1)
		go func(i int, row discogsSearchRow) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			parts[i] = discogsReleaseImages(row.ResourceURL, row)
		}(i, row)
	}
	wg.Wait()
	out := make([]Result, 0, len(rows))
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func usableArtURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || strings.Contains(raw, "spacer.gif") {
		return ""
	}
	return raw
}

func discogsText(v any) string {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "null" || s == "<nil>" {
			return ""
		}
		return s
	case float64:
		if t == 0 {
			return ""
		}
		return strconv.Itoa(int(t))
	default:
		return ""
	}
}

func searchDiscogsQuery(query string) []Result {
	if id := discogsReleaseIDFromQuery(query); id != "" {
		return searchDiscogsReleaseID(id)
	}
	if id := discogsArtistIDFromQuery(query); id != "" {
		return searchDiscogsArtistID(id)
	}
	var artists []Result
	artistSearch, _ := discogsGet("https://api.discogs.com/database/search?type=artist&per_page=8&q=" + url.QueryEscape(query))
	var artistPayload struct {
		Results []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"results"`
	}
	_ = json.Unmarshal(artistSearch, &artistPayload)
	want := strings.ToLower(strings.TrimSpace(query))
	var artistID int
	for _, r := range artistPayload.Results {
		if r.ID == 0 {
			continue
		}
		if strings.ToLower(strings.TrimSpace(r.Title)) == want {
			artistID = r.ID
			break
		}
	}
	if artistID == 0 && len(artistPayload.Results) > 0 {
		artistID = artistPayload.Results[0].ID
	}
	if artistID != 0 {
		artists = searchDiscogsArtistID(strconv.Itoa(artistID))
	}
	releases := searchDiscogs(query, "", "", "release")
	if len(artists) > maxArtistResults {
		artists = artists[:maxArtistResults]
	}
	return append(releases, artists...)
}

func searchDiscogsReleaseID(id string) []Result {
	if id == "" {
		return nil
	}
	return discogsReleaseImages("https://api.discogs.com/releases/"+id, discogsSearchRow{})
}

func searchDiscogsArtistID(id string) []Result {
	if id == "" {
		return nil
	}
	body, _ := discogsGet("https://api.discogs.com/artists/" + id)
	var payload struct {
		Name   string `json:"name"`
		Images []struct {
			URI    string `json:"uri"`
			URI150 string `json:"uri150"`
		} `json:"images"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	name := payload.Name
	if name == "" {
		name = "artist"
	}
	out := make([]Result, 0, len(payload.Images))
	for _, img := range payload.Images {
		u := strings.TrimSpace(img.URI)
		if u == "" {
			continue
		}
		thumb := strings.TrimSpace(img.URI150)
		if thumb == "" {
			thumb = u
		}
		out = append(out, Result{
			URL:    u,
			Thumb:  thumb,
			Label:  name,
			Source: "discogs",
		})
		if len(out) >= maxResults {
			break
		}
	}
	return out
}

func searchDiscogs(query, catno, year, kind string) []Result {
	if kind != "artist" && kind != "release" {
		kind = "release"
	}
	if kind == "artist" {
		catno = ""
		year = ""
	}
	if query == "" && catno == "" {
		return nil
	}
	params := url.Values{}
	params.Set("type", kind)
	params.Set("per_page", "8")
	if catno != "" {
		params.Set("catno", catno)
	}
	if year != "" {
		params.Set("year", year)
	}
	if query != "" {
		params.Set("q", query)
	}
	body, err := discogsGet("https://api.discogs.com/database/search?" + params.Encode())
	if err != nil {
		return nil
	}
	var payload struct {
		Results []struct {
			Title       string `json:"title"`
			Year        any    `json:"year"`
			Catno       any    `json:"catno"`
			Thumb       string `json:"thumb"`
			CoverImage  string `json:"cover_image"`
			ResourceURL string `json:"resource_url"`
		} `json:"results"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	rows := make([]discogsSearchRow, 0, len(payload.Results))
	for _, r := range payload.Results {
		thumb := r.Thumb
		if thumb == "" {
			thumb = r.CoverImage
		}
		rows = append(rows, discogsSearchRow{
			Title:       r.Title,
			Year:        fmt.Sprint(r.Year),
			Catno:       fmt.Sprint(r.Catno),
			Thumb:       thumb,
			ResourceURL: r.ResourceURL,
		})
	}
	want := normCatno(catno)
	if want != "" {
		exact := make([]discogsSearchRow, 0)
		for _, row := range rows {
			if normCatno(row.Catno) == want {
				exact = append(exact, row)
			}
		}
		if len(exact) > 0 {
			rows = exact
		}
	}
	out := make([]Result, 0, len(rows)*2)
	limit := len(rows)
	if limit > maxReleaseSearchRows {
		limit = maxReleaseSearchRows
	}
	for i := 0; i < limit; i++ {
		row := rows[i]
		if row.ResourceURL != "" {
			releaseImages := discogsReleaseImages(row.ResourceURL, row)
			if len(releaseImages) > 0 {
				out = append(out, releaseImages...)
				continue
			}
		}
		artURL := ""
		thumb := row.Thumb
		if thumb != "" && thumb != "null" {
			artURL = sizedDiscogsURL(thumb, PreviewSize)
		}
		if artURL == "" {
			continue
		}
		if thumb == "" {
			thumb = artURL
		}
		label := row.Title
		if label == "" {
			label = strings.TrimSpace(catno + query)
		}
		out = append(out, Result{
			URL:    artURL,
			Thumb:  thumb,
			Label:  label,
			Source: "discogs",
			Year:   row.Year,
			Catno:  row.Catno,
		})
	}
	return out
}

func discogsReleaseImages(resourceURL string, row discogsSearchRow) []Result {
	if resourceURL == "" {
		return nil
	}
	body, err := discogsGet(resourceURL)
	if err != nil {
		return nil
	}
	var payload struct {
		Title  string `json:"title"`
		Year   any    `json:"year"`
		Images []struct {
			Type   string `json:"type"`
			URI    string `json:"uri"`
			URI150 string `json:"uri150"`
		} `json:"images"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	label := strings.TrimSpace(row.Title)
	if label == "" {
		label = strings.TrimSpace(payload.Title)
	}
	if label == "" {
		label = "release"
	}
	year := strings.TrimSpace(row.Year)
	if year == "" {
		year = strings.TrimSpace(fmt.Sprint(payload.Year))
	}
	catno := strings.TrimSpace(row.Catno)
	out := make([]Result, 0, len(payload.Images))
	for i, img := range payload.Images {
		if i >= maxImagesPerRelease {
			break
		}
		u := strings.TrimSpace(img.URI)
		if u == "" {
			continue
		}
		thumb := strings.TrimSpace(img.URI150)
		if thumb == "" {
			thumb = u
		}
		imgLabel := label
		if t := strings.TrimSpace(img.Type); t != "" && t != "primary" {
			imgLabel = label + " (" + t + ")"
		}
		out = append(out, Result{
			URL:    u,
			Thumb:  thumb,
			Label:  imgLabel,
			Source: "discogs",
			Year:   year,
			Catno:  catno,
		})
	}
	return out
}

func discogsGet(u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if token := discogsToken(); token != "" {
		req.Header.Set("Authorization", "Discogs token="+token)
	}
	body, err := httpDo(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: discogs: %v\n", err)
	}
	return body, err
}
