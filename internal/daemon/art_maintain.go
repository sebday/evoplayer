package daemon

import (
	"fmt"
	"os"

	"github.com/sebday/evoplayer/internal/library"
)

func (d *Daemon) scheduleArtMaintain() {
	go func() {
		if err := d.runArtMaintain(); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: art maintain: %v\n", err)
		}
	}()
}

func (d *Daemon) runArtMaintain() error {
	if !d.artMaintainMu.TryLock() {
		return nil
	}
	defer d.artMaintainMu.Unlock()
	if err := library.Maintain(library.EnvFrom(d.Env)); err != nil {
		return err
	}
	d.broadcastState()
	return nil
}
