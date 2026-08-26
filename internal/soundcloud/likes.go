package soundcloud

import (
	"fmt"
	"strings"
)

func (c *Client) LikesTracks() ([]Track, error) {
	if c.OAuthToken == "" {
		return nil, fmt.Errorf("soundcloud: oauth_token required for likes sync")
	}
	var tracks []Track
	next := "/me/likes?limit=50&linked_partitioning=1"
	for next != "" {
		path := next
		if strings.HasPrefix(path, apiBase) {
			path = strings.TrimPrefix(path, apiBase)
		}
		body, err := c.getJSONWithClientID(path)
		if err != nil {
			return nil, err
		}
		var page collectionPage
		if err := decodeJSON(body, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Collection {
			if item.Track.ID != 0 {
				tracks = append(tracks, item.Track)
			}
		}
		next = strings.TrimSpace(page.NextHref)
	}
	return tracks, nil
}
