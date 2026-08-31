package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/playlist"
	"github.com/sebday/evoplayer/server/status"
	"github.com/sebday/evoplayer/server/viz"
	"github.com/sebday/evoplayer/server/warm"
)

type Daemon struct {
	Env               paths.Env
	Actor             *playback.Actor
	Server            *ipc.Server
	jobs              *jobs.Manager
	mpris             mprisCloser
	lock              *os.File
	vizMu             sync.Mutex
	vizSubs           int
	vizSeq            atomic.Uint64
	vizFrame          *viz.FrameWriter
	warm              *warm.Scheduler
	scrobbleMu        sync.Mutex
	scrobbleDedupeKey string
	scrobbleDedupeAt  time.Time
	scrobblePrev      playback.Status
	scrobblePath      string
	scrobbleStartPos  float64
	scrobbleStartedAt int64
	scrobbleSubmitted bool
	artMaintainMu     sync.Mutex
	syncMu            sync.Mutex
	syncing           bool
	syncDelay         time.Duration
	syncInterval      time.Duration
	persistMu         sync.Mutex
	persistTimer      *time.Timer
	persistPath       string
	persistState      string
	persistVolume     int
	lastWarmPath      string
}

type mprisCloser interface {
	Sync(st playback.Status)
	Close()
}

func New(env paths.Env) *Daemon {
	d := &Daemon{Env: env, jobs: jobs.NewManager()}
	d.warm = warm.NewScheduler(env, warm.DefaultWorkers)
	d.warm.SetOnComplete(func(path string, art bool) {
		if path != "" {
			status.InvalidateMeta(path)
		}
		d.Server.Broadcast(ipc.Event{Event: "warm", Data: map[string]any{
			"path": path,
			"art":  art,
		}})
		if d.Actor != nil && d.Actor.Snapshot().Path == path {
			d.broadcastStateFull()
		}
	})
	d.Server = ipc.NewServer(env.SocketPath, func(req ipc.Request) (interface{}, error) {
		before := d.Actor.Snapshot()
		traceIPCIn(req, before)
		start := time.Now()
		data, err := d.handle(req)
		traceIPCOut(req, before, d.Actor.Snapshot(), err, time.Since(start))
		return data, err
	})
	d.Server.OnDisconnect = func() {
		d.vizMu.Lock()
		if d.vizSubs > 0 {
			d.vizSubs--
		}
		want := d.vizSubs > 0
		d.vizMu.Unlock()
		d.Actor.SetVizWanted(want)
	}
	d.Actor = playback.NewActor(func(st playback.Status) {
		if st.Path != "" && st.Path != d.lastWarmPath {
			d.lastWarmPath = st.Path
			d.warm.Enqueue(st.Path, warm.PriorityHigh, true)
		}
		d.autoScrobble(st)
		d.broadcastState()
		d.persistPlayerState(st)
	})
	if vol, ok := status.SavedVolume(env); ok {
		d.Actor.SetVolume(vol)
		d.persistVolume = vol
	} else {
		d.persistVolume = -1
	}
	d.vizFrame = viz.NewFrameWriter(viz.FramePath(env.SocketPath))
	d.Actor.SetVizOnUpdate(func(levels []float32) {
		d.broadcastViz(levels)
	})
	_ = d.applyVizConfig()
	d.jobs.SetOnChange(func() {
		d.broadcastJob()
	})
	return d
}

func (d *Daemon) Run() error {
	if err := d.Env.EnsureDirs(); err != nil {
		return err
	}
	initScrobbleCredentials()
	if err := d.acquireLock(); err != nil {
		return err
	}
	defer d.releaseLock()
	if err := d.Server.Listen(); err != nil {
		return fmt.Errorf("ipc listen: %w", err)
	}
	defer d.Server.Close()
	defer d.flushPlayerState()
	defer d.Actor.CloseOutput()
	defer d.vizFrame.Close()
	if d.mpris != nil {
		defer d.mpris.Close()
	}
	if err := d.resumePlayback(false); err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: restore: %v\n", err)
	}
	go func() {
		if err := d.Actor.EnsureOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: audio init: %v\n", err)
		}
	}()
	d.initMPRIS()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.scheduleLibraryScan(ctx)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		d.flushPlayerState()
		_ = d.Server.Close()
	}()
	return d.Server.Serve()
}

