package viz

import (
	"math"
	"testing"
)

type stubStreamer struct {
	chunks [][][2]float64
	idx    int
}

func (s *stubStreamer) Stream(samples [][2]float64) (int, bool) {
	if s.idx >= len(s.chunks) {
		return 0, false
	}
	chunk := s.chunks[s.idx]
	s.idx++
	n := len(chunk)
	if n > len(samples) {
		n = len(samples)
	}
	copy(samples, chunk[:n])
	return n, true
}

func (s *stubStreamer) Err() error { return nil }

func TestTapPassesSamplesThrough(t *testing.T) {
	src := &stubStreamer{
		chunks: [][][2]float64{
			{{0.25, 0.5}, {0.75, 1.0}},
		},
	}
	a := NewAnalyzer(48000)
	a.SetWanted(true)
	tapped := Tap(src, a).(*tapStreamer)

	out := make([][2]float64, 2)
	n, ok := tapped.Stream(out)
	if !ok || n != 2 {
		t.Fatalf("stream n=%d ok=%v", n, ok)
	}
	if out[0] != src.chunks[0][0] || out[1] != src.chunks[0][1] {
		t.Fatalf("samples not passed through: %#v", out)
	}
}

func TestAnalyzerProducesLevelsForTone(t *testing.T) {
	a := NewAnalyzer(48000)
	a.SetWanted(true)
	freq := 440.0
	chunk := make([][2]float64, BassFFTSize*2)
	for i := range chunk {
		seconds := float64(i) / 48000
		v := math.Sin(2 * math.Pi * freq * seconds)
		chunk[i] = [2]float64{v, v}
	}
	a.Feed(chunk)

	levels := a.Snapshot()
	peak := float32(0)
	peakIdx := 0
	for i, v := range levels {
		if v > peak {
			peak = v
			peakIdx = i
		}
	}
	rawPeak, rawPeakIdx := 0.0, 0
	for i, v := range a.raw {
		if v > rawPeak {
			rawPeak = v
			rawPeakIdx = i
		}
	}
	if peak < 0.05 {
		t.Fatalf("expected non-trivial levels, peak=%v", peak)
	}
	if peakIdx < 35 || peakIdx > 50 {
		t.Fatalf("expected logarithmic 440Hz peak around bar 41, got idx=%d peak=%v raw_idx=%d raw=%v",
			peakIdx, peak, rawPeakIdx, rawPeak)
	}
}

func TestAnalyzerUsesLogarithmicBands(t *testing.T) {
	a := NewAnalyzer(48000)
	low := a.bands[0]
	mid := a.bands[BarCount/2]
	high := a.bands[BarCount-1]

	if !low.useBass {
		t.Fatal("expected low-frequency bars to use the larger bass FFT")
	}
	if mid.useBass || high.useBass {
		t.Fatal("expected mid and high bars to use the regular FFT")
	}

	midHz := float64(mid.lower) * a.sampleRate / FFTSize
	if midHz < 500 || midHz > 1000 {
		t.Fatalf("expected logarithmic midpoint near sqrt(50*10000), got %.1fHz", midHz)
	}
}

func TestAutosensitivityRisesAndBacksOff(t *testing.T) {
	a := NewAnalyzer(48000)
	cfg := DefaultConfig()
	cfg.Sensitivity = 100
	cfg.NoiseReduction = 0
	cfg.Monstercat = 0
	a.ApplyConfig(cfg)

	for i := range a.raw {
		a.raw[i] = 0.01
	}
	for range 8 {
		a.applyCavaDynamicsLocked(true)
	}
	raised := a.autoSensitivity
	if raised <= 1 {
		t.Fatalf("expected autosensitivity to rise, got %v", raised)
	}

	for i := range a.raw {
		a.raw[i] = 10
	}
	a.applyCavaDynamicsLocked(true)
	if a.autoSensitivity >= raised {
		t.Fatalf("expected overshoot to reduce autosensitivity: before=%v after=%v",
			raised, a.autoSensitivity)
	}
}

