package jobs_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/jobs"
)

func waitOnChange(t *testing.T, n *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for n.Load() < want {
		if time.Now().After(deadline) {
			t.Fatalf("onChange count = %d, want %d", n.Load(), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStartNotifiesOnRunningAndDone(t *testing.T) {
	m := jobs.NewManager()
	var n atomic.Int32
	m.SetOnChange(func() { n.Add(1) })
	gate := make(chan struct{})
	st, err := m.Start("scan", func(context.Context) error {
		<-gate
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != "running" || st.Name != "scan" {
		t.Fatalf("start = %+v", st)
	}
	waitOnChange(t, &n, 1)
	close(gate)
	waitOnChange(t, &n, 2)
	got := m.Status()
	if got == nil || got.Status != "done" {
		t.Fatalf("status = %+v, want done", got)
	}
}

func TestCancelUnblocksJob(t *testing.T) {
	m := jobs.NewManager()
	started := make(chan struct{})
	st, err := m.Start("scan", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != "running" {
		t.Fatalf("start = %+v", st)
	}
	<-started
	if !m.Cancel() {
		t.Fatal("cancel should succeed while running")
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		got := m.Status()
		if got != nil && got.Status == "error" {
			if got.Error != context.Canceled.Error() {
				t.Fatalf("error = %q, want %q", got.Error, context.Canceled.Error())
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("status = %+v, want error", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if m.Cancel() {
		t.Fatal("cancel after finish should fail")
	}
}

func TestSetProgressThrottlesOnChange(t *testing.T) {
	m := jobs.NewManager()
	var n atomic.Int32
	m.SetOnChange(func() { n.Add(1) })
	gate := make(chan struct{})
	if _, err := m.Start("scan", func(context.Context) error {
		<-gate
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	waitOnChange(t, &n, 1)
	m.SetProgress(jobs.Progress{Phase: "index", Done: 1, Total: 10})
	afterPhase := n.Load()
	if afterPhase < 2 {
		t.Fatalf("phase progress should notify, onChange=%d", afterPhase)
	}
	m.SetProgress(jobs.Progress{Phase: "index", Done: 2, Total: 10})
	if n.Load() != afterPhase {
		t.Fatalf("same-phase progress should be throttled, onChange=%d", n.Load())
	}
	m.SetProgress(jobs.Progress{Phase: "warm", Done: 0, Total: 10})
	if n.Load() <= afterPhase {
		t.Fatal("phase change should notify")
	}
	close(gate)
}
