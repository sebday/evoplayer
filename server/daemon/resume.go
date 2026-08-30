package daemon

import (
	"os"
	"time"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/playlist"
	"github.com/sebday/evoplayer/server/status"
)

const persistDebounce = 2 * time.Second

func readSavedPlayback(env paths.Env) (path string, position float64, ok bool) {
	st := status.Saved(env)
	if st.Path == "" {
		return "", 0, false
	}
	if _, err := os.Stat(st.Path); err != nil {
		return "", 0, false
	}
	return st.Path, st.Position, true
}

func (d *Daemon) persistPlayerState(st playback.Status) {
	if st.Path == "" {
		return
	}
	st = status.EnrichLight(d.Env, st)
	d.persistMu.Lock()
	pathChanged := st.Path != d.persistPath
	stateChanged := st.State != d.persistState
	d.persistPath = st.Path
	d.persistState = st.State
	if pathChanged || stateChanged {
		if d.persistTimer != nil {
			d.persistTimer.Stop()
			d.persistTimer = nil
		}
		d.persistMu.Unlock()
		_ = status.Write(d.Env, st)
		return
	}
	if d.persistTimer == nil {
		d.persistTimer = time.AfterFunc(persistDebounce, func() {
			d.persistMu.Lock()
			d.persistTimer = nil
			d.persistMu.Unlock()
			latest := status.EnrichLight(d.Env, d.Actor.Snapshot())
			if latest.Path != "" {
				_ = status.Write(d.Env, latest)
			}
		})
	}
	d.persistMu.Unlock()
}

func (d *Daemon) flushPlayerState() {
	d.persistMu.Lock()
	if d.persistTimer != nil {
		d.persistTimer.Stop()
		d.persistTimer = nil
	}
	d.persistMu.Unlock()
	st := status.EnrichLight(d.Env, d.Actor.Snapshot())
	_ = status.Write(d.Env, st)
}

func (d *Daemon) resumePlayback(play bool) error {
	st := d.Actor.Snapshot()
	if st.Path != "" {
		if play && st.State == "paused" {
			return d.Actor.Toggle()
		}
		if !play && st.State == "playing" {
			return d.Actor.Toggle()
		}
		return nil
	}

	savedPath, savedPos, ok := readSavedPlayback(d.Env)
	if !ok {
		return nil
	}

	paths, err := playlist.ReadCurrentPaths(playlist.EnvFrom(d.Env))
	if err != nil {
		paths = nil
	}
	if len(paths) == 0 {
		paths = []string{savedPath}
	} else if !pathInList(paths, savedPath) {
		paths = append([]string{savedPath}, paths...)
	}

	if err := d.Actor.Restore(paths, savedPath, savedPos); err != nil {
		return err
	}
	if play {
		return d.Actor.Toggle()
	}
	return nil
}

func pathInList(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
