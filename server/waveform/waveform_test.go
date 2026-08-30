package waveform

import (
	"encoding/binary"
	"encoding/json"
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

func TestBuildWritesNonEmptyData(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	src := writeTestWAV(t, 44100, 2)
	out := filepath.Join(t.TempDir(), "wave.json")
	if err := Build(src, out); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var payload Payload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Channels != 1 {
		t.Fatalf("channels = %d", payload.Channels)
	}
	if len(payload.Data) == 0 {
		t.Fatal("empty data")
	}
	if payload.Length != len(payload.Data) {
		t.Fatalf("length = %d data = %d", payload.Length, len(payload.Data))
	}
	peak := 0
	for _, v := range payload.Data {
		if v < 0 || v > 255 {
			t.Fatalf("peak out of range: %d", v)
		}
		if v > peak {
			peak = v
		}
	}
	if peak == 0 {
		t.Fatal("all peaks were zero")
	}
	if payload.Version != Version {
		t.Fatalf("version = %d want %d", payload.Version, Version)
	}
}

func TestCacheFreshRequiresCurrentVersion(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "old.json")
	raw, _ := json.Marshal(Payload{Version: 2, Data: []int{1, 2, 3}})
	if err := os.WriteFile(stale, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if CacheFresh(stale) {
		t.Fatal("v2 cache should be stale")
	}
	fresh := filepath.Join(dir, "new.json")
	raw, _ = json.Marshal(Payload{Version: Version, Data: []int{1, 2, 3}})
	if err := os.WriteFile(fresh, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if !CacheFresh(fresh) {
		t.Fatal("current version should be fresh")
	}
}

func TestDownsampleMixesPeakAndAvg(t *testing.T) {
	got := downsamplePeaks([]int{10, 10, 250, 10}, 1)
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0] >= 250 {
		t.Fatalf("peak-only downsample, got %d", got[0])
	}
	if got[0] <= 10 {
		t.Fatalf("ignored peak, got %d", got[0])
	}
}
