package warm_test

import (
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/warm"
)

func TestSchedulerDedupesPaths(t *testing.T) {
	env := paths.Env{}
	s := warm.NewScheduler(env, 1)
	defer s.Close()
	s.Enqueue("/a.mp3", warm.PriorityNormal, true)
	s.Enqueue("/a.mp3", warm.PriorityHigh, true)
	time.Sleep(100 * time.Millisecond)
}

func TestSchedulerWaitIdle(t *testing.T) {
	s := warm.NewScheduler(paths.Env{}, 2)
	defer s.Close()
	s.WaitIdle()
	s.EnqueueFull("/no-such-track.mp3", warm.PriorityLow)
	s.EnqueueFull("/also-missing.mp3", warm.PriorityNormal)
	done := make(chan struct{})
	go func() {
		s.WaitIdle()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WaitIdle hung")
	}
}
