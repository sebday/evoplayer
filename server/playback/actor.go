package playback

import (
	"fmt"
	"math"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sebday/evoplayer/server/viz"
)

const vizPaintLeadSamples = 960 // ~20ms; one overlay frame plus paint

type Actor struct {
	mu                 sync.RWMutex
	playMu             sync.Mutex
	queue              []string
	index              int
	queueRev           uint64
	shuffle            bool
	shuffleOrd         []int
	repeat             bool
	volumePct          int
	paused             bool
	path               string
	durationSec        float64
	positionSec        float64
	playbackAnchorSec  float64
	playbackAnchorTime time.Time
	stream             StreamSeeker
	sourceSampleRate   SampleRate
	notify             func(Status)
	notifyCh           chan struct{}
	stopPos            chan struct{}
	output             PlayerOutput
	outputOnce         sync.Once
	outputInitErr      error
	loadGen            uint64
	detached           bool
	resumePos          float64
	cmdCh              chan func()
	viz                *viz.Analyzer
	eq                 *EQ
}

func NewActor(notify func(Status)) *Actor {
	a := &Actor{
		volumePct: 100,
		notify:    notify,
		notifyCh:  make(chan struct{}, 1),
		cmdCh:     make(chan func(), 64),
		viz:       viz.NewAnalyzer(float64(outputSampleRate)),
		eq:        NewEQ(),
	}
	a.output.SetVolume(volumeGain(a.volumePct))
	a.viz.SetDelayFunc(func() int {
		delay := a.output.PresentationDelaySamples() - vizPaintLeadSamples
		if delay < 0 {
			return 0
		}
		return delay
	})
	go a.workerLoop()
	if notify != nil {
		go a.notifyLoop()
	}
	return a
}

func (a *Actor) VizAnalyzer() *viz.Analyzer {
	return a.viz
}

func (a *Actor) SetVizWanted(on bool) {
	if a.viz != nil {
		a.viz.SetWanted(on)
	}
}

func (a *Actor) SetVizOnUpdate(fn func([]float32)) {
	if a.viz != nil {
		a.viz.SetOnUpdate(fn)
	}
}

func (a *Actor) EnsureOutput() error {
	return a.ensureOutput()
}

func (a *Actor) ensureOutput() error {
	a.outputOnce.Do(func() {
		a.outputInitErr = a.output.Init()
	})
	return a.outputInitErr
}

func (a *Actor) CloseOutput() {
	a.output.Close()
}

func (a *Actor) TrackGeneration() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.loadGen
}

func (a *Actor) Snapshot() Status {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.buildStatusLocked()
}

// emit schedules a notify with the latest status; bursts coalesce into one call.
func (a *Actor) emit() {
	select {
	case a.notifyCh <- struct{}{}:
	default:
	}
}

func (a *Actor) notifyLoop() {
	for range a.notifyCh {
		a.notify(a.Snapshot())
	}
}

func (a *Actor) clearPlayback() {
	a.output.Clear()
}

func (a *Actor) resetPlaybackLocked() {
	a.stopPositionLoop()
	a.stream = nil
	a.path = ""
	a.durationSec = 0
	a.positionSec = 0
}

func (a *Actor) haltPlayback() {
	a.mu.Lock()
	oldStream := a.stream
	a.resetPlaybackLocked()
	a.mu.Unlock()
	if oldStream != nil {
		_ = oldStream.Close()
		a.clearPlayback()
		if a.viz != nil {
			a.viz.SetPaused(true)
		}
	}
}

func volumeGain(pct int) float64 {
	if pct <= 0 {
		return 0
	}
	return math.Pow(2, (float64(pct)/100)*6-6)
}

func (a *Actor) applyPaused(paused bool) {
	a.output.SetPaused(paused)
	if a.viz != nil {
		a.viz.SetPaused(paused)
	}
}

func (a *Actor) playChain() Streamer {
	src := Streamer(a.stream)
	if a.eq != nil {
		src = a.eq.Wrap(src)
	}
	return viz.Tap(src, a.viz)
}

func (a *Actor) SetEQ(cfg EQConfig) {
	if a.eq != nil {
		a.eq.SetConfig(cfg)
	}
}

