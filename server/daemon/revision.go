package daemon

import (
	"github.com/sebday/evoplayer/server/ipc"
)

func checkQueueRevision(current uint64, ifRevision *uint64) error {
	if ifRevision == nil {
		return nil
	}
	if *ifRevision != current {
		return ipc.ErrConflict(current)
	}
	return nil
}
