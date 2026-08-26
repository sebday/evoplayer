package viz

import (
	"math"
	"sync"
	"time"
)

const (
	BarCount    = 100
	FFTSize     = 4096
	BassFFTSize = FFTSize * 2
	bassCutoff  = 100.0

	// ringCapacity holds ~1.4s @ 48kHz so viz can compensate for oto output delay.
	ringCapacity         = 65536
	maxPresentationDelay = ringCapacity - BassFFTSize

	punchWindowSec = 0.03
	punchAttack    = 0.85
	punchRelease   = 0.32
	punchBassBars  = 18
	punchFloor     = 0.22
	punchBassBoost = 0.55
)

type frequencyBand struct {
	lower   int
	upper   int
	useBass bool
	eq      float64
}

type Analyzer struct {
	mu         sync.Mutex
	sampleRate float64
	wanted     bool
	onUpdate   func([]float32)
	cfg        Config

	manualSensitivity float64
	autosensRate      int
	noiseReduction    float64
	monstercat        float64
	emitMin           time.Duration

	ring           []float64
	ringPos        int
	ringFilled     int
	frameHadSignal bool

	window     []float64
	bassWindow []float64
	re         []float64
	im         []float64
	bassRe     []float64
	bassIm     []float64
	bands      []frequencyBand
	raw        []float64

	previous []float64
	memory   []float64
	peaks    []float64
	fall     []float64
	display  []float32

	autoSensitivity float64
	sensitivityInit bool

	presentationDelay int
	punchEnv          float64
	delayFn           func() int

	tickStop chan struct{}
	tickWG   sync.WaitGroup
}

func NewAnalyzer(sampleRate float64) *Analyzer {
	a := &Analyzer{
		sampleRate: sampleRate,
		ring:       make([]float64, ringCapacity),
		window:     hannWindow(FFTSize),
		bassWindow: hannWindow(BassFFTSize),
		re:         make([]float64, FFTSize),
		im:         make([]float64, FFTSize),
		bassRe:     make([]float64, BassFFTSize),
		bassIm:     make([]float64, BassFFTSize),
		bands:      make([]frequencyBand, BarCount),
		raw:        make([]float64, BarCount),
		previous:   make([]float64, BarCount),
		memory:     make([]float64, BarCount),
		peaks:      make([]float64, BarCount),
		fall:       make([]float64, BarCount),
		display:    make([]float32, BarCount),
	}
	a.ApplyConfig(DefaultConfig())
	return a
}

func hannWindow(n int) []float64 {
	w := make([]float64, n)
	if n <= 1 {
		if n == 1 {
			w[0] = 1
		}
		return w
	}
	for i := range w {
		w[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
	return w
}

func (a *Analyzer) initBandsLocked() {
	low := float64(a.cfg.LowCutoff)
	high := float64(a.cfg.HighCutoff)
	ratio := high / low
	for i := range a.bands {
		lowerHz := low * math.Pow(ratio, float64(i)/BarCount)
		upperHz := low * math.Pow(ratio, float64(i+1)/BarCount)
		useBass := lowerHz < bassCutoff
		size := FFTSize
		if useBass {
			size = BassFFTSize
		}
		binHz := a.sampleRate / float64(size)
		lower := int(math.Floor(lowerHz / binHz))
		upper := int(math.Ceil(upperHz/binHz)) - 1
		if lower < 1 {
			lower = 1
		}
		maxBin := size/2 - 1
		if lower > maxBin {
			lower = maxBin
		}
		if upper < lower {
			upper = lower
		}
		if upper > maxBin {
			upper = maxBin
		}
		width := upper - lower + 1
		// CAVA's linear-amplitude EQ compensates for FFT size, grouped
		// bandwidth, and the natural rolloff in higher frequencies.
		eq := math.Pow(2, -28)
		eq *= math.Pow(upperHz, 0.85)
		eq /= math.Log2(float64(size))
		eq /= float64(width)
		a.bands[i] = frequencyBand{
			lower:   lower,
			upper:   upper,
			useBass: useBass,
			eq:      eq,
		}
	}
}

func (a *Analyzer) resetProcessingLocked() {
	clear(a.ring)
	clear(a.raw)
	clear(a.previous)
	clear(a.memory)
	clear(a.peaks)
	clear(a.fall)
	clear(a.display)
	a.ringPos = 0
	a.ringFilled = 0
	a.frameHadSignal = false
	a.autoSensitivity = 1
	a.sensitivityInit = true
	a.punchEnv = 0
}

func (a *Analyzer) stopTicker() {
	a.mu.Lock()
	stop := a.tickStop
	a.tickStop = nil
	a.mu.Unlock()
	if stop == nil {
		return
	}
	close(stop)
	a.tickWG.Wait()
}

func (a *Analyzer) startTicker() {
	a.stopTicker()
	a.mu.Lock()
	if !a.wanted {
		a.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	a.tickStop = stop
	interval := a.emitMin
	a.tickWG.Add(1)
	a.mu.Unlock()
	go func() {
		defer a.tickWG.Done()
		a.tickLoop(stop, interval)
	}()
}

func (a *Analyzer) tickLoop(stop <-chan struct{}, interval time.Duration) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			a.mu.Lock()
			if !a.wanted {
				a.mu.Unlock()
				return
			}
			interval = a.emitMin
			var emit []float32
			var fn func([]float32)
			if a.ringFilled >= BassFFTSize {
				a.analyzeLocked()
				a.frameHadSignal = false
				emit, fn = a.emitLocked()
			}
			a.mu.Unlock()
			if fn != nil {
				fn(emit)
			}
			timer.Reset(interval)
		}
	}
}

