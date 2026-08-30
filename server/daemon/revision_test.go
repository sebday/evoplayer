package daemon

import (
	"testing"

	"github.com/sebday/evoplayer/server/ipc"
)

func TestCheckQueueRevisionConflict(t *testing.T) {
	rev := uint64(2)
	err := checkQueueRevision(5, &rev)
	if err == nil {
		t.Fatal("expected conflict")
	}
	ae, ok := ipc.AsError(err)
	if !ok || ae.Code != ipc.CodeConflict {
		t.Fatalf("got %#v", err)
	}
	if ae.Data["queue_revision"] != uint64(5) {
		t.Fatalf("queue_revision = %v", ae.Data["queue_revision"])
	}
}
