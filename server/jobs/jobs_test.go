package jobs_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sebday/evoplayer/server/jobs"
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

func TestAppendLogCapsLines(t *testing.T) {
	m := jobs.NewManager()
	st, err := m.Start("download", func(context.Context) error {
		<-context.Background().Done()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		m.AppendLog(fmt.Sprintf("line %d", i))
	}
	got := m.Status()
	if got == nil {
		t.Fatal("missing status")
	}
	lines := strings.Split(strings.TrimSpace(got.Log), "\n")
	if len(lines) != 24 {
		t.Fatalf("log lines = %d, want 24: %q", len(lines), got.Log)
	}
	if !strings.HasPrefix(lines[0], "line 6") {
		t.Fatalf("log should keep the newest lines, got %q", got.Log)
	}
	if st.Status != "running" {
		t.Fatalf("start = %+v", st)
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
