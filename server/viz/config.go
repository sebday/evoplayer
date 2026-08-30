package viz

import "time"

// Matches the old evo-cava punchy preset. Sensitivity is applied after the
// CAVA-compatible core, just as cava applies its configured sensitivity.
const (
	tuneSensitivity    = 145
	tuneAutosens       = 2
	tuneNoiseReduction = 34
	tuneMonstercat     = 1.0
	tuneFrameRate      = 30
	tuneLowCutoff      = 50
	tuneHighCutoff     = 10000
)

type Config struct {
	Sensitivity    int     `json:"sensitivity"`
	Autosens       int     `json:"autosens"`
	NoiseReduction int     `json:"noise_reduction"`
	Monstercat     float64 `json:"monstercat"`
	FrameRate      int     `json:"frame_rate"`
	LowCutoff      int     `json:"low_cutoff"`
	HighCutoff     int     `json:"high_cutoff"`
}

func DefaultConfig() Config {
	return Config{
		Sensitivity:    tuneSensitivity,
		Autosens:       tuneAutosens,
		NoiseReduction: tuneNoiseReduction,
		Monstercat:     tuneMonstercat,
		FrameRate:      tuneFrameRate,
		LowCutoff:      tuneLowCutoff,
		HighCutoff:     tuneHighCutoff,
	}
}

func (c Config) normalized(sampleRate float64) Config {
	out := c
	out.Sensitivity = clampInt(out.Sensitivity, 0, 400)
	out.Autosens = clampInt(out.Autosens, 0, 5)
	out.NoiseReduction = clampInt(out.NoiseReduction, 0, 100)
	if out.Monstercat < 0 {
		out.Monstercat = 0
	}
	if out.Monstercat > 10 {
		out.Monstercat = 10
	}
	out.FrameRate = clampInt(out.FrameRate, 20, 60)
	if out.LowCutoff < 1 {
		out.LowCutoff = tuneLowCutoff
	}
	maxHigh := int(sampleRate / 2)
	if maxHigh < 2 {
		maxHigh = tuneHighCutoff
	}
	if out.HighCutoff <= out.LowCutoff {
		out.HighCutoff = tuneHighCutoff
	}
	if out.HighCutoff > maxHigh {
		out.HighCutoff = maxHigh
	}
	if out.LowCutoff >= out.HighCutoff {
		out.LowCutoff = max(1, out.HighCutoff/2)
	}
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (c Config) sensitivityScale() float64 {
	return float64(c.Sensitivity) / 100.0
}

func (c Config) noiseReductionScale() float64 {
	return float64(c.NoiseReduction) / 100.0
}

func (c Config) emitInterval() time.Duration {
	fps := c.FrameRate
	if fps <= 0 {
		fps = tuneFrameRate
	}
	return time.Duration(float64(time.Second) / float64(fps))
}

func (a *Analyzer) ApplyConfig(c Config) {
	c = c.normalized(a.sampleRate)
	a.mu.Lock()

	a.cfg = c
	a.manualSensitivity = c.sensitivityScale()
	a.autosensRate = c.Autosens
	a.noiseReduction = c.noiseReductionScale()
	a.monstercat = c.Monstercat
	a.emitMin = c.emitInterval()
	a.initBandsLocked()
	a.resetProcessingLocked()
	wanted := a.wanted
	a.mu.Unlock()
	if wanted {
		a.startTicker()
	}
}

func (a *Analyzer) Config() Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}
