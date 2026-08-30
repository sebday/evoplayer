package playback

import (
	"errors"
	"fmt"
	"sync"
)

const (
	boundedBufferSamples = 48000 * 2 // ~2s stereo at 48kHz
	pumpChunkSamples     = 512
)

// BoundStream wraps a decoder with a fixed-capacity PCM ring and a background
// pump. Decode runs only as fast as playback consumes, capped at boundedBufferSamples.
func BoundStream(inner StreamSeekCloser) StreamSeekCloser {
	if inner == nil {
		return nil
	}
	b := &boundedStream{
		inner: inner,
		ring:  newPCMRing(boundedBufferSamples),
	}
	b.startPump()
	return b
}

type boundedStream struct {
	inner StreamSeekCloser
	ring  *pcmRing

	mu       sync.Mutex
	closed   bool
	stopPump chan struct{}
	pumpWG   sync.WaitGroup
}

func (b *boundedStream) Stream(samples [][2]float64) (int, bool) {
	n, more, err := b.ring.pop(samples)
	if err != nil {
		return n, n > 0
	}
	return n, n > 0 || more
}

func (b *boundedStream) Err() error {
	if err := b.ring.err(); err != nil {
		return err
	}
	return b.inner.Err()
}

func (b *boundedStream) Len() int {
	if seeker, ok := b.inner.(StreamSeeker); ok {
		return seeker.Len()
	}
	return 0
}

func (b *boundedStream) Position() int {
	if seeker, ok := b.inner.(StreamSeeker); ok {
		return seeker.Position()
	}
	return 0
}

func (b *boundedStream) Seek(p int) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return fmt.Errorf("bounded: closed")
	}
	b.mu.Unlock()

	b.haltPump()
	if seeker, ok := b.inner.(StreamSeeker); ok {
		if err := seeker.Seek(p); err != nil {
			return err
		}
	}
	b.startPump()
	return nil
}

func (b *boundedStream) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	b.haltPump()
	b.ring.close()
	return b.inner.Close()
}

func (b *boundedStream) startPump() {
	stop := make(chan struct{})
	b.stopPump = stop
	b.pumpWG.Add(1)
	go func() {
		defer b.pumpWG.Done()
		scratch := make([][2]float64, pumpChunkSamples)
		for {
			select {
			case <-stop:
				return
			default:
			}
			n, ok := b.inner.Stream(scratch)
			if n > 0 {
				if err := b.ring.push(scratch[:n]); err != nil {
					return
				}
			}
			if err := b.inner.Err(); err != nil {
				b.ring.setErr(err)
				return
			}
			if !ok {
				b.ring.setEOF()
				return
			}
		}
	}()
}

func (b *boundedStream) haltPump() {
	b.ring.close()
	if b.stopPump != nil {
		close(b.stopPump)
		b.pumpWG.Wait()
		b.stopPump = nil
	}
	b.ring.reset()
}

type pcmRing struct {
	buf     [][2]float64
	start   int
	count   int
	mu      sync.Mutex
	cond    sync.Cond
	eof     bool
	readErr error
	closed  bool
}

func newPCMRing(capacity int) *pcmRing {
	r := &pcmRing{buf: make([][2]float64, capacity)}
	r.cond.L = &r.mu
	return r
}

func (r *pcmRing) push(samples [][2]float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sample := range samples {
		for r.count == len(r.buf) && !r.closed && !r.eof && r.readErr == nil {
			r.cond.Wait()
		}
		if r.closed {
			return errors.New("pcm ring: closed")
		}
		if r.readErr != nil {
			return r.readErr
		}
		if r.eof {
			return nil
		}
		idx := (r.start + r.count) % len(r.buf)
		r.buf[idx] = sample
		r.count++
		r.cond.Signal()
	}
	return nil
}

func (r *pcmRing) pop(dst [][2]float64) (int, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for r.count == 0 && !r.eof && !r.closed && r.readErr == nil {
		r.cond.Wait()
	}
	if r.readErr != nil {
		return 0, false, r.readErr
	}
	n := 0
	for n < len(dst) && r.count > 0 {
		dst[n] = r.buf[r.start]
		r.start = (r.start + 1) % len(r.buf)
		r.count--
		n++
	}
	if r.count < len(r.buf) {
		r.cond.Signal()
	}
	more := r.count > 0 || (!r.eof && !r.closed)
	return n, more, nil
}

func (r *pcmRing) setEOF() {
	r.mu.Lock()
	r.eof = true
	r.cond.Broadcast()
	r.mu.Unlock()
}

func (r *pcmRing) setErr(err error) {
	r.mu.Lock()
	r.readErr = err
	r.cond.Broadcast()
	r.mu.Unlock()
}

func (r *pcmRing) err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readErr
}

func (r *pcmRing) close() {
	r.mu.Lock()
	r.closed = true
	r.cond.Broadcast()
	r.mu.Unlock()
}

func (r *pcmRing) reset() {
	r.mu.Lock()
	r.start = 0
	r.count = 0
	r.eof = false
	r.readErr = nil
	r.closed = false
	r.cond.Broadcast()
	r.mu.Unlock()
}
