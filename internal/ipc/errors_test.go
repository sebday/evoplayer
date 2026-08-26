package ipc

import "testing"

func TestErrConflictData(t *testing.T) {
	err := ErrConflict(9)
	ae, ok := AsError(err)
	if !ok {
		t.Fatal("expected ipc error")
	}
	if ae.Code != CodeConflict {
		t.Fatalf("code = %q", ae.Code)
	}
	if ae.Data["queue_revision"] != uint64(9) {
		t.Fatalf("queue_revision = %v", ae.Data["queue_revision"])
	}
}
