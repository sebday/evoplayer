package daemon

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/soundcloud"
)

func (d *Daemon) setPlaybackFollow(ids []int64) {
	paths := make([]string, 0, len(ids))
	cache := d.env().CacheDir
	for _, id := range ids {
		if id == 0 {
			continue
		}
		paths = append(paths, soundcloud.PreviewPath(cache, id))
	}
	d.followMu.Lock()
	d.followGen++
	if d.followCancel != nil {
		d.followCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.followCtx = ctx
	d.followCancel = cancel
	gen := d.followGen
	d.followMu.Unlock()
	d.Actor.SetFollow(paths)
	go d.prefetchFollow(ctx, gen, paths)
}

func (d *Daemon) prefetchFollow(ctx context.Context, gen int, paths []string) {
	for _, path := range paths {
		if ctx.Err() != nil {
			return
		}
		d.followMu.Lock()
		if gen != d.followGen {
			d.followMu.Unlock()
			return
		}
		d.followMu.Unlock()
		if d.followFileReady(path) {
			continue
		}
		d.ensureFollowFile(ctx, gen, path)
	}
}

func (d *Daemon) ensureFollowFile(ctx context.Context, gen int, path string) {
	d.followMu.Lock()
	if gen != d.followGen {
		d.followMu.Unlock()
		return
	}
	if d.followBusy == nil {
		d.followBusy = map[string]bool{}
	}
	if d.followBusy[path] {
		d.followMu.Unlock()
		return
	}
	d.followBusy[path] = true
	d.followMu.Unlock()

	defer func() {
		d.followMu.Lock()
		delete(d.followBusy, path)
		current := gen == d.followGen
		d.followMu.Unlock()
		if current && d.followFileReady(path) {
			_ = d.Actor.ContinueFollow(path)
		}
	}()

	id := followID(path)
	if id == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	_, _ = soundcloud.DownloadPreview(ctx, d.env(), id, jobs.NopReporter)
}

func (d *Daemon) followFileReady(path string) bool {
	d.followMu.Lock()
	busy := d.followBusy[path]
	d.followMu.Unlock()
	if busy || path == "" {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

func followID(path string) int64 {
	base := path
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	base = strings.TrimSuffix(base, ".mp3")
	id, err := strconv.ParseInt(base, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
