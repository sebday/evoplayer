package playback

import (
	"encoding/binary"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)

const outputSampleRate = SampleRate(48000)

// PlayerOutput plays float64 stereo PCM via oto.
type PlayerOutput struct {
	mu     sync.Mutex
	ctx    *oto.Context
	player *oto.Player
	stopCh chan struct{}
	playWG sync.WaitGroup
	volume float64
	paused atomic.Bool
}

const (
	stereoF32Bytes   = 8
	devicePadSamples = 480 // ~10ms past oto's unread software buffer
	minDelaySamples  = 960 // ~20ms floor when the player is nil or empty
)

func (o *PlayerOutput) PresentationDelaySamples() int {
	o.mu.Lock()
	player := o.player
	o.mu.Unlock()
	delay := devicePadSamples
	if player != nil {
		delay += player.BufferedSize() / stereoF32Bytes
	}
	if delay < minDelaySamples {
		delay = minDelaySamples
	}
	return delay
}

const maxPCMReadSamples = 2048 // ~43ms at 48kHz; matches beep-era pull granularity

type pcmReader struct {
	stream  Streamer
	mu      *sync.Mutex
	done    atomic.Bool
	samples [][2]float64
}

func (r *pcmReader) Read(p []byte) (int, error) {
	if r.done.Load() {
		return 0, io.EOF
	}
	if len(p) < stereoF32Bytes {
		return 0, nil
	}
	maxSamples := min(len(p)/stereoF32Bytes, maxPCMReadSamples)
	if cap(r.samples) < maxSamples {
		r.samples = make([][2]float64, maxSamples)
	}
	samples := r.samples[:maxSamples]
	r.mu.Lock()
	n, ok := r.stream.Stream(samples)
	err := r.stream.Err()
	r.mu.Unlock()
	if err != nil {
		return 0, err
	}
	if n == 0 && !ok {
		r.done.Store(true)
		return 0, io.EOF
	}
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint32(p[i*8:], math.Float32bits(float32(samples[i][0])))
		binary.LittleEndian.PutUint32(p[i*8+4:], math.Float32bits(float32(samples[i][1])))
	}
	return n * 8, nil
}

func (o *PlayerOutput) Init() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.ctx != nil {
		return nil
	}
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   int(outputSampleRate),
		ChannelCount: 2,
		Format:       oto.FormatFloat32LE,
	})
	if err != nil {
		return err
	}
	<-ready
	o.ctx = ctx
	return nil
}

func (o *PlayerOutput) Clear() {
	o.mu.Lock()
	if o.stopCh != nil {
		close(o.stopCh)
		o.stopCh = nil
	}
	player := o.player
	o.player = nil
	o.mu.Unlock()
	o.playWG.Wait()
	if player != nil {
		player.Close()
	}
}

func (o *PlayerOutput) Close() {
	o.Clear()
	o.mu.Lock()
	o.ctx = nil
	o.mu.Unlock()
}

// SetVolume sets linear gain; it applies to the current and future players.
func (o *PlayerOutput) SetVolume(volume float64) {
	o.mu.Lock()
	o.volume = volume
	player := o.player
	o.mu.Unlock()
	if player != nil {
		player.SetVolume(volume)
	}
}

// SetPaused pauses or resumes the current player in place, keeping its buffer.
func (o *PlayerOutput) SetPaused(paused bool) {
	o.mu.Lock()
	player := o.player
	o.mu.Unlock()
	if paused {
		o.paused.Store(true)
		if player != nil {
			player.Pause()
		}
		return
	}
	if player != nil {
		player.Play()
	}
	o.paused.Store(false)
}

// Play replaces the current player. Callers serialize Play and SetPaused.
func (o *PlayerOutput) Play(stream Streamer, streamMu *sync.Mutex, onEnd func()) error {
	if err := o.Init(); err != nil {
		return err
	}
	o.Clear()

	stop := make(chan struct{})
	reader := &pcmReader{stream: stream, mu: streamMu}
	player := o.ctx.NewPlayer(reader)

	o.mu.Lock()
	player.SetVolume(o.volume)
	o.stopCh = stop
	o.player = player
	o.mu.Unlock()

	if !o.paused.Load() {
		player.Play()
	}

	o.playWG.Add(1)
	go func() {
		defer o.playWG.Done()
		defer player.Close()
		defer func() {
			if onEnd != nil {
				onEnd()
			}
		}()

		for {
			select {
			case <-stop:
				return
			default:
			}
			if err := player.Err(); err != nil {
				return
			}
			if reader.done.Load() && !player.IsPlaying() && !o.paused.Load() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return nil
}
