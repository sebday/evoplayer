package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/library/find"
	"github.com/sebday/evoplayer/server/mpris"
	"github.com/sebday/evoplayer/server/notify"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/playlist"
	"github.com/sebday/evoplayer/server/soundcloud"
	"github.com/sebday/evoplayer/server/status"
	"github.com/sebday/evoplayer/server/viz"
	"github.com/sebday/evoplayer/server/warm"
)

func (d *Daemon) mprisToggle() error {
	before := d.Actor.Snapshot()
	traceMPRIS("toggle", before)
	var err error
	if before.Path == "" || before.State == "stopped" {
		err = d.resumePlayback(true)
		if err == nil {
			d.broadcastStateFull()
		}
	} else {
		err = d.Actor.Toggle()
	}
	after := d.Actor.Snapshot()
	fmt.Fprintf(os.Stderr, "evoplayer: mpris toggle done %s -> %s\n",
		tracePlaybackSnap(before), tracePlaybackSnap(after))
	return err
}

func (d *Daemon) initMPRIS() {
	m, err := mpris.Start(
		func() playback.Status { return d.Actor.Snapshot() },
		func() error {
			return d.mprisToggle()
		},
		func() {
			traceMPRIS("stop", d.Actor.Snapshot())
			d.Actor.Stop()
		},
		func() error {
			traceMPRIS("next", d.Actor.Snapshot())
			return d.Actor.Next()
		},
		func() error {
			traceMPRIS("prev", d.Actor.Snapshot())
			return d.Actor.Prev()
		},
		func(seconds float64) error {
			traceMPRIS(fmt.Sprintf("seek %.1f", seconds), d.Actor.Snapshot())
			return d.Actor.Seek(seconds)
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evoplayer: mpris: %v\n", err)
		return
	}
	d.mpris = m
}

func (d *Daemon) broadcastState() {
	st := status.EnrichLight(d.env(), d.Actor.Snapshot())
	d.Server.Broadcast(ipc.Event{Event: "state", Data: st})
	if d.mpris != nil {
		d.mpris.Sync(st)
	}
}

func (d *Daemon) broadcastStateFull() {
	st := status.EnrichFull(d.env(), d.Actor.Snapshot())
	d.Server.Broadcast(ipc.Event{Event: "state", Data: st})
	if d.mpris != nil {
		d.mpris.Sync(st)
	}
}

func (d *Daemon) broadcastViz(levels []float32) {
	if d.vizFrame != nil {
		_ = d.vizFrame.Write(levels)
	} else {
		_ = viz.WriteFrame(viz.FramePath(d.env().SocketPath), levels)
	}
	if !d.Server.HasVizClients() || !d.Server.HasEventClients() {
		return
	}
	out := make([]float64, len(levels))
	for i, v := range levels {
		out[i] = float64(v)
	}
	seq := d.vizSeq.Add(1)
	d.Server.Broadcast(ipc.Event{Event: "viz", Data: map[string]any{
		"levels":     out,
		"sequence":   seq,
		"generation": d.Actor.TrackGeneration(),
	}})
}

func (d *Daemon) handleLibrary(req ipc.Request) (interface{}, error) {
	libEnv := library.EnvFrom(d.env())
	switch req.Method {
	case "library.meta":
		var p struct {
			Path string `json:"path"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		playlist := readPlaylist(d.env().PlayerState)
		row, err := library.Meta(libEnv, p.Path, playlist)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, ipc.ErrNotFound("track not found: %s", p.Path)
			}
			return nil, err
		}
		return row, nil
	case "library.browse":
		var p struct {
			Path           string `json:"path"`
			Queue          bool   `json:"queue"`
			QueuePathsOnly bool   `json:"queue_paths_only"`
			Offset         int    `json:"offset"`
			Limit          int    `json:"limit"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		return library.Browse(libEnv, library.BrowseOptions{
			Rel: p.Path, Queue: p.Queue, QueuePathsOnly: p.QueuePathsOnly,
			Offset: p.Offset, Limit: p.Limit,
		})
	case "library.genres":
		return library.Genres(libEnv)
	case "library.incoming.list":
		folders := library.GenreChoices(libEnv)
		return map[string]any{
			"files":   library.ListIncoming(libEnv),
			"folders": folders,
			"genres":  folders,
		}, nil
	case "library.incoming.set_genre":
		var p struct {
			Path   string `json:"path"`
			Genre  string `json:"genre"`
			Folder string `json:"folder"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		folder := strings.TrimSpace(p.Folder)
		if folder == "" {
			folder = strings.TrimSpace(p.Genre)
		}
		return library.SetIncomingGenre(libEnv, p.Path, folder)
	case "library.tracks":
		var p struct {
			Genre string `json:"genre"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		return library.TracksForGenre(libEnv, p.Genre)
	case "library.search":
		var p struct {
			Mode  string `json:"mode"`
			Query string `json:"query"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		return find.Tracks(libEnv, p.Mode, p.Query)
	case "library.soundcloud.search":
		var search struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := ipc.DecodeParams(req.Params, &search); err != nil {
			return nil, err
		}
		if strings.TrimSpace(search.Query) == "" {
			return nil, ipc.ErrInvalidParams("query required")
		}
		tracks, err := soundcloud.Search(d.env(), search.Query, search.Limit)
		if err != nil {
			return nil, err
		}
		return map[string]any{"tracks": tracks}, nil
	case "library.playlist.list":
		return playlist.ListIndex(playlist.EnvFrom(d.env()))
	case "library.playlist.tracks":
		var p struct {
			Name   string `json:"name"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		return playlist.TracksPageFor(playlist.EnvFrom(d.env()), p.Name, p.Offset, p.Limit)
	case "library.playlist.star":
		var p struct {
			Name string `json:"name"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		return playlist.StarToggle(playlist.EnvFrom(d.env()), p.Name)
	case "library.favorite.toggle":
		var p struct {
			Path string `json:"path"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		row, err := playlist.FavoriteToggle(playlist.EnvFrom(d.env()), p.Path)
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		status.InvalidateMeta(p.Path)
		notify.Favorite(d.env(), row.Liked, p.Path)
		return row, nil
	case "library.track.move":
		var p struct {
			Path   string `json:"path"`
			Folder string `json:"folder"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		libEnv := library.EnvFrom(d.env())
		res, err := library.MoveTrackToFolder(libEnv, p.Path, p.Folder)
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		_ = playlist.OnTrackMoved(playlist.EnvFrom(d.env()), res.From, res.To)
		_ = d.Actor.RelocatePath(res.From, res.To)
		status.InvalidateMeta(res.From)
		status.InvalidateMeta(res.To)
		return res, nil
	case "library.track.tags.get":
		var p struct {
			Path string `json:"path"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		tags, err := library.ReadTrackTags(p.Path)
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		return tags, nil
	case "library.track.tags.set":
		var p struct {
			Path   string `json:"path"`
			Title  string `json:"title"`
			Artist string `json:"artist"`
			Album  string `json:"album"`
			Year   string `json:"year"`
			Genre  string `json:"genre"`
			Label  string `json:"label"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		libEnv := library.EnvFrom(d.env())
		row, err := library.UpdateTrackTags(libEnv, p.Path, library.TrackTagsPatch{
			Title:  p.Title,
			Artist: p.Artist,
			Album:  p.Album,
			Year:   p.Year,
			Genre:  p.Genre,
			Label:  p.Label,
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		status.InvalidateMeta(p.Path)
		if snap := d.Actor.Snapshot(); snap.Path == p.Path {
			d.broadcastStateFull()
		}
		return row, nil
	case "library.current.load":
		return playlist.LoadCurrent(playlist.EnvFrom(d.env()))
	case "library.current.save":
		var p struct {
			Paths []string `json:"paths"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		if len(p.Paths) == 0 {
			return nil, fmt.Errorf("current save: no paths")
		}
		env := playlist.EnvFrom(d.env())
		paths := append([]string(nil), p.Paths...)
		changed, err := playlist.SaveCurrentFast(env, paths)
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		_ = playlist.EnrichCurrentTracksJSON(env, paths)
		return map[string]any{"saved": len(paths), "changed": changed}, nil
	case "library.warm":
		var p struct {
			Path string `json:"path"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		libEnv := library.EnvFrom(d.env())
		row, err := library.Meta(libEnv, p.Path, "")
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		path := p.Path
		d.warm.Enqueue(path, warm.PriorityHigh)
		return warm.Result{
			Path:  path,
			Art:   row.Art,
			Thumb: row.Thumb,
		}, nil
	case "library.warm.batch":
		var p struct {
			Paths   []string `json:"paths"`
			Workers int      `json:"workers"`
			Art     bool     `json:"art"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		paths := append([]string(nil), p.Paths...)
		d.warm.EnqueueMany(paths, warm.PriorityNormal)
		return []warm.Result{}, nil
	case "library.warm.waveform":
		var p struct {
			Path string `json:"path"`
		}
		if err := ipc.DecodeParams(req.Params, &p); err != nil {
			return nil, err
		}
		file, err := warm.WaveformForTrack(d.env(), p.Path)
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		status.InvalidateMeta(p.Path)
		return warm.Result{Path: p.Path, Waveform: file}, nil
	default:
		return nil, ipc.ErrUnknownMethod(req.Method)
	}
}

func wrapJobErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "job already running") {
		return ipc.ErrUnavailable("%s", err.Error())
	}
	return err
}

func (d *Daemon) handleJob(req ipc.Request) (interface{}, error) {
	switch req.Method {
	case "job.status":
		st := d.jobs.Status()
		if st == nil {
			return map[string]any{"status": "idle"}, nil
		}
		return st, nil
	case "job.cancel":
		cancelled := d.jobs.Cancel()
		d.warm.ClearPending()
		name := ""
		if st := d.jobs.Status(); st != nil {
			name = st.Name
		}
		return map[string]any{"cancelled": cancelled, "name": name}, nil
	case "library.import":
		st, err := d.jobs.Start("import", func(ctx context.Context) error {
			if err := d.runImportJob(ctx); err != nil {
				return err
			}
			d.jobs.SetResult(map[string]any{
				"files":   library.ListIncoming(library.EnvFrom(d.env())),
				"folders": library.GenreChoices(library.EnvFrom(d.env())),
				"genres":  library.GenreChoices(library.EnvFrom(d.env())),
			})
			status.InvalidateAllMeta()
			d.broadcastState()
			d.scheduleArtMaintain()
			return nil
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		return st, nil
	case "library.cache":
		var p struct {
			Genre string `json:"genre"`
			Force bool   `json:"force"`
		}
		_ = ipc.DecodeParams(req.Params, &p)
		st, err := d.jobs.Start("cache", func(ctx context.Context) error {
			if err := d.runCacheJob(ctx, p.Genre, p.Force); err != nil {
				return err
			}
			status.InvalidateAllMeta()
			d.broadcastState()
			return nil
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		return st, nil
	case "library.soundcloud.download":
		var p struct {
			Import bool `json:"import"`
		}
		_ = ipc.DecodeParams(req.Params, &p)
		st, err := d.jobs.Start("download", func(ctx context.Context) error {
			if err := d.runSoundCloudDownloadJob(ctx, p.Import); err != nil {
				return err
			}
			d.scheduleArtMaintain()
			return nil
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		return st, nil
	case "library.download":
		var p struct {
			URL    string `json:"url"`
			Import *bool  `json:"import"`
		}
		_ = ipc.DecodeParams(req.Params, &p)
		if strings.TrimSpace(p.URL) == "" {
			return nil, ipc.ErrInvalidParams("url required")
		}
		doImport := p.Import == nil || *p.Import
		st, err := d.jobs.Start("download-url", func(ctx context.Context) error {
			var err error
			if isSoundCloudLikesURL(p.URL) {
				err = d.runSoundCloudDownloadJob(ctx, doImport)
			} else {
				err = d.runDownloadURLJob(ctx, p.URL, doImport)
			}
			if err != nil {
				return err
			}
			env := library.EnvFrom(d.env())
			folders := library.GenreChoices(env)
			d.jobs.SetResult(map[string]any{
				"files":   library.ListIncoming(env),
				"folders": folders,
				"genres":  folders,
			})
			if doImport {
				status.InvalidateAllMeta()
				d.broadcastState()
			}
			d.scheduleArtMaintain()
			return nil
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		return st, nil
	case "library.art.maintain":
		st, err := d.jobs.Start("art-maintain", func(context.Context) error {
			return d.runArtMaintain(true)
		})
		if err := wrapJobErr(err); err != nil {
			return nil, err
		}
		return st, nil
	default:
		return nil, ipc.ErrUnknownMethod(req.Method)
	}
}

func readPlaylist(playerState string) string {
	b, err := os.ReadFile(playerState)
	if err != nil {
		return ""
	}
	var saved struct {
		Playlist string `json:"playlist"`
	}
	if json.Unmarshal(b, &saved) != nil {
		return ""
	}
	return saved.Playlist
}

func (d *Daemon) broadcastJob() {
	if ev := d.jobs.BroadcastEvent(); ev != nil {
		d.Server.Broadcast(ipc.Event{Event: "job", Data: ev})
	}
}

func methodDomain(method string) string {
	if i := strings.Index(method, "."); i > 0 {
		return method[:i]
	}
	return method
}
