//go:build integration

package integration_test

import (
	"os/exec"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/playback"
	"github.com/sebday/evoplayer/internal/testharness"
)

func TestVizBroadcastRate(t *testing.T) {
	h := testharness.New(t)
	track := writePlayableTrack(t, h, "bench/sine.wav")
	// Long enough track for a 3s sample window while still playing.
	if _, err := exec.Command("ffmpeg", "-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=15",
		"-ar", "44100", "-ac", "2", track).CombinedOutput(); err != nil {
		t.Fatalf("extend track: %v", err)
	}

	h.StartDaemon()
	defer h.Stop()

	sub := h.StartSubscriber()
	defer sub.Close()

	if resp := h.IPC("viz.subscribe", nil); !resp.OK {
		t.Fatalf("viz.subscribe: %s", resp.Error)
	}
	if resp := h.IPC("queue.replace", map[string]any{
		"paths":      []string{track},
		"start_path": track,
	}); !resp.OK {
		t.Fatalf("queue.replace: %s", resp.Error)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var st playback.Status
		h.IPCData("state.get", nil, &st)
		if st.State == "playing" && st.Path == track {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	var st playback.Status
	h.IPCData("state.get", nil, &st)
	if st.State != "playing" {
		t.Fatalf("expected playing state, got %q path=%q", st.State, st.Path)
	}

	const sample = 3 * time.Second
	vizCount := sub.CountEvents("viz", sample)
	rate := float64(vizCount) / (float64(sample) / float64(time.Second))
	t.Logf("viz events=%d over %s => %.1f Hz (state=%s)", vizCount, sample, rate, st.State)

	if vizCount < 20 {
		t.Fatalf("too few viz events: %d (%.1f Hz)", vizCount, rate)
	}
	if rate < 15 {
		t.Fatalf("viz rate too low: %.1f Hz want >= 15", rate)
	}
}
