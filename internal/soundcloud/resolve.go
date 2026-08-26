package soundcloud

import (
	"fmt"
	"net/url"
	"strings"
)

type Transcoding struct {
	URL    string `json:"url"`
	Format struct {
		Protocol string `json:"protocol"`
		MimeType string `json:"mime_type"`
	} `json:"format"`
	Quality string `json:"quality"`
}

type Track struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	PermalinkURL string `json:"permalink_url"`
	CreatedAt    string `json:"created_at"`
	ArtworkURL   string `json:"artwork_url"`
	User         struct {
		Username string `json:"username"`
	} `json:"user"`
	Media struct {
		Transcodings []Transcoding `json:"transcodings"`
	} `json:"media"`
}

type collectionPage struct {
	Collection []struct {
		Track Track `json:"track"`
	} `json:"collection"`
	NextHref string `json:"next_href"`
}

func (c *Client) ResolveURL(pageURL string) (*Track, error) {
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		return nil, fmt.Errorf("soundcloud: empty url")
	}
	path := "/resolve?url=" + url.QueryEscape(pageURL)
	body, err := c.getJSONWithClientID(path)
	if err != nil {
		return nil, err
	}
	var track Track
	if err := decodeJSON(body, &track); err != nil {
		return nil, err
	}
	if track.ID == 0 {
		return nil, fmt.Errorf("soundcloud: resolve returned no track")
	}
	return &track, nil
}

func (c *Client) streamInfoURL(transcodingURL string) (string, error) {
	body, err := c.getJSONWithClientID(strings.TrimPrefix(transcodingURL, apiBase))
	if err != nil {
		if strings.HasPrefix(transcodingURL, "http") {
			if err2 := c.ensureClientID(); err2 != nil {
				return "", err
			}
			sep := "?"
			if strings.Contains(transcodingURL, "?") {
				sep = "&"
			}
			body, err = c.getJSON(transcodingURL + sep + "client_id=" + c.ClientID)
		}
		if err != nil {
			return "", err
		}
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(body, &out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("soundcloud: empty stream url")
	}
	return out.URL, nil
}

func pickTranscoding(track *Track) (Transcoding, error) {
	var progressive, hls Transcoding
	for _, t := range track.Media.Transcodings {
		switch t.Format.Protocol {
		case "progressive":
			if progressive.URL == "" || t.Quality == "hq" {
				progressive = t
			}
		case "hls":
			if hls.URL == "" {
				hls = t
			}
		}
	}
	if progressive.URL != "" {
		return progressive, nil
	}
	if hls.URL != "" {
		return hls, nil
	}
	return Transcoding{}, fmt.Errorf("soundcloud: no transcodings for track %d", track.ID)
}
