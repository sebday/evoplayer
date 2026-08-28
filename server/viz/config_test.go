package viz

import "testing"

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