func clampPresentationDelay(samples int) int {
	if samples < 0 {
		samples = 0
	}
	if samples > maxPresentationDelay {
		samples = maxPresentationDelay
	}
	return samples
}

func (a *Analyzer) SetPresentationDelay(samples int) {
	samples = clampPresentationDelay(samples)
	a.mu.Lock()
	a.presentationDelay = samples
	a.mu.Unlock()
}

func (a *Analyzer) SetDelayFunc(fn func() int) {
	a.mu.Lock()
	a.delayFn = fn
	a.mu.Unlock()
}

func (a *Analyzer) refreshDelayLocked() {
	if a.delayFn == nil {
		return
	}
	a.presentationDelay = clampPresentationDelay(a.delayFn())
}

func (a *Analyzer) ResetTrack() {
	a.mu.Lock()
	a.resetProcessingLocked()
	a.mu.Unlock()
}

func (a *Analyzer) SetWanted(on bool) {
	a.mu.Lock()
	if a.wanted == on {
		a.mu.Unlock()
		return
	}
	if !on {
		a.wanted = false
		a.mu.Unlock()
		a.stopTicker()
		return
	}
	a.wanted = true
	a.resetProcessingLocked()
	a.mu.Unlock()
	a.startTicker()
}

func (a *Analyzer) SetOnUpdate(fn func([]float32)) {
	a.mu.Lock()
	a.onUpdate = fn
	a.mu.Unlock()
}

func (a *Analyzer) Snapshot() []float32 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.wanted && a.ringFilled >= BassFFTSize {
		a.analyzeLocked()
		a.frameHadSignal = false
	}
	out := make([]float32, BarCount)
	copy(out, a.display)
	return out
}

func (a *Analyzer) Feed(samples [][2]float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.wanted {
		return
	}
	for _, sample := range samples {
		mono := (sample[0] + sample[1]) * 0.5
		a.ring[a.ringPos] = mono
		a.ringPos = (a.ringPos + 1) % len(a.ring)
		if a.ringFilled < len(a.ring) {
			a.ringFilled++
		}
		if math.Abs(mono) > 1e-12 {
			a.frameHadSignal = true
		}
	}
}

func (a *Analyzer) emitLocked() ([]float32, func([]float32)) {
	if a.onUpdate == nil {
		return nil, nil
	}
	levels := make([]float32, BarCount)
	copy(levels, a.display)
	return levels, a.onUpdate
}

func ringIndex(idx, size int) int {
	if size <= 0 {
		return 0
	}
	idx %= size
	if idx < 0 {
		idx += size
	}
	return idx
}

