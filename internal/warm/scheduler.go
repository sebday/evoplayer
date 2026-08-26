package warm

import (
	"context"
	"sync"
	"time"

	"github.com/sebday/evoplayer/internal/paths"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
)

type jobKind int

const (
	kindWaveform jobKind = iota
	kindAssets
	kindFull
)

type job struct {
	path     string
	priority Priority
	kind     jobKind
}

// Scheduler deduplicates and prioritizes background asset warming.
type Scheduler struct {
	mu         sync.Mutex
	idle       *sync.Cond
	inflight   map[string]struct{}
	pending    []job
	env        paths.Env
	workers    int
	cancel     context.CancelFunc
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
	s.idle = sync.NewCond(&s.mu)
	go s.loop(ctx)
	return s
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
	s.idle.Broadcast()
	s.mu.Unlock()
}

func (s *Scheduler) Enqueue(path string, priority Priority, art bool) {
	kind := kindWaveform
	if art {
		kind = kindAssets
	}
	s.enqueue(path, priority, kind)
}

func (s *Scheduler) EnqueueMany(paths []string, priority Priority, art bool) {
	for _, p := range paths {
		s.Enqueue(p, priority, art)
	}
}

func (s *Scheduler) EnqueueFull(path string, priority Priority) {
	s.enqueue(path, priority, kindFull)
}

func (s *Scheduler) EnqueueManyFull(paths []string, priority Priority) {
	for _, p := range paths {
		s.EnqueueFull(p, priority)
	}
}

func (s *Scheduler) enqueue(path string, priority Priority, kind jobKind) {
	path = stringPath(path)
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
			s.pending[i].kind = mergeKind(j.kind, kind)
			return
		}
	}
	s.pending = append(s.pending, job{path: path, priority: priority, kind: kind})
}

func mergeKind(a, b jobKind) jobKind {
	if a == kindFull || b == kindFull {
		return kindFull
	}
	if a != b {
		return kindFull
	}
	return a
}

func (s *Scheduler) ClearPending() {
	s.mu.Lock()
	s.pending = nil
	s.idle.Broadcast()
	s.mu.Unlock()
}

func (s *Scheduler) WaitIdleCtx(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			s.ClearPending()
			return ctx.Err()
		default:
		}
		s.mu.Lock()
		idle := len(s.pending) == 0 && len(s.inflight) == 0
		if idle {
			s.mu.Unlock()
			return nil
		}
		s.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *Scheduler) WaitIdle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.pending) > 0 || len(s.inflight) > 0 {
		s.idle.Wait()
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	sem := make(chan struct{}, s.workers)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		j, ok := s.pop()
		if !ok {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		sem <- struct{}{}
		go func(j job) {
			defer func() { <-sem }()
			defer func() {
				_ = recover()
				s.mu.Lock()
				delete(s.inflight, j.path)
				if len(s.pending) == 0 && len(s.inflight) == 0 {
					s.idle.Broadcast()
				}
				s.mu.Unlock()
				s.complete(j.path, j.kind != kindWaveform)
			}()
			switch j.kind {
			case kindFull:
				_, _ = Track(s.env, j.path)
			case kindAssets:
				_, _ = TrackAssets(s.env, j.path)
			default:
				_, _ = WaveformForTrack(s.env, j.path)
			}
		}(j)
	}
}

func (s *Scheduler) pop() (job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending) == 0 {
		if len(s.inflight) == 0 {
			s.idle.Broadcast()
		}
		return job{}, false
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
		return job{}, false
	}
	s.inflight[j.path] = struct{}{}
	return j, true
}

func stringPath(p string) string {
	return p
}
