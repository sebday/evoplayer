package playback_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/ipc"
)

func TestDaemonPositionAdvances(t *testing.T) {
	exe := filepath.Join("..", "..", ".build", "evoplayer")
	if _, err := os.Stat(exe); err != nil {
		t.Skip("missing .build/evoplayer:", err)
	}

	runtime := t.TempDir()
	socket := filepath.Join(runtime, "evoplayer.sock")
	env := append(os.Environ(),
		"XDG_RUNTIME_DIR="+runtime,
		"XDG_STATE_HOME="+filepath.Join(runtime, "state"),
		"XDG_CACHE_HOME="+filepath.Join(runtime, "cache"),
	)

	cmd := exec.Command(exe, "serve")
	cmd.Env = env
	var serveLog bytes.Buffer
	cmd.Stderr = &serveLog
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := ipc.Call(socket, ipc.Request{ID: 1, Method: "state.get"})
		if err == nil && resp.OK {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	wav := writeIntegrationWAV(t)
	_, err := ipc.Call(socket, ipc.Request{
		ID:     2,
		Method: "queue.replace",
		Params: mustJSON(map[string]interface{}{
			"paths":      []string{wav},
			"start_path": wav,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1500 * time.Millisecond)

	resp, err := ipc.Call(socket, ipc.Request{ID: 3, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("state.get failed: %s", resp.Error)
	}
	pos := jsonNumber(resp.Data, "position")
	if pos < 0.5 {
		t.Fatalf("expected position >= 0.5 after playback, got %v data=%v serveLog=%q", pos, resp.Data, serveLog.String())
	}

	_, err = ipc.Call(socket, ipc.Request{
		ID:     4,
		Method: "playback.seek",
		Params: mustJSON(map[string]float64{"seconds": 1.5}),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)

	resp, err = ipc.Call(socket, ipc.Request{ID: 5, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	seekPos := jsonNumber(resp.Data, "position")
	if seekPos < 1.2 || seekPos > 1.9 {
		t.Fatalf("expected seek position near 1.5, got %v", seekPos)
	}

	_, err = ipc.Call(socket, ipc.Request{
		ID:     6,
		Method: "playback.next",
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	resp, err = ipc.Call(socket, ipc.Request{ID: 7, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("state.get after next failed: %s", resp.Error)
	}
	pos = jsonNumber(resp.Data, "position")
	if pos < 0.5 {
		t.Fatalf("expected position >= 0.5 after next/reload, got %v data=%v serveLog=%q", pos, resp.Data, serveLog.String())
	}
}

func TestDaemonPlayCurrentQueue(t *testing.T) {
	exe := filepath.Join("..", "..", ".build", "evoplayer")
	if _, err := os.Stat(exe); err != nil {
		t.Skip("missing .build/evoplayer:", err)
	}

	runtime := t.TempDir()
	socket := filepath.Join(runtime, "evoplayer.sock")
	stateHome := filepath.Join(runtime, "state")
	env := append(os.Environ(),
		"XDG_RUNTIME_DIR="+runtime,
		"XDG_STATE_HOME="+stateHome,
		"XDG_CACHE_HOME="+filepath.Join(runtime, "cache"),
	)

	cmd := exec.Command(exe, "serve")
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := ipc.Call(socket, ipc.Request{ID: 1, Method: "state.get"})
		if err == nil && resp.OK {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	wav1 := writeIntegrationWAV(t)
	wav2 := writeIntegrationWAVNamed(t, "second.wav")
	paths := []string{wav1, wav2}

	resp, err := ipc.Call(socket, ipc.Request{
		ID:     10,
		Method: "queue.play_current",
		Params: mustJSON(map[string]interface{}{
			"paths":      paths,
			"start_path": wav2,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("queue.play_current failed: %s", resp.Error)
	}

	time.Sleep(1200 * time.Millisecond)

	resp, err = ipc.Call(socket, ipc.Request{ID: 11, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("state.get failed: %s", resp.Error)
	}
	if got := jsonString(resp.Data, "path"); got != wav2 {
		t.Fatalf("expected path %q, got %q", wav2, got)
	}
	pos := jsonNumber(resp.Data, "position")
	if pos < 0.3 {
		t.Fatalf("expected position >= 0.3 after play_current, got %v", pos)
	}
}

func TestDaemonReplaceQueueChangesTrack(t *testing.T) {
	exe := filepath.Join("..", "..", ".build", "evoplayer")
	if _, err := os.Stat(exe); err != nil {
		t.Skip("missing .build/evoplayer:", err)
	}

	runtime := t.TempDir()
	socket := filepath.Join(runtime, "evoplayer.sock")
	env := append(os.Environ(),
		"XDG_RUNTIME_DIR="+runtime,
		"XDG_STATE_HOME="+filepath.Join(runtime, "state"),
		"XDG_CACHE_HOME="+filepath.Join(runtime, "cache"),
	)

	cmd := exec.Command(exe, "serve")
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := ipc.Call(socket, ipc.Request{ID: 1, Method: "state.get"})
		if err == nil && resp.OK {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	wav1 := writeIntegrationWAV(t)
	wav2 := writeIntegrationWAVNamed(t, "second.wav")
	paths := []string{wav1, wav2}

	_, err := ipc.Call(socket, ipc.Request{
		ID:     20,
		Method: "queue.replace",
		Params: mustJSON(map[string]interface{}{
			"paths":      paths,
			"start_path": wav1,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1200 * time.Millisecond)

	resp, err := ipc.Call(socket, ipc.Request{ID: 21, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if got := jsonString(resp.Data, "path"); got != wav1 {
		t.Fatalf("expected first track %q, got %q", wav1, got)
	}

	_, err = ipc.Call(socket, ipc.Request{
		ID:     22,
		Method: "queue.replace",
		Params: mustJSON(map[string]interface{}{
			"paths":      paths,
			"start_path": wav2,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1200 * time.Millisecond)

	resp, err = ipc.Call(socket, ipc.Request{ID: 23, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if got := jsonString(resp.Data, "path"); got != wav2 {
		t.Fatalf("expected second track %q after replace, got %q", wav2, got)
	}
	if count := int(jsonNumber(resp.Data, "playlist_count")); count != 2 {
		t.Fatalf("expected playlist_count 2, got %d", count)
	}
	pos := jsonNumber(resp.Data, "position")
	if pos < 0.3 {
		t.Fatalf("expected position >= 0.3 on second track, got %v", pos)
	}
}

func writeIntegrationWAVNamed(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=880:duration=3", "-ar", "44100", "-ac", "2", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skip("ffmpeg required for integration test:", err, string(out))
	}
	return path
}

func jsonString(data interface{}, key string) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func mustJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func jsonNumber(data interface{}, key string) float64 {
	m, ok := data.(map[string]interface{})
	if !ok {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func writeIntegrationWAV(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.wav")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=3", "-ar", "44100", "-ac", "2", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skip("ffmpeg required for integration test:", err, string(out))
	}
	return path
}
