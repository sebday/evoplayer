package daemon

import (
	"github.com/sebday/evoplayer/server/notify"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/status"
)

func (d *Daemon) onPlaybackNotify(st playback.Status) {
	if !notify.Enabled() {
		return
	}
	d.notifyMu.Lock()
	defer d.notifyMu.Unlock()
	prev := d.notifyPrev
	if !d.notifyReady {
		d.notifyPrev = st
		d.notifyReady = true
		return
	}
	d.notifyPrev = st
	enriched := status.EnrichLight(d.Env, st)
	if enriched.Path != prev.Path && enriched.Path != "" {
		notify.NowPlaying(enriched)
		return
	}
	if enriched.Path != "" && enriched.State != prev.State &&
		(enriched.State == "playing" || enriched.State == "paused") {
		notify.Transport(enriched.State, enriched)
	}
}
