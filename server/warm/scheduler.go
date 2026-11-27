package warm

import (
	"context"
	"sync"

	"github.com/sebday/evoplayer/server/paths"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
)

type job struct {
	path     string
	priority Priority
}

// Scheduler deduplicates and prioritizes background asset warming.
type Scheduler struct {
	mu         sync.Mutex
	wake       *sync.Cond
	inflight   map[string]struct{}
	pending    []job
	env        paths.Env
	workers    int
	cancel     context.CancelFunc
	paused     bool
	onComplete func(path string, art bool)
	onProgress func(path string)
}

func NewScheduler(env paths.Env, workers int) *Scheduler {
	if workers <= 0 {
		workers = DefaultWorkers
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{
		inflight: map[string]struct{}{},
		env:      env,
		workers:  workers,
		cancel:   cancel,
	}
	s.wake = sync.NewCond(&s.mu)
	go s.loop(ctx)
	return s
}

func (s *Scheduler) SetEnv(env paths.Env) {
	s.mu.Lock()
	s.env = env
	s.mu.Unlock()
}

func (s *Scheduler) SetOnComplete(fn func(path string, art bool)) {
	s.mu.Lock()
	s.onComplete = fn
	s.mu.Unlock()
}

func (s *Scheduler) SetOnProgress(fn func(path string)) {
	s.mu.Lock()
	s.onProgress = fn
	s.mu.Unlock()
}

func (s *Scheduler) complete(path string, art bool) {
	s.mu.Lock()
	progress := s.onProgress
	fn := s.onComplete
	s.mu.Unlock()
	if progress != nil {
		progress(path)
	}
	if fn != nil {
		fn(path, art)
	}
}

func (s *Scheduler) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Lock()
	s.pending = nil
	s.wake.Broadcast()
	s.mu.Unlock()
}

func (s *Scheduler) Enqueue(path string, priority Priority) {
	if path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.inflight[path]; ok {
		return
	}
	for i, j := range s.pending {
		if j.path == path {
			if priority > j.priority {
				s.pending[i].priority = priority
			}
			return
		}
	}
	s.pending = append(s.pending, job{path: path, priority: priority})
	s.wake.Broadcast()
}

func (s *Scheduler) EnqueueMany(paths []string, priority Priority) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := make(map[string]int, len(s.pending))
	for i, j := range s.pending {
		index[j.path] = i
	}
	added := false
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := s.inflight[p]; ok {
			continue
		}
		if i, ok := index[p]; ok {
			if priority > s.pending[i].priority {
				s.pending[i].priority = priority
			}
			continue
		}
		s.pending = append(s.pending, job{path: p, priority: priority})
		index[p] = len(s.pending) - 1
		added = true
	}
	if added {
		s.wake.Broadcast()
	}
}

func (s *Scheduler) ClearPending() {
	s.mu.Lock()
	s.pending = nil
	s.wake.Broadcast()
	s.mu.Unlock()
}

func (s *Scheduler) Pause() {
	s.mu.Lock()
	s.paused = true
	s.wake.Broadcast()
	s.mu.Unlock()
}

func (s *Scheduler) Resume() {
	s.mu.Lock()
	s.paused = false
	s.wake.Broadcast()
	s.mu.Unlock()
}

func (s *Scheduler) WaitIdleCtx(ctx context.Context) error {
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			s.wake.Broadcast()
			s.mu.Unlock()
		case <-stop:
		}
	}()
	defer close(stop)
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.pending) > 0 || len(s.inflight) > 0 {
		if err := ctx.Err(); err != nil {
			s.pending = nil
			return err
		}
		s.wake.Wait()
	}
	return nil
}

func (s *Scheduler) loop(ctx context.Context) {
	sem := make(chan struct{}, s.workers)
	s.mu.Lock()
	for {
		for ctx.Err() == nil && (s.paused || len(s.pending) == 0) {
			s.wake.Wait()
		}
		if ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		j, env, ok := s.popLocked()
		if !ok {
			continue
		}
		s.mu.Unlock()
		sem <- struct{}{}
		go func(j job) {
			defer func() { <-sem }()
			var res Result
			var err error
			func() {
				defer func() { _ = recover() }()
				res, err = TrackAssets(env, j.path)
			}()
			s.mu.Lock()
			delete(s.inflight, j.path)
			s.wake.Broadcast()
			s.mu.Unlock()
			s.complete(j.path, err == nil && res.ArtBuilt)
		}(j)
		s.mu.Lock()
	}
}

func (s *Scheduler) pop() (job, paths.Env, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.popLocked()
}

func (s *Scheduler) popLocked() (job, paths.Env, bool) {
	if len(s.pending) == 0 {
		return job{}, paths.Env{}, false
	}
	best := 0
	for i := 1; i < len(s.pending); i++ {
		if s.pending[i].priority > s.pending[best].priority {
			best = i
		}
	}
	j := s.pending[best]
	s.pending = append(s.pending[:best], s.pending[best+1:]...)
	if _, ok := s.inflight[j.path]; ok {
		return job{}, paths.Env{}, false
	}
	s.inflight[j.path] = struct{}{}
	return j, s.env, true
}
