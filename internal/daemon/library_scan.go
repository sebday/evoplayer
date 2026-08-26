package daemon

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/status"
	"github.com/sebday/evoplayer/internal/warm"
)

const (
	defaultLibrarySyncDelay    = 15 * time.Second
	defaultLibrarySyncInterval = 2 * time.Minute
)

func (d *Daemon) scheduleLibraryScan(ctx context.Context) {
	go func() {
		d.startBootstrapScan()
		d.loopLibrarySync(ctx)
	}()
}

func (d *Daemon) startBootstrapScan() bool {
	env := library.EnvFrom(d.Env)
	need, err := library.NeedsBootstrap(env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: library scan check: %v\n", err)
		return false
	}
	if !need {
		return false
	}
	_, err = d.jobs.Start("scan", func(ctx context.Context) error {
		return d.runLibraryScan(ctx, env)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: library scan: %v\n", err)
		return false
	}
	d.broadcastJob()
	return true
}

func (d *Daemon) loopLibrarySync(ctx context.Context) {
	timer := time.NewTimer(d.librarySyncDelay())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			d.syncLibraryOnce(ctx)
			timer.Reset(d.librarySyncInterval())
		}
	}
}

func (d *Daemon) librarySyncDelay() time.Duration {
	if d.syncDelay > 0 {
		return d.syncDelay
	}
	return defaultLibrarySyncDelay
}

func (d *Daemon) librarySyncInterval() time.Duration {
	if d.syncInterval > 0 {
		return d.syncInterval
	}
	return defaultLibrarySyncInterval
}

func (d *Daemon) jobRunning() bool {
	st := d.jobs.Status()
	return st != nil && st.Status == "running"
}

func (d *Daemon) tryBeginSync() bool {
	d.syncMu.Lock()
	defer d.syncMu.Unlock()
	if d.syncing {
		return false
	}
	d.syncing = true
	return true
}

func (d *Daemon) endSync() {
	d.syncMu.Lock()
	d.syncing = false
	d.syncMu.Unlock()
}

func (d *Daemon) syncLibraryOnce(ctx context.Context) {
	if d.jobRunning() {
		return
	}
	if !d.tryBeginSync() {
		return
	}
	defer d.endSync()
	if _, err := d.runLibrarySync(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: library sync: %v\n", err)
	}
}

func (d *Daemon) runLibrarySync(ctx context.Context) ([]string, error) {
	env := library.EnvFrom(d.Env)
	if err := d.runArtMaintain(); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: warn: art maintain: %v\n", err)
	}
	need, err := library.NeedsScan(env)
	if err != nil {
		return nil, err
	}
	if need {
		if _, err := library.CacheAllCtx(ctx, env, false, nil); err != nil {
			return nil, err
		}
		status.InvalidateAllMeta()
		d.broadcastState()
	}
	paths, err := library.MissingWaveforms(env)
	if err != nil {
		return nil, err
	}
	if len(paths) > 0 {
		d.warm.EnqueueManyFull(paths, warm.PriorityLow)
	}
	return paths, nil
}
