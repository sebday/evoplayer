package viz

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
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

func WriteFrame(path string, levels []float32) error {
	if path == "" || len(levels) == 0 {
		return nil
	}
	n := len(levels)
	if n > frameMaxN {
		n = frameMaxN
	}
	buf := make([]byte, 16+n*4)
	binary.LittleEndian.PutUint32(buf[0:4], frameMagic)
	seq := uint32(len(levels)) ^ uint32(n*2654435761)
	if n > 0 {
		seq ^= math.Float32bits(levels[0])
	}
	binary.LittleEndian.PutUint32(buf[4:8], seq)
	binary.LittleEndian.PutUint16(buf[8:10], uint16(n))
	off := 12
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint32(buf[off:off+4], math.Float32bits(levels[i]))
		off += 4
	}
	binary.LittleEndian.PutUint32(buf[off:off+4], seq)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func ReadFrame(path string) ([]float64, error) {
	if path == "" {
		return nil, errFrame
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
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
	out := make([]float64, n)
	off := 12
	for i := 0; i < n; i++ {
		out[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(b[off : off+4])))
		off += 4
	}
	return out, nil
}
