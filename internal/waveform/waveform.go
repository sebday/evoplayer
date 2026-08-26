package waveform

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Payload struct {
	Version           int   `json:"version"`
	Channels          int   `json:"channels"`
	SampleRate        int   `json:"sample_rate"`
	SamplesPerPixel   int   `json:"samples_per_pixel"`
	Bits              int   `json:"bits"`
	Length            int   `json:"length"`
	Data              []int `json:"data"`
}

func Build(path, outPath string) error {
	dur := 0.0
	cmd := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", path)
	if raw, err := cmd.Output(); err == nil {
		dur, _ = strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	}
	target := 900
	if dur > 0 {
		target = int(math.Max(180, math.Min(2400, dur*12)))
	}
	estimatedSamples := int(dur * 11025)
	if estimatedSamples <= 0 {
		estimatedSamples = target * 100
	}
	group := estimatedSamples / target
	if group < 1 {
		group = 1
	}

	audio := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-i", path,
		"-ac", "1", "-ar", "11025", "-f", "s16le", "-")
	stdout, err := audio.StdoutPipe()
	if err != nil {
		return err
	}
	if err := audio.Start(); err != nil {
		return err
	}

	data := make([]int, 0, target)
	buf := make([]byte, 32*1024)
	samplesInGroup := 0
	groupPeak := 0
	totalSamples := 0

	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if len(chunk)%2 != 0 {
				chunk = chunk[:len(chunk)-1]
			}
			for i := 0; i+1 < len(chunk); i += 2 {
				sample := int16(binary.LittleEndian.Uint16(chunk[i : i+2]))
				abs := int(sample)
				if abs < 0 {
					abs = -abs
				}
				if abs > groupPeak {
					groupPeak = abs
				}
				samplesInGroup++
				totalSamples++
				if samplesInGroup >= group {
					val := int(float64(groupPeak) / 32768.0 * 255.0)
					if val > 255 {
						val = 255
					}
					data = append(data, val)
					samplesInGroup = 0
					groupPeak = 0
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = audio.Process.Kill()
			_ = audio.Wait()
			return readErr
		}
	}
	if err := audio.Wait(); err != nil {
		return err
	}
	if samplesInGroup > 0 || groupPeak > 0 {
		val := int(float64(groupPeak) / 32768.0 * 255.0)
		if val > 255 {
			val = 255
		}
		data = append(data, val)
	}
	if len(data) == 0 {
		return errors.New("empty waveform data")
	}
	if totalSamples > 0 {
		group = totalSamples / len(data)
		if group < 1 {
			group = 1
		}
	}
	payload := Payload{
		Version:         2,
		Channels:        1,
		SampleRate:      11025,
		SamplesPerPixel: group,
		Bits:            8,
		Length:          len(data),
		Data:            data,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}
