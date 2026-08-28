package art

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	discogsFitInRe      = regexp.MustCompile(`/fit-in/[0-9]+x[0-9]+/`)
	discogsReleaseIDRe  = regexp.MustCompile(`discogs\.com/release/([0-9]+)`)
	discogsReleaseAPIRe = regexp.MustCompile(`api\.discogs\.com/releases/([0-9]+)`)
)

type discogsSearchRow struct {
	Title       string `json:"title"`
	Year        string `json:"year"`
	Catno       string `json:"catno"`
	Thumb       string `json:"thumb"`
	ResourceURL string `json:"resource_url"`
}

func discogsArtistIDFromQuery(q string) string {
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`discogs\.com/artists?/([0-9]+)`),
		regexp.MustCompile(`api\.discogs\.com/artists/([0-9]+)`),
	} {
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
			artURL = discogsFitInRe.ReplaceAllString(thumb, "/fit-in/600x600/")
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
	return httpDo(req)
}
