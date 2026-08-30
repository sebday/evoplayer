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
)

const (
	sampleRate = 8000
	targetBins = 256
	Version    = 3
)

type Payload struct {
	Version         int   `json:"version"`
	Channels        int   `json:"channels"`
	SampleRate      int   `json:"sample_rate"`
	SamplesPerPixel int   `json:"samples_per_pixel"`
	Bits            int   `json:"bits"`
	Length          int   `json:"length"`
	Data            []int `json:"data"`
}

func Build(path, outPath string) error {
	group := sampleRate * 240 / targetBins
	if group < 1 {
		group = 1
	}

	audio := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-i", path,
		"-ac", "1", "-ar", strconv.Itoa(sampleRate), "-f", "s16le", "-")
	stdout, err := audio.StdoutPipe()
	if err != nil {
		return err
	}
	if err := audio.Start(); err != nil {
		return err
	}

	data := make([]int, 0, targetBins)
	buf := make([]byte, 32*1024)
	samplesInGroup := 0
	groupPeak := 0
	groupSumSq := 0.0
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
				groupSumSq += float64(abs) * float64(abs)
				samplesInGroup++
				totalSamples++
				if samplesInGroup >= group {
					data = append(data, mixPeakRMS(groupPeak, groupSumSq, samplesInGroup))
					samplesInGroup = 0
					groupPeak = 0
					groupSumSq = 0
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
		data = append(data, mixPeakRMS(groupPeak, groupSumSq, samplesInGroup))
	}
	if len(data) == 0 {
		return errors.New("empty waveform data")
	}
	data = downsamplePeaks(data, targetBins)
	if totalSamples > 0 {
		group = totalSamples / len(data)
		if group < 1 {
			group = 1
		}
	}
	payload := Payload{
		Version:         Version,
		Channels:        1,
		SampleRate:      sampleRate,
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

func mixPeakRMS(peak int, sumSq float64, n int) int {
	if n < 1 {
		return peakToByte(peak)
	}
	rms := int(math.Sqrt(sumSq / float64(n)))
	mixed := int(float64(peak)*0.48 + float64(rms)*0.52)
	return peakToByte(mixed)
}

func peakToByte(peak int) int {
	val := int(float64(peak) / 32768.0 * 255.0)
	if val > 255 {
		return 255
	}
	if val < 0 {
		return 0
	}
	return val
}

func downsamplePeaks(data []int, n int) []int {
	if n < 1 || len(data) <= n {
		return data
	}
	out := make([]int, n)
	for i := 0; i < n; i++ {
		start := i * len(data) / n
		end := (i + 1) * len(data) / n
		if end <= start {
			end = start + 1
		}
		if end > len(data) {
			end = len(data)
		}
		mx, sum := 0, 0
		for _, v := range data[start:end] {
			if v > mx {
				mx = v
			}
			sum += v
		}
		cnt := end - start
		out[i] = int(float64(mx)*0.48 + float64(sum)/float64(cnt)*0.52)
	}
	return out
}

func CacheFresh(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var p Payload
	if json.Unmarshal(raw, &p) != nil {
		return false
	}
	return p.Version >= Version && len(p.Data) > 0
}
