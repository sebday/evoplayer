package playback

import "math"

type volumeCtrl struct {
	inner  Streamer
	gain   func() float64
	silent func() bool
}

func volumeStreamer(inner Streamer, gain func() float64, silent func() bool) Streamer {
	if inner == nil {
		return nil
	}
	return &volumeCtrl{inner: inner, gain: gain, silent: silent}
}

func (v *volumeCtrl) Stream(samples [][2]float64) (int, bool) {
	n, ok := v.inner.Stream(samples)
	if n <= 0 {
		return n, ok
	}
	if v.silent != nil && v.silent() {
		for i := 0; i < n; i++ {
			samples[i][0] = 0
			samples[i][1] = 0
		}
		return n, ok
	}
	g := 1.0
	if v.gain != nil {
		g = math.Pow(2, v.gain())
	}
	for i := 0; i < n; i++ {
		samples[i][0] *= g
		samples[i][1] *= g
	}
	return n, ok
}

func (v *volumeCtrl) Err() error {
	return v.inner.Err()
}
