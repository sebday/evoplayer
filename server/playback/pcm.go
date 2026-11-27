package playback

// SampleRate is samples per second.
type SampleRate int

// Format describes decoded PCM layout.
type Format struct {
	SampleRate SampleRate
}

// Streamer pulls stereo float64 PCM samples.
type Streamer interface {
	Stream(samples [][2]float64) (n int, ok bool)
	Err() error
}

// StreamSeeker supports sample-index seeking on the underlying decoder.
type StreamSeeker interface {
	Streamer
	Close() error
	Len() int
	Seek(p int) error
}
