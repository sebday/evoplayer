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
	durationSamples int
	positionSamples int
	err             error
	closed          bool
}

func openFFmpegDecoder(path string) (StreamSeekCloser, Format, error) {
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
	d.positionSamples = int(seekSec * float64(ffmpegSampleRate))
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

func (d *ffmpegDecoder) Stream(samples [][2]float64) (n int, ok bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err != nil || d.closed || d.stdout == nil {
		return 0, false
	}
	var scratch [8]byte
	for i := range samples {
		_, err := io.ReadFull(d.stdout, scratch[:])
		if err == io.EOF {
			return n, n > 0
		}
		if err != nil {
			d.err = err
			return n, n > 0
		}
		l := math.Float32frombits(binary.LittleEndian.Uint32(scratch[0:4]))
		r := math.Float32frombits(binary.LittleEndian.Uint32(scratch[4:8]))
		samples[i][0] = float64(l)
		samples[i][1] = float64(r)
		d.positionSamples++
		n++
		ok = true
	}
	return n, ok
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

func (d *ffmpegDecoder) Position() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.positionSamples
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
