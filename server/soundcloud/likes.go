package soundcloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/sebday/evoplayer/server/secrets"
)

func (c *Client) LikesTracksProgressCtx(ctx context.Context, onPage func(n int)) ([]Track, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.OAuthToken == "" {
		return nil, fmt.Errorf("soundcloud: oauth_token required (brave cookie or pass show %s)", secrets.SoundcloudPassPath())
	}
	userID, err := c.meID()
	if err != nil {
		return nil, err
	}
	var tracks []Track
	next := fmt.Sprintf("/users/%d/track_likes?limit=50&linked_partitioning=1", userID)
	for next != "" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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
		if onPage != nil {
			onPage(len(tracks))
		}
		next = strings.TrimSpace(page.NextHref)
	}
	return tracks, nil
}

func (c *Client) meID() (int64, error) {
	body, err := c.getJSONWithClientID("/me")
	if err != nil {
		return 0, err
	}
	var me struct {
		ID int64 `json:"id"`
	}
	if err := decodeJSON(body, &me); err != nil {
		return 0, err
	}
	if me.ID == 0 {
		return 0, fmt.Errorf("soundcloud: /me returned no user id")
	}
	return me.ID, nil
}
