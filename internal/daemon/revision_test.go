package daemon

import (
	"testing"

	"github.com/sebday/evoplayer/internal/ipc"
)

func TestCheckQueueRevisionOptional(t *testing.T) {
	if err := checkQueueRevision(3, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckQueueRevisionMatch(t *testing.T) {
	rev := uint64(5)
	if err := checkQueueRevision(5, &rev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

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
