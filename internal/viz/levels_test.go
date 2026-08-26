package viz

import (
	"math"
	"testing"
)

func TestAnalyzerLevelsVisible(t *testing.T) {
	a := NewAnalyzer(48000)
	a.SetWanted(true)
	chunk := make([][2]float64, FFTSize*8)
	for i := range chunk {
		t := float64(i) / 48000
		v := math.Sin(2*math.Pi*440*t) * 0.5
		chunk[i] = [2]float64{v, v}
	}
	a.Feed(chunk)
	s := a.Snapshot()
	max := float32(0)
	visible := 0
	for _, v := range s {
		if v > 0.015 {
			visible++
		}
		if v > max {
			max = v
		}
	}
	if max < 0.05 {
		t.Fatalf("max level %v too low for viz paint threshold", max)
	}
	if visible < 5 {
		t.Fatalf("only %d bars above paint threshold, max=%v", visible, max)
	}
}
