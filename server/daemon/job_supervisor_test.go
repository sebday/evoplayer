package daemon

import (
	"context"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sebday/evoplayer/server/jobs"
)

func TestRelayWorkerStdout(t *testing.T) {
	in := strings.NewReader(`{"type":"log","line":"hello"}
{"type":"progress","done":1,"total":2,"phase":"fetching"}
`)
	relay := &testJobRelay{}
	if err := relayWorkerStdout(in, relay); err != nil {
		t.Fatal(err)
	}
	if len(relay.logs) != 1 || relay.logs[0] != "hello" {
		t.Fatalf("logs = %#v", relay.logs)
	}
	if relay.progress.Done != 1 || relay.progress.Total != 2 {
		t.Fatalf("progress = %+v", relay.progress)
	}
}

func TestRelayWorkerStdoutErrorEvent(t *testing.T) {
	in := strings.NewReader(`{"type":"error","message":"boom"}
`)
	relay := &testJobRelay{}
	err := relayWorkerStdout(in, relay)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v", err)
	}
}

func TestSuperviseWorkerCancelKillsChild(t *testing.T) {
	cmd := exec.Command("sh", "-c", "sleep 30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	jm := startTestJob(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- superviseWorker(ctx, jm, cmd)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancel error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("supervisor did not return after cancel")
	}
}

func startTestJob(t *testing.T) *jobs.Manager {
	t.Helper()
	jm := jobs.NewManager()
	block := make(chan struct{})
	_, err := jm.Start("download", func(ctx context.Context) error {
		<-block
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(block) })
	return jm
}

type testJobRelay struct {
	logs     []string
	progress jobs.Progress
}

func (r *testJobRelay) AppendLog(s string)          { r.logs = append(r.logs, s) }
func (r *testJobRelay) SetProgress(p jobs.Progress) { r.progress = p }
