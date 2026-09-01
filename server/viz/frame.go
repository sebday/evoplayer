package viz

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"sync"
)

const (
	frameMagic = 0x45565053 // EVPS
	frameMaxN  = 512
)

var errFrame = errors.New("invalid spectrum frame")

func FramePath(socketPath string) string {
	if socketPath == "" {
		return ""
	}
	return socketPath + ".spectrum"
}

func encodeFrame(buf []byte, levels []float32) []byte {
	n := len(levels)
	if n > frameMaxN {
		n = frameMaxN
	}
	need := 16 + n*4
	if cap(buf) < need {
		buf = make([]byte, need)
	}
	buf = buf[:need]
	binary.LittleEndian.PutUint32(buf[0:4], frameMagic)
	seq := uint32(len(levels)) ^ uint32(n*2654435761)
	if n > 0 {
		seq ^= math.Float32bits(levels[0])
	}
	binary.LittleEndian.PutUint32(buf[4:8], seq)
	binary.LittleEndian.PutUint16(buf[8:10], uint16(n))
	binary.LittleEndian.PutUint16(buf[10:12], 0)
	off := 12
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint32(buf[off:off+4], math.Float32bits(levels[i]))
		off += 4
	}
	binary.LittleEndian.PutUint32(buf[off:off+4], seq)
	return buf
}

func decodeFrame(b []byte, dst []float64) ([]float64, error) {
	if len(b) < 16 {
		return nil, errFrame
	}
	if binary.LittleEndian.Uint32(b[0:4]) != frameMagic {
		return nil, errFrame
	}
	seq := binary.LittleEndian.Uint32(b[4:8])
	n := int(binary.LittleEndian.Uint16(b[8:10]))
	if n < 1 || n > frameMaxN || len(b) < 16+n*4 {
		return nil, errFrame
	}
	end := 12 + n*4
	if binary.LittleEndian.Uint32(b[end:end+4]) != seq {
		return nil, errFrame
	}
	if cap(dst) < n {
		dst = make([]float64, n)
	}
	dst = dst[:n]
	off := 12
	for i := 0; i < n; i++ {
		dst[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off : off+4])))
		off += 4
	}
	return dst, nil
}

func WriteFrame(path string, levels []float32) error {
	w := NewFrameWriter(path)
	defer w.Close()
	return w.Write(levels)
}

// FrameWriter keeps a reusable buffer and an open file so the analyzer can
// publish frames without a temp-file rename on every tick.
type FrameWriter struct {
	mu   sync.Mutex
	path string
	f    *os.File
	buf  []byte
}

func NewFrameWriter(path string) *FrameWriter {
	return &FrameWriter{path: path}
}

func (w *FrameWriter) Write(levels []float32) error {
	if w == nil || w.path == "" || len(levels) == 0 {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = encodeFrame(w.buf, levels)
	if w.f == nil {
		f, err := os.OpenFile(w.path, os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			return err
		}
		w.f = f
	}
	if err := w.f.Truncate(int64(len(w.buf))); err != nil {
		return err
	}
	_, err := w.f.WriteAt(w.buf, 0)
	return err
}

func (w *FrameWriter) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}

// FrameReader reuses scratch buffers so the TUI painter can poll without
// allocating a new file copy every tick.
type FrameReader struct {
	scratch []byte
	levels  []float64
}

func (r *FrameReader) Read(path string) ([]float64, error) {
	if path == "" {
		return nil, errFrame
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	n := int(st.Size())
	if n < 16 {
		return nil, errFrame
	}
	if cap(r.scratch) < n {
		r.scratch = make([]byte, n)
	}
	r.scratch = r.scratch[:n]
	if _, err := io.ReadFull(f, r.scratch); err != nil {
		return nil, err
	}
	levels, err := decodeFrame(r.scratch, r.levels)
	if err != nil {
		return nil, err
	}
	r.levels = levels
	return levels, nil
}