func (a *Actor) EQConfig() EQConfig {
	if a.eq == nil {
		return FlatEQ()
	}
	return a.eq.Config()
}

func (a *Actor) livePositionLocked() float64 {
	if a.paused {
		return a.positionSec
	}
	pos := a.playbackAnchorSec + time.Since(a.playbackAnchorTime).Seconds()
	if a.durationSec > 0 && pos > a.durationSec {
		pos = a.durationSec
	}
	return pos
}

func (a *Actor) Stop() {
	a.dispatch(func() {
		a.mu.Lock()
		a.queue = nil
		a.index = 0
		a.shuffleOrd = nil
		a.bumpQueueRevLocked()
		a.mu.Unlock()
		a.haltPlayback()
		a.emit()
	})
}

func (a *Actor) buildStatusLocked() Status {
	state := "playing"
	if a.path == "" {
		state = "stopped"
	} else if a.paused {
		state = "paused"
	}
	pos := a.index
	if pos < 0 {
		pos = 0
	}
	count := len(a.queue)
	st := Status{
		State:         state,
		Path:          a.path,
		Title:         titleFromPath(a.path),
		Position:      a.positionSec,
		Duration:      a.durationSec,
		Volume:        a.volumePct,
		Shuffle:       a.shuffle,
		Repeat:        a.repeat,
		PlaylistPos:   pos,
		PlaylistCount: count,
		QueueRevision: a.queueRev,
	}
	return st.WithLabels()
}

func supportedPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if IsSupportedPath(p) {
			out = append(out, p)
		}
	}
	return out
}

func indexOrFirst(paths []string, want string) int {
	return max(slices.Index(paths, want), 0)
}

func (a *Actor) ReplaceQueue(paths []string, startPath string) error {
	return a.dispatchErr(func() error {
		filtered := supportedPaths(paths)
		if len(filtered) == 0 {
			return nil
		}
		idx := indexOrFirst(filtered, startPath)
		a.mu.Lock()
		if a.path != "" && a.path == startPath && !a.paused &&
			a.index == idx && slices.Equal(filtered, a.queue) {
			a.mu.Unlock()
			return nil
		}
		a.queue = filtered
		a.index = idx
		a.shuffleOrd = nil
		a.bumpQueueRevLocked()
		a.mu.Unlock()
		return a.loadCurrent()
	})
}

// SetQueue replaces the queue and cursor without loading audio. If startPath is in
// the queue it becomes the current index; otherwise the index is unchanged when
// possible, or reset to 0.
func (a *Actor) SetQueue(paths []string, startPath string) error {
	return a.dispatchErr(func() error {
		filtered := supportedPaths(paths)
		if len(filtered) == 0 {
			return nil
		}
		a.mu.Lock()
		want := startPath
		if want == "" {
			want = a.path
		}
		a.queue = filtered
		a.index = indexOrFirst(filtered, want)
		a.shuffleOrd = nil
		a.bumpQueueRevLocked()
		a.mu.Unlock()
		a.emit()
		return nil
	})
}

func (a *Actor) UpNextPaths(limit int) []string {
	if limit <= 0 {
		limit = 3
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.queue) <= 1 || a.index < 0 || a.index >= len(a.queue) {
		return nil
	}
	out := make([]string, 0, limit)
	if a.shuffle {
		if len(a.shuffleOrd) != len(a.queue) {
			a.shuffleOrd = shuffledOrder(len(a.queue), a.index)
		}
		pos := max(slices.Index(a.shuffleOrd, a.index), 0)
		for i := pos + 1; i < len(a.shuffleOrd) && len(out) < limit; i++ {
			out = append(out, a.queue[a.shuffleOrd[i]])
		}
	} else {
		for i := a.index + 1; i < len(a.queue) && len(out) < limit; i++ {
			out = append(out, a.queue[i])
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *Actor) PlayPathInQueue(path string) error {
	return a.dispatchErr(func() error {
		if !IsSupportedPath(path) {
			return fmt.Errorf("unsupported path: %s", path)
		}
		a.mu.Lock()
		idx := slices.Index(a.queue, path)
		if idx < 0 {
			a.mu.Unlock()
			return fmt.Errorf("path not in queue")
		}
		a.index = idx
		a.mu.Unlock()
		return a.loadCurrent()
	})
}

func (a *Actor) QueuePaths() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]string, len(a.queue))
	copy(out, a.queue)
	return out
}

