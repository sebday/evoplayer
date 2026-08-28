package playback

import (
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeTestWAV(t *testing.T, sampleRate int, seconds int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wav")
	numSamples := sampleRate * seconds
	data := make([]byte, numSamples*4)
	for i := 0; i < numSamples; i++ {
		v := int16(math.Sin(float64(i)/float64(sampleRate)*2*math.Pi*440) * 16000)
		binary.LittleEndian.PutUint16(data[i*4:], uint16(v))
		binary.LittleEndian.PutUint16(data[i*4+2:], uint16(v))
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	byteRate := sampleRate * 2 * 16 / 8
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+len(data)))
	copy(header[8:], "WAVE")
	copy(header[12:], "fmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 2)
	binary.LittleEndian.PutUint32(header[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:], 4)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(data)))
	if _, err := f.Write(header); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	return path
}

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
