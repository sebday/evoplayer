package playback

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/viz"
)

func TestTapVizUpdateRateWhilePlaying(t *testing.T) {
	path := writeTestWAV(t, 48000, 15)
	stream, _, err := OpenDecoder(path)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	a := viz.NewAnalyzer(48000)
	a.SetWanted(true)
	var updates atomic.Int32
	a.SetOnUpdate(func([]float32) { updates.Add(1) })

	var paused bool
	tapped := viz.Tap(pausedStreamer(stream, &paused), a)

	done := make(chan struct{})
	go func() {
		defer close(done)
		samples := make([][2]float64, 512)
		for {
			n, ok := tapped.Stream(samples)
			if n == 0 && !ok {
				return
			}
		}
	}()

	const sample = 3 * time.Second
	time.Sleep(sample)
	got := int(updates.Load())
	rate := float64(got) / (float64(sample) / float64(time.Second))
	t.Logf("tap viz updates=%d over %s => %.1f Hz", got, sample, rate)

	if got < 20 {
		t.Fatalf("too few viz updates: %d (%.1f Hz)", got, rate)
	}
	if rate < 15 {
		t.Fatalf("viz update rate too low: %.1f Hz want >= 15", rate)
	}
}
