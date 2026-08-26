package playback

import (
	"os/exec"
	"testing"
)

func TestFFmpegDecoderStreams(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	path := writeTestWAV(t, 48000, 2)
	stream, format, err := openFFmpegDecoder(path)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if format.SampleRate != ffmpegSampleRate {
		t.Fatalf("sample rate = %v", format.SampleRate)
	}
	samples := make([][2]float64, 512)
	n, ok := stream.Stream(samples)
	if n == 0 || !ok {
		t.Fatalf("expected samples, n=%d ok=%v", n, ok)
	}
	if seeker, ok := stream.(interface{ Seek(int) error }); ok {
		if err := seeker.Seek(48000); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDurationForPath(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe required")
	}
	path := writeTestWAV(t, 48000, 1)
	dur := DurationForPath(path)
	if dur < 0.9 {
		t.Fatalf("duration = %v, want ~1s", dur)
	}
}
