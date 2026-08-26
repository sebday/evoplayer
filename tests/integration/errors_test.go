//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/playback"
	"github.com/sebday/evoplayer/internal/testharness"
)

func assertEnrichedState(t *testing.T, st playback.Status, rawTitle string) {
	t.Helper()
	if st.Path == "" {
		t.Fatal("expected path in status")
	}
	if st.Title == rawTitle {
		t.Fatalf("state title still raw filename %q; enrichStatus regression", rawTitle)
	}
	if st.Title != "midnight" {
		t.Fatalf("title=%q want midnight", st.Title)
	}
	if st.Artist != "loefah" {
		t.Fatalf("artist=%q want loefah", st.Artist)
	}
}

func writePlayableTrack(t *testing.T, h *testharness.Harness, rel string) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required for playback integration test")
	}
	p := filepath.Join(h.MusicRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-ar", "44100", "-ac", "2", p)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v: %s", err, out)
	}
	return p
}

func TestEnrichedStateGetAndBroadcast(t *testing.T) {
	h := testharness.New(t)
	track := writePlayableTrack(t, h, "grime/2008/loefah-midnight.wav")
	rawTitle := "loefah-midnight"

	h.StartDaemon()
	defer h.Stop()

	sub := h.StartSubscriber()
	defer sub.Close()

	resp := h.IPC("queue.replace", map[string]any{
		"paths":      []string{track},
		"start_path": track,
	})
	if !resp.OK {
		t.Fatalf("queue.replace: %s", resp.Error)
	}
	time.Sleep(200 * time.Millisecond)

	var st playback.Status
	h.IPCData("state.get", nil, &st)
	assertEnrichedState(t, st, rawTitle)

	ev := sub.WaitState(3*time.Second, func() {
		_ = h.IPC("playback.volume.set", map[string]any{"volume": 42})
	})
	if ev.Event != "state" {
		t.Fatal("timed out waiting for enriched state broadcast")
	}
	b, err := json.Marshal(ev.Data)
	if err != nil {
		t.Fatal(err)
	}
	var broadcast playback.Status
	if err := json.Unmarshal(b, &broadcast); err != nil {
		t.Fatal(err)
	}
	assertEnrichedState(t, broadcast, rawTitle)
}

func TestEnrichedBroadcastIncludesArt(t *testing.T) {
	h := testharness.New(t)
	track := writePlayableTrack(t, h, "grime/2008/loefah-midnight.wav")
	artDir := h.Env().ArtDir
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artFile := filepath.Join(artDir, "grime_2008.jpg")
	if err := os.WriteFile(artFile, []byte("fake-jpg"), 0o644); err != nil {
		t.Fatal(err)
	}

	h.StartDaemon()
	defer h.Stop()

	sub := h.StartSubscriber()
	defer sub.Close()

	resp := h.IPC("queue.replace", map[string]any{
		"paths":      []string{track},
		"start_path": track,
	})
	if !resp.OK {
		t.Fatalf("queue.replace: %s", resp.Error)
	}
	time.Sleep(200 * time.Millisecond)

	var st playback.Status
	h.IPCData("state.get", nil, &st)
	if st.Art == "" {
		t.Fatalf("expected art on enriched state.get, got %#v", st)
	}

	ev := sub.WaitState(3*time.Second, func() {
		_ = h.IPC("playback.volume.set", map[string]any{"volume": 55})
	})
	if ev.Event != "state" {
		t.Fatal("timed out waiting for state broadcast with art")
	}
	b, _ := json.Marshal(ev.Data)
	var broadcast playback.Status
	if err := json.Unmarshal(b, &broadcast); err != nil {
		t.Fatal(err)
	}
	if broadcast.Art == "" {
		t.Fatalf("broadcast missing art: %#v", broadcast)
	}
}
