package playback

// SampleRate is samples per second.
type SampleRate int

// Format describes decoded PCM layout.
type Format struct {
	SampleRate  SampleRate
	NumChannels int
	Precision   int
}

// Streamer pulls stereo float64 PCM samples.
type Streamer interface {
	Stream(samples [][2]float64) (n int, ok bool)
	Err() error
}

// StreamSeekCloser is a readable PCM source that must be closed.
type StreamSeekCloser interface {
	Streamer
	Close() error
}

// StreamSeeker supports sample-index seeking on the underlying decoder.
type StreamSeeker interface {
	StreamSeekCloser
	Len() int
	Position() int
	Seek(p int) error
}
