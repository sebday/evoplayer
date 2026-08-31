package warm

import (
	"context"
	"testing"
	"time"

	"github.com/sebday/evoplayer/server/paths"
)

func TestSchedulerPauseBlocksDequeuing(t *testing.T) {
	dir := t.TempDir()
	s := NewScheduler(paths.Env{StateDir: dir, CacheDir: dir}, 1)
	s.Pause()
	s.Enqueue("/tmp/track.mp3", PriorityNormal, true)
	time.Sleep(150 * time.Millisecond)
	s.mu.Lock()
	inflight := len(s.inflight)
	s.mu.Unlock()
	if inflight != 0 {
		t.Fatalf("paused scheduler should not dequeue, inflight=%d", inflight)
	}
	s.Resume()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		s.mu.Lock()
		idle := len(s.pending) == 0 && len(s.inflight) == 0
		s.mu.Unlock()
		if idle {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("scheduler did not process after resume")
		case <-time.After(20 * time.Millisecond):
		}
	}
	s.Close()
}
