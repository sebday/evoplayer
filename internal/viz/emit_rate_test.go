package viz

import (
	"math"
	"testing"
	"time"
)

func TestAnalyzerEmitRateWhileFeeding(t *testing.T) {
	a := NewAnalyzer(48000)
	a.SetWanted(true)
	cfg := DefaultConfig()
	cfg.FrameRate = 30
	a.ApplyConfig(cfg)

	updates := 0
	a.SetOnUpdate(func([]float32) { updates++ })

	const seconds = 2
	const chunk = 512
	total := 48000 * seconds
	for i := 0; i < total/chunk; i++ {
		buf := make([][2]float64, chunk)
		for j := range buf {
			t := float64(i*chunk + j) / 48000
			v := math.Sin(2 * math.Pi * 440 * t)
			buf[j] = [2]float64{v, v}
		}
		a.Feed(buf)
	}
	time.Sleep(time.Duration(seconds) * time.Second)

	want := cfg.FrameRate * seconds - 4
	if updates < want {
		t.Fatalf("analyzer updates=%d want >= %d (~%d Hz)", updates, want, cfg.FrameRate)
	}
	t.Logf("analyzer updates=%d over ~%ds => ~%.1f Hz", updates, seconds, float64(updates)/float64(seconds))
}
