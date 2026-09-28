package daemon

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/soundcloud"
)

func (d *Daemon) handleDiscover(req ipc.Request) (interface{}, error) {
	switch req.Method {
	case "discover.similar":
		var p struct {
			Path string `json:"path"`
			ID   int64  `json:"id"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		p.Path = strings.TrimSpace(p.Path)
		if p.ID == 0 && p.Path == "" {
			p.Path = d.Actor.Snapshot().Path
		}
		if p.ID == 0 && p.Path == "" {
			return nil, ipc.ErrInvalidParams("path or id required")
		}
		if res, ok := d.discoverLookup(p.Path, p.ID); ok {
			return res, nil
		}
		res, err := soundcloud.Similar(d.Env, soundcloud.Seed{Path: p.Path, ID: p.ID})
		if err != nil {
			return nil, err
		}
		d.discoverStore(p.Path, res)
		return res, nil
	case "discover.preview":
		id, err := discoverID(req)
		if err != nil {
			return nil, err
		}
		path := soundcloud.PreviewPath(d.Env.CacheDir, id)
		if st, err := os.Stat(path); err == nil && !st.IsDir() && st.Size() > 0 {
			if err := d.playPreview(path); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "path": path}, nil
		}
		st, err := d.jobs.Start("discover-preview", func(ctx context.Context) error {
			if err := d.runDiscoverPreviewJob(ctx, id); err != nil {
				return err
			}
			path := soundcloud.PreviewPath(d.Env.CacheDir, id)
			d.jobs.SetResult(map[string]any{"id": id, "path": path})
			return d.playPreview(path)
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		d.broadcastJob()
		return st, nil
	case "discover.keep":
		id, err := discoverID(req)
		if err != nil {
			return nil, err
		}
		preview := soundcloud.PreviewPath(d.Env.CacheDir, id)
		if st, err := os.Stat(preview); err == nil && !st.IsDir() && st.Size() > 0 {
			wasPlaying := d.Actor.Snapshot().Path == preview
			dest, err := soundcloud.Keep(context.Background(), d.Env, id, jobs.NopReporter)
			if err != nil {
				return nil, err
			}
			d.discoverDrop(id)
			if wasPlaying {
				if err := d.playPreview(dest); err != nil {
					return nil, err
				}
			}
			return map[string]any{"id": id, "path": dest}, nil
		}
		st, err := d.jobs.Start("discover-keep", func(ctx context.Context) error {
			if err := d.runDiscoverKeepJob(ctx, id); err != nil {
				return err
			}
			d.discoverDrop(id)
			d.jobs.SetResult(map[string]any{"id": id})
			return nil
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		d.broadcastJob()
		return st, nil
	case "discover.dismiss":
		id, err := discoverID(req)
		if err != nil {
			return nil, err
		}
		if err := soundcloud.DismissID(d.Env.StateDir, id); err != nil {
			return nil, err
		}
		d.discoverDrop(id)
		return map[string]any{"id": id}, nil
	default:
		return nil, ipc.ErrUnknownMethod(req.Method)
	}
}

func discoverID(req ipc.Request) (int64, error) {
	var p struct {
		ID int64 `json:"id"`
	}
	if err := ipc.DecodeParams(req.Params, &p); err != nil {
		return 0, err
	}
	if p.ID == 0 {
		return 0, ipc.ErrInvalidParams("id required")
	}
	return p.ID, nil
}

func (d *Daemon) playPreview(path string) error {
	if err := d.Actor.PlayDetached(path); err != nil {
		return err
	}
	d.broadcastStateFull()
	return nil
}

func (d *Daemon) runDiscoverPreviewJob(ctx context.Context, id int64) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return d.pauseWarmAndSupervise(ctx, discoverPreviewWorkerCmd(exe, id))
}

func (d *Daemon) runDiscoverKeepJob(ctx context.Context, id int64) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return d.pauseWarmAndSupervise(ctx, discoverKeepWorkerCmd(exe, id))
}

func discoverPreviewWorkerCmd(exe string, id int64) *exec.Cmd {
	return workerCmd(exe, "_job", "discover-preview", strconv.FormatInt(id, 10))
}

func discoverKeepWorkerCmd(exe string, id int64) *exec.Cmd {
	return workerCmd(exe, "_job", "discover-keep", strconv.FormatInt(id, 10))
}

func (d *Daemon) discoverLookup(path string, id int64) (soundcloud.SimilarResult, bool) {
	d.discoverMu.Lock()
	defer d.discoverMu.Unlock()
	if id == 0 && path != "" {
		id = d.discoverByPath[path]
	}
	if id == 0 {
		return soundcloud.SimilarResult{}, false
	}
	res, ok := d.discoverByID[id]
	return res, ok
}

func (d *Daemon) discoverStore(path string, res soundcloud.SimilarResult) {
	if res.SeedID == 0 {
		return
	}
	d.discoverMu.Lock()
	defer d.discoverMu.Unlock()
	if d.discoverByID == nil {
		d.discoverByID = map[int64]soundcloud.SimilarResult{}
		d.discoverByPath = map[string]int64{}
	}
	d.discoverByID[res.SeedID] = res
	if path != "" {
		d.discoverByPath[path] = res.SeedID
	}
}

func withoutDiscoverPreviews(cacheDir string, paths []string) []string {
	if len(paths) == 0 {
		return paths
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if soundcloud.IsPreviewPath(cacheDir, p) {
			continue
		}
		out = append(out, p)
	}
	if len(out) == len(paths) {
		return paths
	}
	return out
}

func (d *Daemon) discoverDrop(id int64) {
	if id == 0 {
		return
	}
	d.discoverMu.Lock()
	defer d.discoverMu.Unlock()
	for seed, res := range d.discoverByID {
		next := make([]soundcloud.SimilarTrack, 0, len(res.Tracks))
		for _, track := range res.Tracks {
			if track.ID != id {
				next = append(next, track)
			}
		}
		res.Tracks = next
		d.discoverByID[seed] = res
	}
}
