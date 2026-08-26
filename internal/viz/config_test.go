package viz

import (
	"testing"
	"time"
)

func TestDefaultConfigMatchesPunchyPreset(t *testing.T) {
	c := DefaultConfig()
	if c.Sensitivity != 145 || c.Autosens != 2 {
		t.Fatalf("unexpected sensitivity defaults: %+v", c)
	}
	if c.NoiseReduction != 34 || c.Monstercat != 1 {
		t.Fatalf("unexpected smoothing defaults: %+v", c)
	}
	if c.FrameRate != 45 || c.LowCutoff != 50 || c.HighCutoff != 10000 {
		t.Fatalf("unexpected spectrum defaults: %+v", c)
	}
}

func TestConfigNormalizesCutoffsToSampleRate(t *testing.T) {
	c := Config{
		Sensitivity:    999,
		Autosens:       9,
		NoiseReduction: 999,
		Monstercat:     -1,
		FrameRate:      999,
		LowCutoff:      10000,
		HighCutoff:     30000,
	}.normalized(48000)

	if c.Sensitivity != 400 || c.Autosens != 5 || c.NoiseReduction != 100 {
		t.Fatalf("unexpected normalized levels: %+v", c)
	}
	if c.Monstercat != 0 || c.FrameRate != 60 {
		t.Fatalf("unexpected normalized smoothing: %+v", c)
	}
	if c.HighCutoff != 24000 || c.LowCutoff >= c.HighCutoff {
		t.Fatalf("unexpected normalized cutoffs: %+v", c)
	}
}

func TestEmitIntervalFromFrameRate(t *testing.T) {
	d := Config{FrameRate: 30}.emitInterval()
	if d < 32*time.Millisecond || d > 34*time.Millisecond {
		t.Fatalf("expected ~33ms, got %v", d)
	}
}
