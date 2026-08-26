package playback

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
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
	writeWAVHeader(f, sampleRate, 2, 16, len(data))
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeWAVHeader(w *os.File, sampleRate, channels, bits, dataLen int) {
	byteRate := sampleRate * channels * bits / 8
	blockAlign := channels * bits / 8
	chunkSize := 36 + dataLen
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(chunkSize))
	copy(header[8:], "WAVE")
	copy(header[12:], "fmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:], uint16(bits))
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(dataLen))
	_, _ = w.Write(header)
}

func TestDecoderPositionAdvances(t *testing.T) {
	path := writeTestWAV(t, 44100, 2)
	stream, _, err := OpenDecoder(path)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	samples := make([][2]float64, 512)
	for i := 0; i < 200; i++ {
		if _, ok := stream.Stream(samples); !ok {
			break
		}
	}
	seeker, ok := stream.(StreamSeeker)
	if !ok {
		t.Fatal("stream is not StreamSeeker")
	}
	if seeker.Position() <= 0 {
		t.Fatalf("expected position > 0, got %d", seeker.Position())
	}
}

func TestPositionLoopUpdates(t *testing.T) {
	a := NewActor(nil)
	a.mu.Lock()
	a.path = "/test.mp3"
	a.durationSec = 100
	a.playbackAnchorSec = 0
	a.playbackAnchorTime = time.Now().Add(-2 * time.Second)
	a.paused = false
	a.startPositionLoopLocked()
	a.mu.Unlock()

	time.Sleep(400 * time.Millisecond)

	a.mu.Lock()
	pos := a.positionSec
	a.stopPositionLoop()
	a.mu.Unlock()

	if pos < 1.5 {
		t.Fatalf("expected position >= 1.5, got %v", pos)
	}
}

func TestPlaybackAnchorPosition(t *testing.T) {
	a := &Actor{
		durationSec:        120,
		playbackAnchorSec:  30,
		playbackAnchorTime: time.Now().Add(-5 * time.Second),
	}
	pos := a.playbackAnchorSec + time.Since(a.playbackAnchorTime).Seconds()
	if pos < 34 || pos > 36 {
		t.Fatalf("expected ~35s, got %v", pos)
	}
}
