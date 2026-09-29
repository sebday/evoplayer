package playback

import (
	"math"
	"sync"
)

const (
	eqBandCount = 10
	eqQ         = 1.1
	eqMinDB     = -12
	eqMaxDB     = 12
	eqSampleHz  = 48000
)

// EQFreqs are the graphic-EQ centers, low to high.
var EQFreqs = [eqBandCount]float64{31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}

// EQKeys are the music.toml names for those bands.
var EQKeys = [eqBandCount]string{"b31", "b62", "b125", "b250", "b500", "b1k", "b2k", "b4k", "b8k", "b16k"}

// EQConfig is the saved equalizer. Gains are decibels, clamped to ±12.
type EQConfig struct {
	Enabled bool
	Preamp  float64
	Gains   [eqBandCount]float64
}

// FlatEQ is on, with every slider at 0 dB.
func FlatEQ() EQConfig {
	return EQConfig{Enabled: true}
}

// EQ is a 10-band peaking equalizer plus preamp. SetConfig applies on the next sample.
type EQ struct {
	mu      sync.Mutex
	enabled bool
	preamp  float64
	preDB   float64
	gains   [eqBandCount]float64
	bypass  bool
	bands   [eqBandCount]biquad
	live    [eqBandCount]bool
}

func NewEQ() *EQ {
	e := &EQ{enabled: true, preamp: 1}
	e.rebuild()
	return e
}

func (e *EQ) Config() EQConfig {
	e.mu.Lock()
	defer e.mu.Unlock()
	return EQConfig{Enabled: e.enabled, Preamp: e.preDB, Gains: e.gains}
}

func (e *EQ) SetConfig(cfg EQConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = cfg.Enabled
	e.preDB = clampDB(cfg.Preamp)
	e.preamp = math.Pow(10, e.preDB/20)
	for i := range e.gains {
		e.gains[i] = clampDB(cfg.Gains[i])
	}
	e.rebuild()
}

// Reset clears filter memory so a seek or a new track does not click.
func (e *EQ) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.bands {
		e.bands[i].reset()
	}
}

func (e *EQ) Wrap(src Streamer) Streamer {
	if src == nil || e == nil {
		return src
	}
	return &eqStreamer{src: src, eq: e}
}

func (e *EQ) rebuild() {
	flat := math.Abs(e.preDB) < 0.05
	for i, gain := range e.gains {
		if math.Abs(gain) < 0.05 {
			e.live[i] = false
			continue
		}
		flat = false
		e.live[i] = true
		e.bands[i].setPeaking(eqSampleHz, EQFreqs[i], eqQ, gain)
	}
	e.bypass = !e.enabled || flat
}

func (e *EQ) process(samples [][2]float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.bypass {
		return
	}
	pre := e.preamp
	for i := range samples {
		l := softClip(samples[i][0] * pre)
		r := softClip(samples[i][1] * pre)
		for b := range e.bands {
			if !e.live[b] {
				continue
			}
			l = e.bands[b].step(0, l)
			r = e.bands[b].step(1, r)
		}
		samples[i][0] = softClip(l)
		samples[i][1] = softClip(r)
	}
}

func clampDB(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < eqMinDB {
		return eqMinDB
	}
	if v > eqMaxDB {
		return eqMaxDB
	}
	return v
}

func softClip(x float64) float64 {
	if x > 1 {
		return 1 + math.Tanh(x-1)*0.2
	}
	if x < -1 {
		return -1 + math.Tanh(x+1)*0.2
	}
	return x
}

type biquad struct {
	b0, b1, b2 float64
	a1, a2     float64
	x1, x2     [2]float64
	y1, y2     [2]float64
}

func (b *biquad) reset() {
	*b = biquad{b0: b.b0, b1: b.b1, b2: b.b2, a1: b.a1, a2: b.a2}
}

func (b *biquad) setPeaking(fs, f0, q, gainDB float64) {
	a := math.Pow(10, gainDB/40)
	w0 := 2 * math.Pi * f0 / fs
	cosw := math.Cos(w0)
	alpha := math.Sin(w0) / (2 * q)
	b0 := 1 + alpha*a
	b1 := -2 * cosw
	b2 := 1 - alpha*a
	a0 := 1 + alpha/a
	a1 := -2 * cosw
	a2 := 1 - alpha/a
	b.b0 = b0 / a0
	b.b1 = b1 / a0
	b.b2 = b2 / a0
	b.a1 = a1 / a0
	b.a2 = a2 / a0
}

func (b *biquad) step(ch int, x float64) float64 {
	y := b.b0*x + b.b1*b.x1[ch] + b.b2*b.x2[ch] - b.a1*b.y1[ch] - b.a2*b.y2[ch]
	b.x2[ch] = b.x1[ch]
	b.x1[ch] = x
	b.y2[ch] = b.y1[ch]
	b.y1[ch] = y
	return y
}

type eqStreamer struct {
	src Streamer
	eq  *EQ
}

func (s *eqStreamer) Stream(samples [][2]float64) (int, bool) {
	n, ok := s.src.Stream(samples)
	if n > 0 {
		s.eq.process(samples[:n])
	}
	return n, ok
}

func (s *eqStreamer) Err() error {
	return s.src.Err()
}
