package soundcloud

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/tags"
)

// ArtRefreshStats is the outcome of a dump-folder artwork refresh.
type ArtRefreshStats struct {
	Queued  int
	Updated int
	Cleared int
	Failed  int
}

// RefreshSharedDumpArt re-fetches SoundCloud artwork for dump-folder tracks
// that currently share one cached cover. Each image is stored on that track
// only.
func RefreshSharedDumpArt(ctx context.Context, env paths.Env, limit int) (ArtRefreshStats, error) {
	lib := library.EnvFrom(env)
	dirs, tracks, err := library.SharedDumpCovers(lib)
	if err != nil {
		return ArtRefreshStats{}, err
	}
	for _, dir := range dirs {
		library.ClearDumpFolderCover(lib, dir)
	}
	found := len(tracks)
	if limit > 0 && len(tracks) > limit {
		tracks = tracks[:limit]
	}
	stats := ArtRefreshStats{Queued: len(tracks)}
	if len(tracks) == 0 {
		return stats, nil
	}
	opts, err := LoadOptions(env)
	if err != nil {
		return stats, err
	}
	fmt.Fprintf(os.Stderr, "soundcloud: refreshing artwork for %d of %d shared covers\n", len(tracks), found)

	jobs := make(chan string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	workers := 3
	if len(tracks) < workers {
		workers = len(tracks)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := NewClient(opts.ClientID, opts.OAuthToken)
			for path := range jobs {
				if ctx.Err() != nil {
					mu.Lock()
					stats.Failed++
					mu.Unlock()
					continue
				}
				kind, rerr := refreshTrackArt(client, lib, path)
				mu.Lock()
				switch {
				case rerr != nil:
					stats.Failed++
					fmt.Fprintf(os.Stderr, "soundcloud: art fail %s: %v\n", path, rerr)
				case kind == "cleared":
					stats.Cleared++
				default:
					stats.Updated++
				}
				done := stats.Updated + stats.Cleared + stats.Failed
				if done%25 == 0 || done == stats.Queued {
					fmt.Fprintf(os.Stderr, "soundcloud: art %d/%d\n", done, stats.Queued)
				}
				mu.Unlock()
			}
		}()
	}
	for _, path := range tracks {
		jobs <- path
	}
	close(jobs)
	wg.Wait()
	if ctx.Err() != nil {
		return stats, ctx.Err()
	}
	return stats, nil
}

func refreshTrackArt(client *Client, lib library.Env, path string) (string, error) {
	var last error
	for attempt := 0; attempt < 4; attempt++ {
		kind, err := refreshTrackArtOnce(client, lib, path)
		if err == nil || !retryableArtErr(err) {
			return kind, err
		}
		last = err
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	return "", last
}

func refreshTrackArtOnce(client *Client, lib library.Env, path string) (string, error) {
	probed, err := tags.Probe(path)
	if err != nil {
		return "", err
	}
	track, err := resolveSoundcloudTrack(client, probed.Tag, probed.Duration)
	if err != nil {
		return "", err
	}
	data, _, err := fetchArtwork(client, track)
	if err != nil {
		if errors.Is(err, errNoArtwork) {
			if serr := library.StripTrackCover(lib, path); serr != nil {
				return "", serr
			}
			return "cleared", nil
		}
		return "", err
	}
	tmp, err := os.CreateTemp("", "scart-*"+artworkExt(data))
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	defer os.Remove(tmpName)
	if err := library.ReplaceTrackCover(lib, path, tmpName); err != nil {
		return "", err
	}
	return "updated", nil
}

func resolveSoundcloudTrack(client *Client, info tags.TagInfo, duration float64) (*Track, error) {
	if id, err := strconv.ParseInt(strings.TrimSpace(info.SoundcloudID), 10, 64); err == nil && id != 0 {
		if track, ferr := client.Track(id); ferr == nil && track != nil && track.ID != 0 {
			return track, nil
		}
	}
	return client.MatchTrack(info.Artist, info.Title, duration)
}

var errNoArtwork = errors.New("soundcloud: track has no artwork")

func fetchArtwork(client *Client, track *Track) ([]byte, string, error) {
	if track == nil {
		return nil, "", errNoArtwork
	}
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 4)
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		large := tags.ArtworkURLLarge(raw)
		for _, u := range []string{large, raw} {
			if u == "" {
				continue
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			candidates = append(candidates, u)
		}
	}
	add(track.ArtworkURL)
	add(track.User.AvatarURL)
	if len(candidates) == 0 {
		return nil, "", errNoArtwork
	}
	var last error
	for _, u := range candidates {
		data, mime, err := fetchBytes(client.HTTP, u)
		if err == nil && len(data) > 0 {
			if mime == "" || strings.Contains(mime, "text/") || mime == "application/octet-stream" {
				mime = tags.PictureMIME(data)
			}
			return data, mime, nil
		}
		last = err
	}
	if last == nil {
		last = errNoArtwork
	}
	return nil, "", last
}

func artworkExt(data []byte) string {
	switch tags.PictureMIME(data) {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func retryableArtErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "temporarily")
}