func (a *Actor) RelocatePath(from, to string) error {
	if from == "" || to == "" || from == to {
		return nil
	}
	return a.dispatchErr(func() error {
		a.mu.Lock()
		changed := false
		for i, p := range a.queue {
			if p == from {
				a.queue[i] = to
				changed = true
			}
		}
		if changed {
			a.bumpQueueRevLocked()
		}
		current := a.path == from
		loaded := current && a.stream != nil
		if current {
			a.path = to
		}
		pos := a.livePositionLocked()
		paused := a.paused
		a.mu.Unlock()
		if loaded {
			return a.loadPath(to, pos, paused)
		}
		if changed || current {
			a.emit()
		}
		return nil
	})
}

func (a *Actor) QueueRevision() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.queueRev
}

func (a *Actor) bumpQueueRevLocked() {
	a.queueRev++
}

func (a *Actor) Append(paths []string) {
	a.dispatch(func() {
		a.mu.Lock()
		a.queue = append(a.queue, supportedPaths(paths)...)
		a.bumpQueueRevLocked()
		a.mu.Unlock()
		a.emit()
	})
}

func (a *Actor) MoveQueueIndex(index, delta int) error {
	if delta != -1 && delta != 1 {
		return fmt.Errorf("invalid queue move delta: %d", delta)
	}
	return a.dispatchErr(func() error {
		a.mu.Lock()
		if a.shuffle {
			a.mu.Unlock()
			return fmt.Errorf("cannot reorder queue while shuffle is on")
		}
		n := len(a.queue)
		j := index + delta
		if index < 0 || index >= n || j < 0 || j >= n {
			a.mu.Unlock()
			return fmt.Errorf("cannot move track")
		}
		a.queue[index], a.queue[j] = a.queue[j], a.queue[index]
		switch a.index {
		case index:
			a.index = j
		case j:
			a.index = index
		}
		a.bumpQueueRevLocked()
		a.mu.Unlock()
		a.emit()
		return nil
	})
}

func (a *Actor) Toggle() error {
	return a.dispatchErr(func() error {
		a.mu.Lock()
		if a.path == "" {
			a.mu.Unlock()
			return nil
		}
		if a.paused && a.stream == nil {
			pos := a.positionSec
			a.mu.Unlock()
			return a.loadCurrentAt(pos, false)
		}
		if a.paused {
			a.playbackAnchorTime = time.Now()
		} else {
			a.positionSec = a.livePositionLocked()
			a.playbackAnchorSec = a.positionSec
		}
		a.paused = !a.paused
		paused := a.paused
		a.mu.Unlock()
		a.applyPaused(paused)
		a.emit()
		return nil
	})
}

func (a *Actor) Seek(seconds float64) error {
	return a.dispatchErr(func() error {
		return a.seekPlayback(seconds)
	})
}

func (a *Actor) Next() error {
	return a.dispatchErr(func() error {
		a.mu.Lock()
		a.detached = false
		if len(a.queue) == 0 {
			a.mu.Unlock()
			return nil
		}
		a.index = a.nextIndexLocked(1)
		a.mu.Unlock()
		return a.loadCurrent()
	})
}

func (a *Actor) Prev() error {
	return a.dispatchErr(func() error {
		a.mu.Lock()
		if a.detached && a.positionSec <= 3 {
			resume := a.resumePos
			a.detached = false
			a.mu.Unlock()
			return a.loadCurrentAt(resume, false)
		}
		if len(a.queue) == 0 {
			a.mu.Unlock()
			return nil
		}
		if a.positionSec > 3 {
			a.mu.Unlock()
			return a.seekPlayback(0)
		}
		a.detached = false
		a.index = a.nextIndexLocked(-1)
		a.mu.Unlock()
		return a.loadCurrent()
	})
}

// PlayDetached plays path without changing the queue. When it ends, playback
// returns to the queue track that was interrupted.
func (a *Actor) PlayDetached(path string) error {
	return a.dispatchErr(func() error {
		if !IsSupportedPath(path) {
			return fmt.Errorf("unsupported path: %s", path)
		}
		a.mu.Lock()
		if !a.detached {
			a.resumePos = a.positionSec
		}
		a.detached = true
		a.mu.Unlock()
		return a.loadPath(path, 0, false)
	})
}

