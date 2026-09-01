package playback

import (
	"fmt"
	"math/rand/v2"
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
	volumePct          int
	paused             bool
	path               string
	durationSec        float64
	positionSec        float64
	playbackAnchorSec  float64
	playbackAnchorTime time.Time
	stream             StreamSeekCloser
	sourceSampleRate   SampleRate
	notify             func(Status)
	stopPos            chan struct{}
	output             PlayerOutput
	outputOnce         sync.Once
	outputInitErr      error
	loadGen            uint64
	cmdCh              chan func()
	viz                *viz.Analyzer
}

func NewActor(notify func(Status)) *Actor {
	a := &Actor{
		volumePct: 100,
		notify:    notify,
		stopPos:   make(chan struct{}, 1),
		cmdCh:     make(chan func(), 64),
		viz:       viz.NewAnalyzer(float64(outputSampleRate)),
	}
	a.viz.SetDelayFunc(func() int {
		delay := a.output.PresentationDelaySamples() - vizPaintLeadSamples
		if delay < 0 {
			return 0
		}
		return delay
	})
	go a.workerLoop()
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

func (a *Actor) emit() {
	a.mu.RLock()
	st := a.buildStatusLocked()
	fn := a.notify
	a.mu.RUnlock()
	if fn != nil {
		go fn(st)
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
	}
}

func (a *Actor) volumeGain() float64 {
	if a.volumePct <= 0 {
		return 0
	}
	return (float64(a.volumePct)/100)*6 - 6
}

func (a *Actor) volumeSilent() bool {
	return a.volumePct <= 0
}

func (a *Actor) playChain() Streamer {
	ctrl := pausedStreamer(a.stream, &a.paused)
	tapped := viz.Tap(ctrl, a.viz)
	return volumeStreamer(tapped, a.volumeGain, a.volumeSilent)
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
		PlaylistPos:   pos,
		PlaylistCount: count,
		QueueRevision: a.queueRev,
	}
	return st.WithLabels()
}

