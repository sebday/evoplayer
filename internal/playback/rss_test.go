package playback_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/ipc"
)

func TestDaemonRSSAfterPlaylistSave(t *testing.T) {
	exe := filepath.Join("..", "..", ".build", "evoplayer")
	if _, err := os.Stat(exe); err != nil {
		t.Skip("missing .build/evoplayer:", err)
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
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
	paths := make([]string, 200)
	for i := range paths {
		paths[i] = wav
	}

	_, err := ipc.Call(socket, ipc.Request{
		ID:     2,
		Method: "library.current.save",
		Params: mustJSON(map[string]interface{}{"paths": paths}),
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		_, err := ipc.Call(socket, ipc.Request{
			ID:     3 + i,
			Method: "queue.play_current",
			Params: mustJSON(map[string]interface{}{"start_path": wav}),
		})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	rss, err := readRSSKB(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	const maxRSSKB = 400 * 1024 // 400MB
	if rss > maxRSSKB {
		t.Fatalf("rss %d KiB exceeds %d KiB", rss, maxRSSKB)
	}
}

func readRSSKB(pid int) (int64, error) {
	out, err := exec.Command("ps", "-o", "rss=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return 0, err
	}
	var kb int64
	_, err = fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &kb)
	return kb, err
}