func (a *Analyzer) clampedDelayLocked() int {
	delay := a.presentationDelay
	if delay < 0 {
		delay = 0
	}
	maxDelay := min(a.ringFilled-1, len(a.ring)-1)
	if maxDelay < 0 {
		maxDelay = 0
	}
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

func (a *Analyzer) analysisEndLocked() int {
	return ringIndex(a.ringPos-a.clampedDelayLocked(), len(a.ring))
}

func (a *Analyzer) fillFFTLocked(size int, window, re, im []float64) {
	clear(re)
	clear(im)
	if size <= 0 || len(re) < size || len(window) < size || len(a.ring) == 0 {
		return
	}
	available := min(a.ringFilled, size)
	if available <= 0 {
		return
	}
	// Center the window on the sample that is playing now. Samples after
	// "now" are already in the ring (oto software buffer), so we can use
	// them as lookahead instead of a causal window that lags by size/2.
	delay := a.clampedDelayLocked()
	half := size / 2
	lookahead := delay
	if lookahead > half {
		lookahead = half
	}
	end := ringIndex(a.ringPos-delay+lookahead, len(a.ring))
	dstStart := size - available
	srcStart := ringIndex(end-available, len(a.ring))
	for i := 0; i < available; i++ {
		src := ringIndex(srcStart+i, len(a.ring))
		// CAVA's linear mode expects signed 16-bit-scale input.
		re[dstStart+i] = a.ring[src] * 32768 * window[dstStart+i]
	}
	fftInPlace(re, im)
}

func (a *Analyzer) delayedPunchStatsLocked() (rms, peak float64) {
	if a.ringFilled <= 0 || len(a.ring) == 0 {
		return 0, 0
	}
	window := int(a.sampleRate * punchWindowSec)
	if window < 64 {
		window = 64
	}
	available := min(a.ringFilled, window)
	end := a.analysisEndLocked()
	srcStart := ringIndex(end-available, len(a.ring))
	var sumSq float64
	for i := 0; i < available; i++ {
		v := a.ring[ringIndex(srcStart+i, len(a.ring))]
		sumSq += v * v
		if abs := math.Abs(v); abs > peak {
			peak = abs
		}
	}
	rms = math.Sqrt(sumSq / float64(available))
	return rms, peak
}

func (a *Analyzer) applyPunchLocked() {
	rms, peak := a.delayedPunchStatsLocked()
	raw := clampFloat(rms*3.0+peak*0.25, 0, 1)
	rate := punchRelease
	if raw > a.punchEnv {
		rate = punchAttack
	}
	a.punchEnv += (raw - a.punchEnv) * rate

	gain := punchFloor + (1-punchFloor)*a.punchEnv
	bassExtra := clampFloat(a.punchEnv*punchBassBoost, 0, punchBassBoost)
	for i := range a.display {
		v := float64(a.display[i]) * gain
		if i < punchBassBars {
			taper := 1 - float64(i)/float64(punchBassBars)
			v *= 1 + bassExtra*taper
		}
		a.display[i] = float32(clampFloat(v, 0, 1))
	}
}

func (a *Analyzer) analyzeLocked() {
	a.refreshDelayLocked()
	a.fillFFTLocked(FFTSize, a.window, a.re, a.im)
	a.fillFFTLocked(BassFFTSize, a.bassWindow, a.bassRe, a.bassIm)

	for i, band := range a.bands {
		re, im := a.re, a.im
		if band.useBass {
			re, im = a.bassRe, a.bassIm
		}
		sum := 0.0
		for bin := band.lower; bin <= band.upper; bin++ {
			sum += math.Hypot(re[bin], im[bin])
		}
		a.raw[i] = sum * band.eq
	}

	a.applyCavaDynamicsLocked(a.frameHadSignal)
	a.applyMonstercatLocked()
	a.applyPunchLocked()
}

func (a *Analyzer) applyCavaDynamicsLocked(hadSignal bool) {
	fps := float64(a.cfg.FrameRate)
	framerateMod := 66.0 / fps
	integralMod := math.Pow(framerateMod, 0.1)
	gravityMod := 0.0
	if a.noiseReduction > 0 {
		gravityMod = math.Pow(framerateMod, 2.5) * 2 / a.noiseReduction
	}
	overshoot := false

	for i, raw := range a.raw {
		level := raw
		if a.autosensRate > 0 {
			level *= a.autoSensitivity
		}

		if level < a.previous[i] && a.noiseReduction > 0.1 {
			level = a.peaks[i] * (1 - a.fall[i]*a.fall[i]*gravityMod)
			if level < 0 {
				level = 0
			}
			a.fall[i] += 0.028
		} else {
			a.peaks[i] = level
			a.fall[i] = 0
		}
		a.previous[i] = level

		level += a.memory[i] * a.noiseReduction / integralMod
		a.memory[i] = level
		if a.autosensRate > 0 && level > 1 {
			level = 1
			overshoot = true
		}

		level *= a.manualSensitivity
		a.display[i] = float32(clampFloat(level, 0, 1))
	}

	if a.autosensRate > 0 {
		if overshoot {
			a.autoSensitivity *= 1 - 0.02*framerateMod
			a.sensitivityInit = false
		} else if hadSignal {
			a.autoSensitivity *= 1 + 0.001*framerateMod*float64(a.autosensRate)
			if a.sensitivityInit {
				a.autoSensitivity *= 1 + 0.1*framerateMod
			}
		}
		a.autoSensitivity = clampFloat(a.autoSensitivity, 1e-6, 1e6)
	}
}

func (a *Analyzer) applyMonstercatLocked() {
	if a.monstercat <= 0 {
		return
	}
	decay := a.monstercat * 1.5
	if decay <= 1 {
		decay = 1.000001
	}
	for source := 0; source < len(a.display); source++ {
		peak := float64(a.display[source])
		spread := peak / decay
		for target := source - 1; target >= 0; target-- {
			if spread > float64(a.display[target]) {
				a.display[target] = float32(spread)
			}
			spread /= decay
		}
		spread = peak / decay
		for target := source + 1; target < len(a.display); target++ {
			if spread > float64(a.display[target]) {
				a.display[target] = float32(spread)
			}
			spread /= decay
		}
	}
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
