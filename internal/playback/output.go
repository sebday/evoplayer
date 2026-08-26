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

	submittedSamples atomic.Int64
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
	stream           Streamer
	mu               *sync.Mutex
	done             bool
	submittedSamples *atomic.Int64
}

func (r *pcmReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	if len(p) < 8 {
		return 0, nil
	}
	maxSamples := len(p) / 8
	if maxSamples > maxPCMReadSamples {
		maxSamples = maxPCMReadSamples
	}
	samples := make([][2]float64, maxSamples)
	if r.mu != nil {
		r.mu.Lock()
	}
	n, ok := r.stream.Stream(samples)
	err := r.stream.Err()
	if r.mu != nil {
		r.mu.Unlock()
	}
	if err != nil {
		return 0, err
	}
	if n == 0 && !ok {
		r.done = true
		return 0, io.EOF
	}
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint32(p[i*8:], math.Float32bits(float32(samples[i][0])))
		binary.LittleEndian.PutUint32(p[i*8+4:], math.Float32bits(float32(samples[i][1])))
	}
	if r.submittedSamples != nil {
		r.submittedSamples.Add(int64(n))
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
	o.submittedSamples.Store(0)
	o.playWG.Wait()
	if player != nil {
		player.Close()
	}
}

// SubmittedSamples returns stereo frames handed to the audio backend.
func (o *PlayerOutput) SubmittedSamples() int64 {
	return o.submittedSamples.Load()
}

// ReanchorPresentation resets the sample counter so PresentedSeconds matches a seek target.
func (o *PlayerOutput) ReanchorPresentation(seconds float64, sampleRate SampleRate) {
	if sampleRate <= 0 {
		sampleRate = outputSampleRate
	}
	delay := int64(o.PresentationDelaySamples())
	o.submittedSamples.Store(int64(seconds*float64(sampleRate)) + delay)
}

// PresentedSeconds estimates audible playback time from submitted samples minus device delay.
func (o *PlayerOutput) PresentedSeconds(sampleRate SampleRate) (float64, bool) {
	o.mu.Lock()
	player := o.player
	o.mu.Unlock()
	if player == nil {
		return 0, false
	}
	submitted := o.submittedSamples.Load()
	if submitted <= 0 {
		return 0, false
	}
	delay := int64(o.PresentationDelaySamples())
	presented := submitted - delay
	if presented < 0 {
		presented = 0
	}
	if sampleRate <= 0 {
		sampleRate = outputSampleRate
	}
	return float64(presented) / float64(sampleRate), true
}

func (o *PlayerOutput) Close() {
	o.Clear()
	o.mu.Lock()
	o.ctx = nil
	o.mu.Unlock()
}

func (o *PlayerOutput) Play(stream Streamer, streamMu *sync.Mutex, onEnd func()) error {
	if err := o.Init(); err != nil {
		return err
	}
	o.Clear()

	stop := make(chan struct{})
	reader := &pcmReader{stream: stream, mu: streamMu, submittedSamples: &o.submittedSamples}
	o.submittedSamples.Store(0)
	player := o.ctx.NewPlayer(reader)

	o.mu.Lock()
	o.stopCh = stop
	o.player = player
	o.mu.Unlock()

	o.playWG.Add(1)
	go func() {
		defer o.playWG.Done()
		defer player.Close()
		defer func() {
			if onEnd != nil {
				onEnd()
			}
		}()

		player.Play()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if reader.done {
				return
			}
			if !player.IsPlaying() {
				if err := player.Err(); err != nil {
					return
				}
			}
			if err := player.Err(); err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return nil
}