func (a *Actor) SetVolume(pct int) {
	a.dispatch(func() {
		a.mu.Lock()
		a.volumePct = min(max(pct, 0), 100)
		pct = a.volumePct
		a.mu.Unlock()
		a.output.SetVolume(volumeGain(pct))
		a.emit()
	})
}

func (a *Actor) AdjustVolume(delta int) {
	a.dispatch(func() {
		a.mu.Lock()
		a.volumePct = min(max(a.volumePct+delta, 0), 100)
		pct := a.volumePct
		a.mu.Unlock()
		a.output.SetVolume(volumeGain(pct))
		a.emit()
	})
}

func (a *Actor) SetRepeat(on bool) {
	a.dispatch(func() {
		a.mu.Lock()
		a.repeat = on
		a.mu.Unlock()
		a.emit()
	})
}

func (a *Actor) SetShuffle(on bool) {
	a.dispatch(func() {
		a.mu.Lock()
		a.shuffle = on
		if on {
			a.shuffleOrd = shuffledOrder(len(a.queue), a.index)
		} else {
			a.shuffleOrd = nil
		}
		a.mu.Unlock()
		a.emit()
	})
}

func (a *Actor) Restore(paths []string, startPath string, position float64) error {
	return a.dispatchErr(func() error {
		filtered := supportedPaths(paths)
		if len(filtered) == 0 {
			return nil
		}
		idx := indexOrFirst(filtered, startPath)
		path := filtered[idx]
		if position < 0 {
			position = 0
		}
		a.mu.Lock()
		a.queue = filtered
		a.index = idx
		a.shuffleOrd = nil
		a.bumpQueueRevLocked()
		a.path = path
		a.positionSec = position
		a.playbackAnchorSec = position
		a.paused = true
		a.stream = nil
		a.mu.Unlock()
		a.emit()
		return nil
	})
}

func (a *Actor) loadCurrent() error {
	return a.loadCurrentAt(0, false)
}

func (a *Actor) loadCurrentAt(position float64, paused bool) error {
	a.mu.Lock()
	a.detached = false
	if len(a.queue) == 0 {
		a.mu.Unlock()
		a.haltPlayback()
		a.emit()
		return nil
	}
	if a.index < 0 || a.index >= len(a.queue) {
		a.index = 0
	}
	path := a.queue[a.index]
	a.mu.Unlock()
	return a.loadPath(path, position, paused)
}

func (a *Actor) loadPath(path string, position float64, paused bool) error {
	stream, format, err := OpenDecoder(path)
	if err != nil {
		return err
	}
	durationSec := float64(stream.Len()) / float64(format.SampleRate)

	a.mu.Lock()
	oldStream := a.stream
	a.resetPlaybackLocked()
	a.loadGen++
	loadGen := a.loadGen
	a.stream = stream
	a.sourceSampleRate = format.SampleRate
	a.path = path
	a.durationSec = durationSec
	a.positionSec = 0
	a.playbackAnchorSec = 0
	a.playbackAnchorTime = time.Now()
	a.paused = paused
	a.mu.Unlock()

	if oldStream != nil {
		_ = oldStream.Close()
		a.clearPlayback()
	}

	if err := a.ensureOutput(); err != nil {
		a.mu.Lock()
		a.resetPlaybackLocked()
		a.mu.Unlock()
		_ = stream.Close()
		a.emit()
		return fmt.Errorf("audio output init: %w", err)
	}

	if a.viz != nil {
		a.viz.ResetTrack()
	}
	if a.eq != nil {
		a.eq.Reset()
	}
	a.applyPaused(paused)
	done := make(chan struct{})
	if err := a.output.Play(a.playChain(), &a.playMu, func() { close(done) }); err != nil {
		a.mu.Lock()
		a.resetPlaybackLocked()
		a.mu.Unlock()
		_ = stream.Close()
		a.emit()
		return err
	}

	a.mu.Lock()
	a.startPositionLoopLocked()
	a.mu.Unlock()
	if position > 0 {
		_ = a.seekPlayback(position)
	} else {
		a.emit()
	}

	go a.watchPlaybackEnd(path, loadGen, done)
	return nil
}

