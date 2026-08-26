package playback

type pausedCtrl struct {
	inner  Streamer
	paused *bool
}

func pausedStreamer(inner Streamer, paused *bool) Streamer {
	if inner == nil || paused == nil {
		return inner
	}
	return &pausedCtrl{inner: inner, paused: paused}
}

func (c *pausedCtrl) Stream(samples [][2]float64) (int, bool) {
	if *c.paused {
		for i := range samples {
			samples[i][0] = 0
			samples[i][1] = 0
		}
		return len(samples), true
	}
	return c.inner.Stream(samples)
}

func (c *pausedCtrl) Err() error {
	return c.inner.Err()
}