func (a *Actor) ReplaceQueue(paths []string, startPath string) error {
	return a.dispatchErr(func() error {
		filtered := make([]string, 0, len(paths))
		for _, p := range paths {
			if IsSupportedPath(p) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return nil
		}
		idx := 0
		for i, p := range filtered {
			if p == startPath {
				idx = i
				break
			}
		}
		a.mu.Lock()
		if a.path != "" && a.path == startPath && !a.paused &&
			len(filtered) == len(a.queue) && a.index == idx {
			same := true
			for i := range filtered {
				if filtered[i] != a.queue[i] {
					same = false
					break
				}
			}
			if same {
				a.mu.Unlock()
				return nil
			}
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
		filtered := make([]string, 0, len(paths))
		for _, p := range paths {
			if IsSupportedPath(p) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return nil
		}
		idx := 0
		if startPath != "" {
			for i, p := range filtered {
				if p == startPath {
					idx = i
					break
				}
			}
		} else {
			a.mu.Lock()
			cur := a.path
			a.mu.Unlock()
			for i, p := range filtered {
				if p == cur {
					idx = i
					break
				}
			}
		}
		a.mu.Lock()
		a.queue = filtered
		a.index = idx
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
		pos := 0
		for i, qi := range a.shuffleOrd {
			if qi == a.index {
				pos = i
				break
			}
		}
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
		idx := -1
		for i, p := range a.queue {
			if p == path {
				idx = i
				break
			}
		}
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
		playing := false
		for i, p := range a.queue {
			if p == from {
				a.queue[i] = to
				changed = true
			}
		}
		if a.path == from {
			a.path = to
			playing = true
		}
		if changed {
			a.bumpQueueRevLocked()
		}
		pos := a.positionSec
		paused := a.paused
		a.mu.Unlock()
		if playing && !paused {
			return a.loadCurrentAt(pos, false)
		}
		if changed || playing {
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
		for _, p := range paths {
			if IsSupportedPath(p) {
				a.queue = append(a.queue, p)
			}
		}
		a.bumpQueueRevLocked()
		a.mu.Unlock()
		a.emit()
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
		if !a.paused {
			pos := a.playbackAnchorSec + time.Since(a.playbackAnchorTime).Seconds()
			if a.durationSec > 0 && pos > a.durationSec {
				pos = a.durationSec
			}
			a.positionSec = pos
			a.playbackAnchorSec = pos
		} else {
			a.playbackAnchorTime = time.Now()
		}
		a.paused = !a.paused
		a.mu.Unlock()
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
		if len(a.queue) == 0 {
			a.mu.Unlock()
			return nil
		}
		if a.positionSec > 3 {
			a.mu.Unlock()
			return a.restartPlaybackFrom(0)
		}
		a.index = a.nextIndexLocked(-1)
		a.mu.Unlock()
		return a.loadCurrent()
	})
}

func (a *Actor) SetVolume(pct int) {
	a.dispatch(func() {
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		a.mu.Lock()
		a.volumePct = pct
		a.mu.Unlock()
		a.emit()
	})
}

func (a *Actor) AdjustVolume(delta int) {
	a.dispatch(func() {
		a.mu.Lock()
		a.volumePct += delta
		if a.volumePct < 0 {
			a.volumePct = 0
		}
		if a.volumePct > 100 {
			a.volumePct = 100
		}
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

func (a *Actor) syncVizDelay() {
	if a.viz == nil || !a.viz.Wanted() {
		return
	}
	delay := a.output.PresentationDelaySamples() - vizPaintLeadSamples
	if delay < 0 {
		delay = 0
	}
	a.viz.SetPresentationDelay(delay)
}

func (a *Actor) Restore(paths []string, startPath string, position float64) error {
	return a.dispatchErr(func() error {
		filtered := make([]string, 0, len(paths))
		for _, p := range paths {
			if IsSupportedPath(p) {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return nil
		}
		idx := 0
		for i, p := range filtered {
			if p == startPath {
				idx = i
				break
			}
		}
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

	stream, format, err := OpenDecoder(path)
	if err != nil {
		return err
	}

	durationSec := DurationForPath(path)
	if durationSec <= 0 {
		if seeker, ok := stream.(StreamSeeker); ok && seeker.Len() > 0 {
			durationSec = float64(seeker.Len()) / float64(format.SampleRate)
		}
	}

	if a.viz != nil {
		a.viz.ResetTrack()
	}
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

	done := make(chan struct{})
	chain := a.playChain()
	if err := a.output.Play(chain, &a.playMu, func() { close(done) }); err != nil {
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
	a.syncVizDelay()
	if position > 0 {
		_ = a.seekPlayback(position)
	} else {
		a.emit()
	}

	go a.watchPlaybackEnd(path, loadGen, done)
	return nil
}

func (a *Actor) restartPlaybackFrom(seconds float64) error {
	return a.seekPlayback(seconds)
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
	if srcRate <= 0 {
		srcRate = outputSampleRate
	}
	samples := int(seconds * float64(srcRate))
	a.mu.Unlock()

	if seeker, ok := stream.(StreamSeeker); ok {
		if samples < 0 {
			samples = 0
		}
		if max := seeker.Len(); max > 0 && samples > max {
			samples = max
		}
		a.playMu.Lock()
		err := seeker.Seek(samples)
		a.playMu.Unlock()
		if err != nil {
			return err
		}
		seconds = float64(samples) / float64(srcRate)
	}

	a.mu.Lock()
	a.positionSec = seconds
	a.playbackAnchorSec = seconds
	a.playbackAnchorTime = time.Now()
	a.mu.Unlock()
	if srcRate <= 0 {
		srcRate = outputSampleRate
	}
	a.output.ReanchorPresentation(seconds, srcRate)
	if a.viz != nil {
		a.viz.ResetTrack()
	}
	a.emit()
	return nil
}

func (a *Actor) watchPlaybackEnd(path string, gen uint64, done <-chan struct{}) {
	<-done
	a.dispatch(func() {
		a.mu.Lock()
		stale := a.loadGen != gen || a.path != path
		a.mu.Unlock()
		if stale {
			return
		}
		a.mu.Lock()
		a.index = a.nextIndexLocked(1)
		a.mu.Unlock()
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
		pos := 0
		for i, idx := range a.shuffleOrd {
			if idx == a.index {
				pos = i
				break
			}
		}
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
				if a.path != "" && !a.paused {
					pos := a.playbackAnchorSec + time.Since(a.playbackAnchorTime).Seconds()
					if a.durationSec > 0 && pos > a.durationSec {
						pos = a.durationSec
					}
					a.positionSec = pos
				}
				a.mu.Unlock()
				a.syncVizDelay()
				a.emit()
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
	base := path
	if i := len(base) - 1; i >= 0 {
		for j := i; j >= 0; j-- {
			if base[j] == '/' {
				base = base[j+1:]
				break
			}
		}
	}
	if dot := len(base) - 1; dot > 0 {
		for i := dot; i >= 0; i-- {
			if base[i] == '.' {
				return base[:i]
			}
		}
	}
	return base
}

func shuffledOrder(n, current int) []int {
	if n <= 0 {
		return nil
	}
	ord := make([]int, n)
	for i := range ord {
		ord[i] = i
	}
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(current+1)))
	r.Shuffle(len(ord), func(i, j int) { ord[i], ord[j] = ord[j], ord[i] })
	return ord
}
