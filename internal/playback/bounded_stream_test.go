package playback

import (
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/viz"
)

func TestBoundedStreamPassesSamples(t *testing.T) {
	path := writeTestWAV(t, 48000, 2)
	stream, _, err := OpenDecoder(path)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	samples := make([][2]float64, 512)
	total := 0
	for i := 0; i < 200; i++ {
		n, ok := stream.Stream(samples)
		total += n
		if !ok && n == 0 {
			break
		}
	}
	if total == 0 {
		t.Fatal("expected streamed samples")
	}
}

func TestBoundedStreamSeek(t *testing.T) {
	path := writeTestWAV(t, 48000, 3)
	stream, _, err := OpenDecoder(path)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	seeker := stream.(interface {
		Seek(int) error
		Position() int
	})
	if err := seeker.Seek(48000); err != nil {
		t.Fatal(err)
	}
	if seeker.Position() < 48000-4800 || seeker.Position() > 48000+4800 {
		t.Fatalf("position after seek = %d, want ~48000", seeker.Position())
	}
}

func TestBoundedStreamFeedsViz(t *testing.T) {
	path := writeTestWAV(t, 48000, 2)
	stream, _, err := OpenDecoder(path)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	a := viz.NewAnalyzer(48000)
	a.SetWanted(true)
	updates := 0
	a.SetOnUpdate(func([]float32) { updates++ })
	var paused bool
	tapped := viz.Tap(pausedStreamer(stream, &paused), a)

	samples := make([][2]float64, 512)
	total := 0
	for i := 0; i < 2000; i++ {
		n, ok := tapped.Stream(samples)
		total += n
		if !ok && n == 0 {
			break
		}
	}
	if total == 0 {
		t.Fatal("expected streamed samples")
	}
	time.Sleep(500 * time.Millisecond)
	if updates == 0 {
		t.Fatalf("expected viz updates, streamed %d samples", total)
	}
}

func TestPCMRingPopCopiesSamples(t *testing.T) {
	r := newPCMRing(8)
	if err := r.push([][2]float64{{0.25, 0.5}, {0.75, 1.0}}); err != nil {
		t.Fatal(err)
	}
	out := make([][2]float64, 2)
	n, more, err := r.pop(out)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || !more {
		t.Fatalf("pop n=%d more=%v", n, more)
	}
	if out[0] != [2]float64{0.25, 0.5} || out[1] != [2]float64{0.75, 1.0} {
		t.Fatalf("unexpected samples: %#v", out)
	}
}

func TestPCMRingCapacityBound(t *testing.T) {
	capacity := 256
	r := newPCMRing(capacity)
	chunk := make([][2]float64, 128)
	for i := range chunk {
		chunk[i] = [2]float64{1, 1}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			if err := r.push(chunk); err != nil {
				return
			}
		}
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		count := r.count
		r.mu.Unlock()
		if count >= capacity {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	r.mu.Lock()
	count := r.count
	r.mu.Unlock()
	if count > capacity {
		t.Fatalf("ring count = %d, want <= %d", count, capacity)
	}

	r.close()
	<-done
}

func TestBoundedStreamRingStaysBounded(t *testing.T) {
	path := writeTestWAV(t, 48000, 10)
	inner, format, err := openFFmpegDecoder(path)
	if err != nil {
		t.Fatal(err)
	}
	b := BoundStream(inner).(*boundedStream)
	defer b.Close()

	time.Sleep(100 * time.Millisecond)

	b.ring.mu.Lock()
	count := b.ring.count
	b.ring.mu.Unlock()
	if count > boundedBufferSamples {
		t.Fatalf("ring count = %d, want <= %d", count, boundedBufferSamples)
	}
	if format.SampleRate == 0 {
		t.Fatal("expected format sample rate")
	}
}