func TestCavaFalloffHoldsThenDecays(t *testing.T) {
	a := NewAnalyzer(48000)
	cfg := DefaultConfig()
	cfg.Sensitivity = 100
	cfg.Autosens = 0
	cfg.Monstercat = 0
	a.ApplyConfig(cfg)

	a.raw[20] = 0.4
	a.applyCavaDynamicsLocked(true)
	start := a.display[20]
	a.raw[20] = 0
	a.applyCavaDynamicsLocked(false)
	firstRelease := a.display[20]
	if firstRelease < start*0.9 {
		t.Fatalf("expected gravity hold on first release: start=%v release=%v", start, firstRelease)
	}

	for range 30 {
		a.applyCavaDynamicsLocked(false)
	}
	if a.display[20] >= firstRelease*0.5 {
		t.Fatalf("expected sustained silence to decay: first=%v final=%v",
			firstRelease, a.display[20])
	}
}

func TestMonstercatLinksNeighborBars(t *testing.T) {
	a := NewAnalyzer(48000)
	cfg := DefaultConfig()
	cfg.Monstercat = 1
	a.ApplyConfig(cfg)
	clear(a.display)
	a.display[50] = 1

	a.applyMonstercatLocked()

	if a.display[49] < 0.65 || a.display[49] > 0.68 {
		t.Fatalf("expected first neighbor near 1/1.5, got %v", a.display[49])
	}
	if a.display[48] < 0.43 || a.display[48] > 0.46 {
		t.Fatalf("expected second neighbor near 1/1.5², got %v", a.display[48])
	}
}

func TestFFTRoundTripEnergy(t *testing.T) {
	n := 64
	re := make([]float64, n)
	im := make([]float64, n)
	for i := range re {
		re[i] = math.Sin(2 * math.Pi * float64(i) / 8)
	}
	var energy float64
	for _, v := range re {
		energy += v * v
	}
	fftInPlace(re, im)
	var specEnergy float64
	for i := 0; i < n; i++ {
		specEnergy += re[i]*re[i] + im[i]*im[i]
	}
	if specEnergy < energy*0.5 {
		t.Fatalf("spectrum energy too low: time=%v spec=%v", energy, specEnergy)
	}
}

func TestFFTPlacesToneInExpectedBin(t *testing.T) {
	re := make([]float64, FFTSize)
	im := make([]float64, FFTSize)
	window := hannWindow(FFTSize)
	for i := range re {
		re[i] = math.Sin(2*math.Pi*440*float64(i)/48000) * window[i]
	}
	fftInPlace(re, im)
	peak, peakIdx := 0.0, 0
	for i := 1; i < len(re)/2; i++ {
		v := math.Hypot(re[i], im[i])
		if v > peak {
			peak, peakIdx = v, i
		}
	}
	if peakIdx < 36 || peakIdx > 39 {
		t.Fatalf("expected 440Hz around bin 38, got bin=%d hz=%v", peakIdx,
			float64(peakIdx)*48000/FFTSize)
	}
}

func TestTapNilSafe(t *testing.T) {
	if Tap(nil, NewAnalyzer(48000)) != nil {
		t.Fatal("nil src should stay nil")
	}
	src := &stubStreamer{chunks: [][][2]float64{{{1, 1}}}}
	if Tap(src, nil) != src {
		t.Fatal("nil analyzer should return src unchanged")
	}
}

func sumLevels(levels []float32) float64 {
	var sum float64
	for _, v := range levels {
		sum += float64(v)
	}
	return sum
}

func TestPresentationDelayShiftsAnalysisWindow(t *testing.T) {
	silence := BassFFTSize * 2
	tone := BassFFTSize * 2
	chunk := make([][2]float64, silence+tone)
	for i := silence; i < len(chunk); i++ {
		v := math.Sin(2 * math.Pi * 440 * float64(i) / 48000)
		chunk[i] = [2]float64{v, v}
	}

	onTone := NewAnalyzer(48000)
	onTone.SetWanted(true)
	onTone.SetPresentationDelay(0)
	onTone.Feed(chunk)
	before := sumLevels(onTone.Snapshot())

	// Keep the centered bass window inside the leading silence.
	onSilence := NewAnalyzer(48000)
	onSilence.SetWanted(true)
	onSilence.SetPresentationDelay(silence + BassFFTSize/2)
	onSilence.Feed(chunk)
	after := sumLevels(onSilence.Snapshot())

	if before <= after*1.5 {
		t.Fatalf("presentation delay should shift analysis window (tone=%v silence=%v)", before, after)
	}
}

