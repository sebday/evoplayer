package warm_test

import (
	"testing"
	"time"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/warm"
)

func TestSchedulerDedupesPaths(t *testing.T) {
	env := paths.Env{}
	s := warm.NewScheduler(env, 1)
	defer s.Close()
	s.Enqueue("/a.mp3", warm.PriorityNormal, true)
	s.Enqueue("/a.mp3", warm.PriorityHigh, true)
	time.Sleep(100 * time.Millisecond)
}
