package playback

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"sync"
)

const ffmpegSampleRate = SampleRate(48000)

type ffmpegDecoder struct {
	path            string
	mu              sync.Mutex
	cmd             *exec.Cmd
	stdout          io.ReadCloser
	buf             []byte
	durationSamples int
	err             error
	closed          bool
}

func openFFmpegDecoder(path string) (StreamSeeker, Format, error) {
	if !ffmpegAvailable() {
		return nil, Format{}, fmt.Errorf("ffmpeg not available")
	}
	dur := DurationForPath(path)
	d := &ffmpegDecoder{
		path:            path,
		durationSamples: int(dur * float64(ffmpegSampleRate)),
	}
	if err := d.start(0); err != nil {
		return nil, Format{}, err
	}
	format := Format{
		SampleRate:  ffmpegSampleRate,
		NumChannels: 2,
		Precision:   4,
	}
	return d, format, nil
}

func (d *ffmpegDecoder) start(seekSec float64) error {
	d.killProcess()
	args := []string{
		"-nostdin", "-hide_banner", "-loglevel", "error",
	}
	if seekSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", seekSec))
	}
	args = append(args,
		"-i", d.path,
		"-f", "f32le", "-ac", "2", "-ar", fmt.Sprintf("%d", int(ffmpegSampleRate)),
		"pipe:1",
	)
	cmd := exec.Command("ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	d.cmd = cmd
	d.stdout = stdout
	d.err = nil
	return nil
}

func (d *ffmpegDecoder) killProcess() {
	if d.stdout != nil {
		_ = d.stdout.Close()
		d.stdout = nil
	}
	if d.cmd != nil && d.cmd.Process != nil {
		_ = d.cmd.Process.Kill()
		_ = d.cmd.Wait()
	}
	d.cmd = nil
}

func (d *ffmpegDecoder) Stream(samples [][2]float64) (int, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil || d.closed || d.stdout == nil {
		return 0, false
	}
	need := len(samples) * stereoF32Bytes
	if cap(d.buf) < need {
		d.buf = make([]byte, need)
	}
	buf := d.buf[:need]
	read, err := io.ReadFull(d.stdout, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		d.err = err
	}
	n := read / stereoF32Bytes
	for i := 0; i < n; i++ {
		frame := buf[i*stereoF32Bytes:]
		samples[i][0] = float64(math.Float32frombits(binary.LittleEndian.Uint32(frame[0:4])))
		samples[i][1] = float64(math.Float32frombits(binary.LittleEndian.Uint32(frame[4:8])))
	}
	return n, n > 0
}

func (d *ffmpegDecoder) Err() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.err
}

func (d *ffmpegDecoder) Len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.durationSamples
}

func (d *ffmpegDecoder) Seek(p int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return fmt.Errorf("ffmpeg: closed")
	}
	if p < 0 {
		p = 0
	}
	if d.durationSamples > 0 && p > d.durationSamples {
		p = d.durationSamples
	}
	seekSec := float64(p) / float64(ffmpegSampleRate)
	return d.start(seekSec)
}

func (d *ffmpegDecoder) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	d.killProcess()
	return nil
}