func (a *Actor) seekPlayback(seconds float64) error {
	a.mu.Lock()
	if a.path == "" {
		a.mu.Unlock()
		return nil
	}
	if seconds < 0 {
		seconds = 0
	}
	if a.durationSec > 0 && seconds > a.durationSec {
		seconds = a.durationSec
	}
	if a.stream == nil {
		a.positionSec = seconds
		a.playbackAnchorSec = seconds
		a.mu.Unlock()
		a.emit()
		return nil
	}
	stream := a.stream
	srcRate := a.sourceSampleRate
	path := a.path
	if srcRate <= 0 {
		srcRate = outputSampleRate
	}
	samples := int(seconds * float64(srcRate))
	a.mu.Unlock()

	if max := stream.Len(); max > 0 && samples > max {
		samples = max
	}
	// Seeking restarts the decoder. That looks like EOF to the current
	// reader, so invalidate the end watcher before the read loop notices.
	a.mu.Lock()
	a.loadGen++
	loadGen := a.loadGen
	a.mu.Unlock()
	a.playMu.Lock()
	err := stream.Seek(samples)
	a.playMu.Unlock()
	if err != nil {
		return err
	}
	seconds = float64(samples) / float64(srcRate)

	if a.viz != nil {
		a.viz.ResetTrack()
	}
	if a.eq != nil {
		a.eq.Reset()
	}
	done := make(chan struct{})
	if err := a.output.Play(a.playChain(), &a.playMu, func() { close(done) }); err != nil {
		return err
	}
	go a.watchPlaybackEnd(path, loadGen, done)

	a.mu.Lock()
	a.positionSec = seconds
	a.playbackAnchorSec = seconds
	a.playbackAnchorTime = time.Now()
	a.startPositionLoopLocked()
	a.mu.Unlock()
	a.emit()
	return nil
}

func (a *Actor) watchPlaybackEnd(path string, gen uint64, done <-chan struct{}) {
	<-done
	a.dispatch(func() {
		a.mu.Lock()
		stale := a.loadGen != gen || a.path != path
		repeat := a.repeat
		detached := a.detached
		resume := a.resumePos
		a.mu.Unlock()
		if stale {
			return
		}
		if detached {
			if repeat {
				_ = a.loadPath(path, 0, false)
				return
			}
			a.mu.Lock()
			a.detached = false
			a.mu.Unlock()
			_ = a.loadCurrentAt(resume, false)
			return
		}
		if !repeat {
			a.mu.Lock()
			a.index = a.nextIndexLocked(1)
			a.mu.Unlock()
		}
		_ = a.loadCurrent()
	})
}

func (a *Actor) nextIndexLocked(step int) int {
	if len(a.queue) == 0 {
		return 0
	}
	if a.shuffle {
		if len(a.shuffleOrd) != len(a.queue) {
			a.shuffleOrd = shuffledOrder(len(a.queue), a.index)
		}
		pos := max(slices.Index(a.shuffleOrd, a.index), 0)
		pos += step
		if pos < 0 {
			pos = len(a.shuffleOrd) - 1
		}
		if pos >= len(a.shuffleOrd) {
			pos = 0
		}
		return a.shuffleOrd[pos]
	}
	next := a.index + step
	if next < 0 {
		next = len(a.queue) - 1
	}
	if next >= len(a.queue) {
		next = 0
	}
	return next
}

func (a *Actor) startPositionLoopLocked() {
	a.stopPositionLoop()
	stop := make(chan struct{})
	a.stopPos = stop
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				a.mu.Lock()
				playing := a.path != "" && !a.paused
				if playing {
					a.positionSec = a.livePositionLocked()
				}
				a.mu.Unlock()
				if playing {
					a.emit()
				}
			}
		}
	}()
}

func (a *Actor) stopPositionLoop() {
	if a.stopPos != nil {
		close(a.stopPos)
		a.stopPos = nil
	}
}

func titleFromPath(path string) string {
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func shuffledOrder(n, current int) []int {
	if n <= 0 {
		return nil
	}
	ord := rand.Perm(n)
	if i := slices.Index(ord, current); i > 0 {
		ord[0], ord[i] = ord[i], ord[0]
	}
	return ord
}
