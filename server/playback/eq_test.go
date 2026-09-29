package playback

import (
	"math"
	"testing"
)

func TestFlatEQLeavesSamples(t *testing.T) {
	eq := NewEQ()
	in := [][2]float64{{0.2, -0.4}, {0, 1}, {-0.75, 0.5}}
	got := append([][2]float64(nil), in...)
	eq.process(got)
	for i := range in {
		if got[i] != in[i] {
			t.Fatalf("sample %d = %v, want %v", i, got[i], in[i])
		}
	}
}

func TestEQBoostsTargetFrequency(t *testing.T) {
	eq := NewEQ()
	var cfg EQConfig
	cfg.Enabled = true
	cfg.Gains[5] = 6 // 1 kHz
	eq.SetConfig(cfg)

	boost := measureDB(eq, 1000)
	if math.Abs(boost-6) > 1 {
		t.Fatalf("1 kHz boost = %.2f dB, want about 6", boost)
	}
	side := measureDB(eq, 100)
	if math.Abs(side) > 1 {
		t.Fatalf("100 Hz change = %.2f dB, want about 0", side)
	}
}

func TestEQClampsGain(t *testing.T) {
	eq := NewEQ()
	var cfg EQConfig
	cfg.Enabled = true
	cfg.Preamp = 40
	cfg.Gains[0] = -80
	eq.SetConfig(cfg)
	got := eq.Config()
	if got.Preamp != 12 || got.Gains[0] != -12 {
		t.Fatalf("clamped preamp=%v band=%v", got.Preamp, got.Gains[0])
	}
}

func measureDB(eq *EQ, freq float64) float64 {
	const rate = int(outputSampleRate)
	const amp = 0.1
	n := rate / 2
	samples := make([][2]float64, n)
	for i := range samples {
		s := amp * math.Sin(2*math.Pi*freq*float64(i)/float64(rate))
		samples[i] = [2]float64{s, s}
	}
	eq.Reset()
	eq.process(samples)
	skip := rate / 10
	var inSum, outSum float64
	for i := skip; i < n; i++ {
		s := amp * math.Sin(2*math.Pi*freq*float64(i)/float64(rate))
		inSum += s * s
		outSum += samples[i][0] * samples[i][0]
	}
	return 20 * math.Log10(math.Sqrt(outSum/inSum))
}