func (d *Daemon) acquireLock() error {
	if err := os.MkdirAll(filepath.Dir(d.Env.DaemonLock), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(d.Env.DaemonLock, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	if err := lockFile(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("daemon already running: %w", err)
	}
	if err := writeLockPID(f); err != nil {
		_ = unlockFile(f)
		_ = f.Close()
		return err
	}
	d.lock = f
	return nil
}

func (d *Daemon) releaseLock() {
	if d.lock != nil {
		_ = unlockFile(d.lock)
		_ = d.lock.Close()
		d.lock = nil
	}
}

func (d *Daemon) handle(req ipc.Request) (interface{}, error) {
	switch req.Method {
	case "capabilities":
		return capabilities(), nil
	case "state.get":
		return status.EnrichLight(d.Env, d.Actor.Snapshot()), nil
	case "subscribe":
		return status.EnrichFull(d.Env, d.Actor.Snapshot()), nil
	case "playback.toggle":
		st := d.Actor.Snapshot()
		if st.Path == "" || st.State == "stopped" {
			if err := d.resumePlayback(true); err != nil {
				return nil, err
			}
			d.broadcastStateFull()
			return nil, nil
		}
		if err := d.Actor.Toggle(); err != nil {
			return nil, err
		}
		return nil, nil
	case "playback.next":
		go func() { _ = d.Actor.Next() }()
		return nil, nil
	case "playback.prev":
		go func() { _ = d.Actor.Prev() }()
		return nil, nil
	case "playback.stop":
		go func() { d.Actor.Stop() }()
		return nil, nil
	case "queue.replace", "queue.load":
		var p struct {
			Paths      []string `json:"paths"`
			StartPath  string   `json:"start_path"`
			IfRevision *uint64  `json:"if_revision"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, ipc.ErrInvalidParams("%v", err)
		}
		if err := checkQueueRevision(d.Actor.QueueRevision(), p.IfRevision); err != nil {
			return nil, err
		}
		if err := d.Actor.ReplaceQueue(p.Paths, p.StartPath); err != nil {
			return nil, err
		}
		if err := d.saveCurrentQueue(); err != nil {
			return nil, err
		}
		d.broadcastStateFull()
		return map[string]any{
			"paths":          len(p.Paths),
			"queue_revision": d.Actor.QueueRevision(),
		}, nil
	case "queue.play_path":
		var p struct {
			Path       string  `json:"path"`
			IfRevision *uint64 `json:"if_revision"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, ipc.ErrInvalidParams("%v", err)
		}
		if err := checkQueueRevision(d.Actor.QueueRevision(), p.IfRevision); err != nil {
			return nil, err
		}
		if err := d.Actor.PlayPathInQueue(p.Path); err != nil {
			return nil, err
		}
		d.broadcastStateFull()
		return map[string]any{"queue_revision": d.Actor.QueueRevision()}, nil
	case "queue.play_current":
		var p struct {
			Paths      []string `json:"paths"`
			StartPath  string   `json:"start_path"`
			IfRevision *uint64  `json:"if_revision"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, ipc.ErrInvalidParams("%v", err)
		}
		if err := checkQueueRevision(d.Actor.QueueRevision(), p.IfRevision); err != nil {
			return nil, err
		}
		env := playlist.EnvFrom(d.Env)
		if len(p.Paths) > 0 {
			paths := append([]string(nil), p.Paths...)
			if _, err := playlist.SaveCurrentFast(env, paths); err != nil {
				return nil, err
			}
			go func() {
				_ = playlist.EnrichCurrentTracksJSON(env, paths)
			}()
		}
		paths, err := playlist.ReadCurrentPaths(env)
		if err != nil {
			return nil, err
		}
		if len(paths) == 0 {
			return nil, fmt.Errorf("current queue empty")
		}
		start := p.StartPath
		st := d.Actor.Snapshot()
		if start == "" {
			if st.Path != "" && pathInList(paths, st.Path) {
				start = st.Path
			} else {
				start = paths[0]
			}
		} else if !pathInList(paths, start) {
			if st.Path != "" && pathInList(paths, st.Path) {
				start = st.Path
			} else {
				start = paths[0]
			}
		}
		if st.Path == start && pathInList(paths, start) {
			if err := d.Actor.SetQueue(paths, start); err != nil {
				return nil, err
			}
		} else if err := d.Actor.ReplaceQueue(paths, start); err != nil {
			return nil, err
		}
		return map[string]any{"queue_revision": d.Actor.QueueRevision()}, nil
	case "queue.append":
		var p struct {
			Paths      []string `json:"paths"`
			IfRevision *uint64  `json:"if_revision"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, ipc.ErrInvalidParams("%v", err)
		}
		if err := checkQueueRevision(d.Actor.QueueRevision(), p.IfRevision); err != nil {
			return nil, err
		}
		d.Actor.Append(p.Paths)
		if err := d.saveCurrentQueue(); err != nil {
			return nil, err
		}
		return map[string]any{"queue_revision": d.Actor.QueueRevision()}, nil
	case "queue.append_folder":
		var p struct {
			Path       string  `json:"path"`
			IfRevision *uint64 `json:"if_revision"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, ipc.ErrInvalidParams("%v", err)
		}
		if err := checkQueueRevision(d.Actor.QueueRevision(), p.IfRevision); err != nil {
			return nil, err
		}
		libEnv := library.EnvFrom(d.Env)
		rel := strings.Trim(strings.TrimPrefix(p.Path, "/"), "/")
		if strings.HasPrefix(p.Path, d.Env.MusicRoot) {
			rel = strings.TrimPrefix(p.Path, d.Env.MusicRoot)
			rel = strings.Trim(strings.TrimPrefix(rel, "/"), "/")
		}
		dir := filepath.Join(d.Env.MusicRoot, filepath.FromSlash(rel))
		paths, err := library.CollectQueuePaths(libEnv, rel, dir)
		if err != nil {
			return nil, err
		}
		if len(paths) == 0 {
			return map[string]any{"added": 0, "paths": []string{}}, nil
		}
		env := playlist.EnvFrom(d.Env)
		current, _ := playlist.ReadCurrentPaths(env)
		seen := map[string]struct{}{}
		merged := make([]string, 0, len(current)+len(paths))
		for _, p := range current {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			merged = append(merged, p)
		}
		added := 0
		for _, p := range paths {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			merged = append(merged, p)
			added++
		}
		st := d.Actor.Snapshot()
		start := st.Path
		if !pathInList(merged, start) {
			start = ""
		}
		if err := d.Actor.SetQueue(merged, start); err != nil {
			return nil, err
		}
		if _, err := playlist.SaveCurrentFast(env, merged); err != nil {
			return nil, err
		}
		go func(saved []string) {
			_ = playlist.EnrichCurrentTracksJSON(env, saved)
		}(append([]string(nil), merged...))
		return map[string]any{"added": added, "paths": paths, "queue_revision": d.Actor.QueueRevision()}, nil
	case "queue.up_next":
		var p struct {
			Limit int `json:"limit"`
		}
		_ = ipc.DecodeParams(req.Params, &p)
		if p.Limit <= 0 {
			p.Limit = 3
		}
		paths := d.Actor.UpNextPaths(p.Limit)
		if len(paths) == 0 {
			return []library.Track{}, nil
		}
		return library.TracksForPaths(library.EnvFrom(d.Env), paths, ""), nil
	case "playback.seek":
		var p struct {
			Seconds float64 `json:"seconds"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		go func() { _ = d.Actor.Seek(p.Seconds) }()
		return nil, nil
	case "playback.volume.set":
		var p struct {
			Volume int `json:"volume"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		go func() { d.Actor.SetVolume(p.Volume) }()
		return nil, nil
	case "playback.volume.delta":
		var p struct {
			Delta int `json:"delta"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		go func() { d.Actor.AdjustVolume(p.Delta) }()
		return nil, nil
	case "playback.shuffle":
		var p struct {
			On bool `json:"on"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		go func() { d.Actor.SetShuffle(p.On) }()
		return nil, nil
	case "viz.config":
		return d.vizConfigView(), nil
	case "viz.config.apply":
		return d.reloadVizFromFile()
	case "viz.config.set":
		current := d.Actor.VizAnalyzer().Config()
		patch := map[string]any{}
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &patch); err != nil {
				return nil, err
			}
		}
		merged := mergeVizPatch(current, patch)
		if err := d.persistVizConfig(merged); err != nil {
			return nil, err
		}
		d.Actor.VizAnalyzer().ApplyConfig(merged)
		return d.vizConfigView(), nil
	case "config.set":
		return d.handleConfigSet(req)
	case "viz.subscribe":
		d.vizMu.Lock()
		d.vizSubs++
		want := d.vizSubs > 0
		d.vizMu.Unlock()
		d.Actor.SetVizWanted(want)
		if want {
			d.broadcastViz(d.Actor.VizAnalyzer().Snapshot())
		}
		return map[string]any{"subscribed": true}, nil
	case "viz.unsubscribe":
		d.vizMu.Lock()
		if d.vizSubs > 0 {
			d.vizSubs--
		}
		want := d.vizSubs > 0
		d.vizMu.Unlock()
		d.Actor.SetVizWanted(want)
		return map[string]any{"subscribed": false}, nil
	case "spectrum.get":
		levels := d.Actor.VizAnalyzer().Snapshot()
		out := make([]float64, len(levels))
		for i, v := range levels {
			out[i] = float64(v)
		}
		return map[string]any{
			"ok":       true,
			"levels":   out,
			"sequence": d.vizSeq.Load(),
		}, nil
	default:
		domain := methodDomain(req.Method)
		switch domain {
		case "library":
			if req.Method == "library.import" || req.Method == "library.cache" || req.Method == "library.soundcloud.download" || req.Method == "library.download" || req.Method == "library.art.maintain" {
				return d.handleJob(req)
			}
			return d.handleLibrary(req)
		case "job":
			return d.handleJob(req)
		case "scrobble":
			return d.handleScrobble(req)
		default:
			return nil, ipc.ErrUnknownMethod(req.Method)
		}
	}
}

func (d *Daemon) saveCurrentQueue() error {
	env := playlist.EnvFrom(d.Env)
	paths := d.Actor.QueuePaths()
	if _, err := playlist.SaveCurrentFast(env, paths); err != nil {
		return err
	}
	go func(saved []string) {
		_ = playlist.EnrichCurrentTracksJSON(env, saved)
	}(append([]string(nil), paths...))
	return nil
}
