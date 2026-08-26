package playback_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/ipc"
)

// TestPlaybackSurvivesVizSubscribe simulates switching to the Now Playing tab
// (viz.subscribe) while audio is playing.
func TestPlaybackSurvivesVizSubscribe(t *testing.T) {
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

	wav := writeIntegrationWAV(t)
	paths := []string{wav}

	_, err := ipc.Call(socket, ipc.Request{
		ID:     2,
		Method: "queue.replace",
		Params: mustJSON(map[string]interface{}{
			"paths":      paths,
			"start_path": wav,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(800 * time.Millisecond)

	resp, err := ipc.Call(socket, ipc.Request{ID: 3, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if jsonString(resp.Data, "state") != "playing" {
		t.Fatalf("expected playing before viz.subscribe, got %v", resp.Data)
	}
	posBefore := jsonNumber(resp.Data, "position")

	_, err = ipc.Call(socket, ipc.Request{ID: 4, Method: "viz.subscribe"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	resp, err = ipc.Call(socket, ipc.Request{ID: 5, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if jsonString(resp.Data, "state") != "playing" {
		t.Fatalf("expected playing after viz.subscribe, got %v", resp.Data)
	}
	posAfter := jsonNumber(resp.Data, "position")
	if posAfter < posBefore+0.15 {
		t.Fatalf("expected position to advance after viz.subscribe, before=%v after=%v", posBefore, posAfter)
	}

	// Identical replace must not restart playback (guards against duplicate queue ops).
	_, err = ipc.Call(socket, ipc.Request{
		ID:     6,
		Method: "queue.replace",
		Params: mustJSON(map[string]interface{}{
			"paths":      paths,
			"start_path": wav,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)

	resp, err = ipc.Call(socket, ipc.Request{ID: 7, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if jsonString(resp.Data, "state") != "playing" {
		t.Fatalf("expected playing after duplicate replace, got %v", resp.Data)
	}
	if jsonNumber(resp.Data, "position") < posAfter+0.05 {
		t.Fatalf("duplicate replace should not restart track (position did not advance)")
	}
}

// TestPlaybackSurvivesVizUnsubscribeResubscribe simulates the old QML path where
// leaving Now Playing unsubscribed viz and returning re-subscribed while audio plays.
func TestPlaybackSurvivesVizUnsubscribeResubscribe(t *testing.T) {
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

	wav := writeIntegrationWAV(t)
	paths := []string{wav}

	_, err := ipc.Call(socket, ipc.Request{
		ID:     2,
		Method: "queue.replace",
		Params: mustJSON(map[string]interface{}{
			"paths":      paths,
			"start_path": wav,
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(800 * time.Millisecond)

	_, err = ipc.Call(socket, ipc.Request{ID: 3, Method: "viz.subscribe"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ipc.Call(socket, ipc.Request{ID: 4, Method: "viz.unsubscribe"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ipc.Call(socket, ipc.Request{ID: 5, Method: "viz.subscribe"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)

	resp, err := ipc.Call(socket, ipc.Request{ID: 6, Method: "state.get"})
	if err != nil {
		t.Fatal(err)
	}
	if jsonString(resp.Data, "state") != "playing" {
		t.Fatalf("expected playing after viz churn, got %v", resp.Data)
	}
}
