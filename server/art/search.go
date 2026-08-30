package art

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/paths"
)

const (
	maxResults           = 24
	maxReleaseSearchRows = 8
	maxImagesPerRelease  = 8
	maxArtistResults     = 2
	userAgent            = "evoplayer/1.0 (local music player)"
)

type Result struct {
	URL    string `json:"url"`
	Thumb  string `json:"thumb"`
	Label  string `json:"label"`
	Source string `json:"source"`
	Year   string `json:"year,omitempty"`
	Catno  string `json:"catno,omitempty"`
}

type SearchResponse struct {
	Query   string   `json:"query"`
	Results []Result `json:"results"`
}

func SearchTrack(env paths.Env, trackPath string) (SearchResponse, error) {
	_ = env
	if !audio.IsAudio(trackPath) {
		return SearchResponse{}, fmt.Errorf("evoplayer: not an audio file: %s", trackPath)
	}
	loadSecrets()
	meta, err := readTrackMeta(trackPath)
	if err != nil {
		return SearchResponse{}, err
	}
	primary := discogsJoinQuery(meta.Catno, meta.Title, meta.Artist)
	if primary == "" {
		primary = strings.TrimSuffix(filepath.Base(trackPath), filepath.Ext(trackPath))
	}
	used := primary
	var results []Result
	if meta.Catno != "" {
		catnoHits := searchDiscogs("", meta.Catno, meta.Year, "release")
		primaryHits := searchDiscogsQuery(primary)
		results = dedupe(append(catnoHits, primaryHits...))
	} else {
		results = dedupe(searchDiscogsQuery(primary))
	}
	if len(results) == 0 && meta.Artist != "" && meta.Artist != primary {
		used = meta.Artist
		results = dedupe(searchDiscogsQuery(meta.Artist))
	}
	return SearchResponse{Query: used, Results: results}, nil
}

func SearchQuery(query string) SearchResponse {
	loadSecrets()
	q := strings.TrimSpace(query)
	return SearchResponse{Query: q, Results: dedupe(searchDiscogsQuery(q))}
}

func dedupe(rows []Result) []Result {
	seen := map[string]struct{}{}
	out := make([]Result, 0, maxResults)
	for _, r := range rows {
		if r.URL == "" {
			continue
		}
		if _, ok := seen[r.URL]; ok {
			continue
		}
		seen[r.URL] = struct{}{}
		out = append(out, r)
		if len(out) >= maxResults {
			break
		}
	}
	return out
}

func httpDo(req *http.Request) ([]byte, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", userAgent)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return io.ReadAll(resp.Body)
	}
	return io.ReadAll(resp.Body)
}
