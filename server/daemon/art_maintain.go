package daemon

import (
	"fmt"
	"os"

	"github.com/sebday/evoplayer/server/library"
)

func (d *Daemon) scheduleArtMaintain() {
	go func() {
		if err := d.runArtMaintain(false); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: art maintain: %v\n", err)
		}
	}()
}

// runArtMaintain skips when another pass is running unless wait is set.
func (d *Daemon) runArtMaintain(wait bool) error {
	if wait {
		d.artMaintainMu.Lock()
	} else if !d.artMaintainMu.TryLock() {
		return nil
	}
	defer d.artMaintainMu.Unlock()
	if err := library.Maintain(library.EnvFrom(d.env())); err != nil {
		return err
	}
	d.broadcastState()
	return nil
}
