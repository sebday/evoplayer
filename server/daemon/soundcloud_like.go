package daemon

import (
	"context"
	"fmt"
	"strings"

	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/playlist"
	"github.com/sebday/evoplayer/server/soundcloud"
	"github.com/sebday/evoplayer/server/status"
)

func (d *Daemon) likeSoundCloud(id int64, artist, title string, like bool) (any, error) {
	env := playlist.EnvFrom(d.env())
	path, err := importedSoundCloudPath(env, artist, title)
	if err != nil {
		return nil, err
	}
	if path != "" {
		return d.finishSoundCloudLike(env, id, path, like, false)
	}
	if !like {
		return map[string]any{"id": id, "liked": false, "imported": false}, nil
	}
	if id == 0 {
		return nil, ipc.ErrInvalidParams("id required")
	}
	opts, err := soundcloud.LoadOptions(d.env())
	if err != nil {
		return nil, err
	}
	track, err := soundcloud.NewClient(opts.ClientID, opts.OAuthToken).Track(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(artist) == "" {
		artist = track.User.Username
	}
	if strings.TrimSpace(title) == "" {
		title = track.Title
	}
	path, err = importedSoundCloudPath(env, artist, title)
	if err != nil {
		return nil, err
	}
	if path != "" {
		return d.finishSoundCloudLike(env, id, path, true, false)
	}
	pageURL := strings.TrimSpace(track.PermalinkURL)
	if pageURL == "" {
		return nil, fmt.Errorf("soundcloud: track %d has no permalink", id)
	}
	artist, title = strings.TrimSpace(artist), strings.TrimSpace(title)
	st, err := d.jobs.Start("download-url", func(ctx context.Context) error {
		if err := d.runDownloadURLJob(ctx, pageURL, true); err != nil {
			return err
		}
		got, err := importedSoundCloudPath(playlist.EnvFrom(d.env()), artist, title)
		if err != nil {
			return err
		}
		if got == "" {
			return fmt.Errorf("left in incoming")
		}
		row, err := d.finishSoundCloudLike(playlist.EnvFrom(d.env()), id, got, true, true)
		if err != nil {
			return err
		}
		d.jobs.SetResult(row)
		d.scheduleArtMaintain()
		return nil
	})
	if err := wrapJobErr(err); err != nil {
		return nil, err
	}
	return st, nil
}

func importedSoundCloudPath(env playlist.Env, artist, title string) (string, error) {
	if strings.TrimSpace(artist) == "" && strings.TrimSpace(title) == "" {
		return "", nil
	}
	db, err := library.EnsureDB(env.Env)
	if err != nil || db == nil {
		return "", err
	}
	return library.FindImported(db, artist, title)
}

func (d *Daemon) finishSoundCloudLike(env playlist.Env, id int64, path string, like, imported bool) (map[string]any, error) {
	row, err := playlist.SetFavorite(env, path, like)
	if err != nil {
		return nil, err
	}
	status.InvalidateMeta(path)
	return map[string]any{
		"id":       id,
		"path":     row.Path,
		"liked":    row.Liked,
		"imported": imported,
	}, nil
}
