package tui

import (
	"math"
	"sync"
	"time"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/viz"
)

type vizPainter struct {
	mu      sync.Mutex
	env     paths.Env
	frames  *frameCache
	stop    chan struct{}
	playing bool
	hasWave bool
	pos     float64
	dur     float64
	posAt   time.Time
	peaks   []float64
	hold    []int
	lastKey uint64
	lastX   int
	lastOut string
	lastSalt uint64
	mode    vizMode
}

func newVizPainter(env paths.Env, frames *frameCache) *vizPainter {
	return &vizPainter{env: env, frames: frames, lastX: -2}
}

func (p *vizPainter) running() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stop != nil
}

func (p *vizPainter) setTransport(playing bool, pos, dur float64, hasWave bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playing = playing
	p.hasWave = hasWave
	p.pos = pos
	p.dur = dur
	p.posAt = time.Now()
	if playing {
		if p.stop == nil {
			p.stop = make(chan struct{})
			go p.loop(p.stop)
		}
		return
	}
	p.peaks = nil
	p.hold = nil
	p.lastKey = 0
	p.lastX = -2
	p.lastOut = ""
	p.lastSalt = 0
	if p.stop != nil {
		close(p.stop)
		p.stop = nil
	}
}

func (p *vizPainter) setMode(mode vizMode) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.mode = mode
	p.lastOut = ""
	p.lastKey = 0
	p.lastSalt = 0
	p.mu.Unlock()
}

func (p *vizPainter) loop(stop <-chan struct{}) {
	tick := time.NewTicker(vizTickInterval)
	defer tick.Stop()
	p.frame()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			p.frame()
		}
	}
}

func (p *vizPainter) frame() {
	levels := readSpectrumFrame(p.env)
	p.mu.Lock()
	playing := p.playing
	hasWave := p.hasWave
	pos, dur, posAt := p.pos, p.dur, p.posAt
	if playing {
		p.peaks, p.hold = updateVizPeaks(p.peaks, p.hold, levels)
	}
	livePeaks := p.peaks
	mode := p.mode
	p.mu.Unlock()
	if !playing {
		return
	}

	key := spectrumKey(levels)
	playFrac := waveformPlayedFrac(pos, dur)
	if dur > 0 {
		playFrac = waveformPlayedFrac(pos+time.Since(posAt).Seconds(), dur)
	}

	if p.frames == nil {
		return
	}
	p.frames.mu.Lock()
	row, col := p.frames.vizRow, p.frames.vizCol
	innerW := p.frames.vizW
	p.frames.mu.Unlock()
	if row < 1 || col < 1 || innerW < 8 {
		return
	}
	waveW := max(8, innerW-2)
	playX := wavePlayCol(playFrac, waveW)
	salt := uint64(0)
	if mode == vizModeScatter {
		salt = uint64(time.Now().UnixNano() / int64(90*time.Millisecond))
	}
	if key == p.lastKey && playX == p.lastX && salt == p.lastSalt && p.lastOut != "" {
		return
	}

	lines := composeVizLines(p.frames, levels, livePeaks, playX, hasWave, mode, salt)
	if len(lines) == 0 {
		return
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	if out == p.lastOut {
		p.lastKey = key
		p.lastX = playX
		p.lastSalt = salt
		return
	}
	p.lastKey = key
	p.lastX = playX
	p.lastSalt = salt
	p.lastOut = out
	paintVizAt(row, col, lines)
}

func spectrumKey(levels []float64) uint64 {
	h := uint64(len(levels))
	for i, v := range levels {
		h ^= math.Float64bits(v) + uint64(i)*0x9e3779b97f4a7c15
	}
	return h
}

func readSpectrumFrame(env paths.Env) []float64 {
	if env.SocketPath == "" {
		return nil
	}
	levels, err := viz.ReadFrame(viz.FramePath(env.SocketPath))
	if err != nil || len(levels) == 0 {
		return nil
	}
	return levels
}