func TestRingIndexNegativeModulo(t *testing.T) {
	if got := ringIndex(-48000, ringCapacity); got < 0 || got >= ringCapacity {
		t.Fatalf("ringIndex(-48000, %d) = %d, want [0,%d)", ringCapacity, got, ringCapacity)
	}
}

func TestFillFFTHighPresentationDelayNoPanic(t *testing.T) {
	a := NewAnalyzer(48000)
	a.SetWanted(true)
	a.SetPresentationDelay(48000) // larger than ring
	chunk := make([][2]float64, BassFFTSize*4)
	for i := range chunk {
		v := math.Sin(2 * math.Pi * 440 * float64(i) / 48000)
		chunk[i] = [2]float64{v, v}
	}
	a.Feed(chunk)
	a.ResetTrack()
	a.Feed(chunk)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("analyze panicked: %v", r)
		}
	}()
	a.Snapshot()
}

func TestCenteredWindowSeesImpulseAtPlaybackInstant(t *testing.T) {
	const delay = 20000
	chunk := make([][2]float64, delay+BassFFTSize)
	chunk[len(chunk)-1-delay] = [2]float64{1, 1}

	a := NewAnalyzer(48000)
	a.SetWanted(true)
	a.SetPresentationDelay(delay)
	a.Feed(chunk)
	if sumLevels(a.Snapshot()) == 0 {
		t.Fatal("expected FFT centered on playback instant to see the impulse")
	}
}

func bassSum(levels []float32) float64 {
	n := punchBassBars
	if n > len(levels) {
		n = len(levels)
	}
	var sum float64
	for _, v := range levels[:n] {
		sum += float64(v)
	}
	return sum
}

func TestPunchEnvelopeFollowsDelayedBurst(t *testing.T) {
	const (
		quietAmp = 0.06
		loudAmp  = 0.9
		freq     = 60.0
	)
	sr := 48000.0
	punchN := int(sr * punchWindowSec)
	pre := BassFFTSize + punchN
	post := punchN * 4
	burstStart := pre
	chunk := make([][2]float64, burstStart+punchN+post)
	for i := range chunk {
		amp := quietAmp
		if i >= burstStart && i < burstStart+punchN {
			amp = loudAmp
		}
		v := math.Sin(2*math.Pi*freq*float64(i)/sr) * amp
		chunk[i] = [2]float64{v, v}
	}

	delayOnBurst := len(chunk) - 1 - (burstStart + punchN/2)
	delayOnQuiet := len(chunk) - 1 - (burstStart - punchN)

	loud := NewAnalyzer(sr)
	loud.SetWanted(true)
	loud.SetPresentationDelay(delayOnBurst)
	loud.Feed(chunk)
	_ = loud.Snapshot()
	loudLevels := loud.Snapshot()

	quiet := NewAnalyzer(sr)
	quiet.SetWanted(true)
	quiet.SetPresentationDelay(delayOnQuiet)
	quiet.Feed(chunk)
	_ = quiet.Snapshot()
	quietLevels := quiet.Snapshot()

	if sumLevels(loudLevels) <= sumLevels(quietLevels)*1.2 {
		t.Fatalf("expected delayed burst to raise envelope: loud=%v quiet=%v",
			sumLevels(loudLevels), sumLevels(quietLevels))
	}
	if bassSum(loudLevels) <= bassSum(quietLevels)*1.25 {
		t.Fatalf("expected bass punch on delayed burst: loudBass=%v quietBass=%v",
			bassSum(loudLevels), bassSum(quietLevels))
	}
}
