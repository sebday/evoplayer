package viz

type pcmStreamer interface {
	Stream(samples [][2]float64) (n int, ok bool)
	Err() error
}

type tapStreamer struct {
	src pcmStreamer
	a   *Analyzer
}

func Tap(src pcmStreamer, a *Analyzer) pcmStreamer {
	if src == nil || a == nil {
		return src
	}
	return &tapStreamer{src: src, a: a}
}

func (t *tapStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = t.src.Stream(samples)
	if n > 0 {
		t.a.Feed(samples[:n])
	}
	return n, ok
}

func (t *tapStreamer) Err() error {
	return t.src.Err()
}
