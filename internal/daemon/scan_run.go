package daemon

import (
	"context"
	"sync"

	"github.com/sebday/evoplayer/internal/jobs"
	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/status"
	"github.com/sebday/evoplayer/internal/warm"
)

func (d *Daemon) runLibraryScan(ctx context.Context, env library.Env) error {
	d.jobs.SetProgress(jobs.Progress{Phase: "index"})

	_, err := library.CacheAllCtx(ctx, env, false, func(p library.CacheProgress) {
		d.jobs.SetProgress(jobs.Progress{
			Phase:  "index",
			Folder: p.Folder,
			Done:   p.Done,
			Total:  p.Total,
		})
	})
	if err != nil {
		return err
	}
	status.InvalidateAllMeta()

	paths, err := library.ListTrackPaths(env)
	if err != nil {
		return err
	}
	total := len(paths)
	d.jobs.SetProgress(jobs.Progress{Phase: "warm", Total: total})

	var warmMu sync.Mutex
	warmDone := 0
	d.warm.SetOnProgress(func(path string) {
		warmMu.Lock()
		warmDone++
		done := warmDone
		folder := library.CacheFolderLabel(env, path)
		warmMu.Unlock()
		d.jobs.SetProgress(jobs.Progress{
			Phase:  "warm",
			Folder: folder,
			Done:   done,
			Total:  total,
		})
	})
	defer d.warm.SetOnProgress(nil)

	d.warm.EnqueueManyFull(paths, warm.PriorityLow)
	if err := d.warm.WaitIdleCtx(ctx); err != nil {
		return err
	}
	d.jobs.ClearProgress()
	d.broadcastState()
	return nil
}
