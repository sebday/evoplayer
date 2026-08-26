package testharness

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/ipc"
	"github.com/sebday/evoplayer/internal/paths"
)

type Harness struct {
	t         *testing.T
	Root      string
	MusicRoot string
	StateDir  string
	CacheDir  string
	Socket    string
	Runtime   string
	env       paths.Env
	cmd       *exec.Cmd
}

func New(t *testing.T) *Harness {
	t.Helper()
	root := t.TempDir()
	runtime := filepath.Join(root, "runtime")
	music := filepath.Join(root, "music")
	state := filepath.Join(root, "state", "panel", "player")
	cache := filepath.Join(root, "cache")
	socket := filepath.Join(runtime, "evoplayer-test.sock")
	for _, dir := range []string{runtime, music, state, cache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	h := &Harness{
		t:         t,
		Root:      root,
		MusicRoot: music,
		StateDir:  state,
		CacheDir:  cache,
		Socket:    socket,
		Runtime:   runtime,
	}
	h.env = paths.Env{
		MusicRoot:   music,
		StateDir:    state,
		CacheDir:    cache,
		SocketPath:  socket,
		DaemonLock:  filepath.Join(state, "daemon.lock"),
		LegacyRoot:  repoRoot(),
		TracksCacheDir: filepath.Join(cache, "tracks"),
		WaveformDir: filepath.Join(cache, "waveforms"),
		ArtDir:      filepath.Join(cache, "art"),
		LibraryDB:   filepath.Join(cache, "library.sqlite3"),
	}
	return h
}

func (h *Harness) Env() paths.Env {
	return h.env
}

func (h *Harness) Exe() string {
	if v := os.Getenv("EVOPLAYER_BIN"); v != "" {
		return v
	}
	candidates := []string{
		filepath.Join(repoRoot(), ".build", "evoplayer"),
		filepath.Join(os.Getenv("HOME"), ".local/bin/evoplayer"),
		filepath.Join(os.Getenv("HOME"), ".local/lib/evoplayer/evoplayer"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return "evoplayer"
}

func (h *Harness) StartDaemon() {
	h.t.Helper()
	exe := h.Exe()
	cmd := exec.Command(exe, "serve")
	cmd.Env = append(os.Environ(),
		"EVOPLAYER_ROOT="+repoRoot(),
		"EVO_PLAYER_MUSIC_ROOT="+h.MusicRoot,
		"EVO_PLAYER_MUSIC_STATE="+h.StateDir,
		"EVO_PLAYER_MUSIC_CACHE="+h.CacheDir,
		"XDG_RUNTIME_DIR="+h.Runtime,
		"EVOPLAYER_SOCKET="+h.Socket,
	)
	if err := cmd.Start(); err != nil {
		h.t.Fatal(err)
	}
	h.cmd = cmd
	for i := 0; i < 80; i++ {
		if h.DaemonUp() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	h.t.Fatal("daemon did not start")
}

func (h *Harness) Stop() {
	if h.cmd != nil && h.cmd.Process != nil {
		_ = h.cmd.Process.Kill()
		_, _ = h.cmd.Process.Wait()
		h.cmd = nil
	}
	_ = os.Remove(h.Socket)
}

func (h *Harness) DaemonUp() bool {
	resp, err := ipc.Call(h.Socket, ipc.Request{ID: 1, Method: "state.get"})
	return err == nil && resp.OK
}

func (h *Harness) IPC(method string, params any) ipc.Response {
	h.t.Helper()
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			h.t.Fatal(err)
		}
		raw = b
	}
	resp, err := ipc.Call(h.Socket, ipc.Request{ID: 1, Method: method, Params: raw})
	if err != nil {
		h.t.Fatal(err)
	}
	return resp
}

func (h *Harness) IPCData(method string, params any, out any) {
	h.t.Helper()
	resp := h.IPC(method, params)
	if !resp.OK {
		h.t.Fatalf("%s: %s", method, resp.Error)
	}
	if out == nil {
		return
	}
	b, err := json.Marshal(resp.Data)
	if err != nil {
		h.t.Fatal(err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		h.t.Fatal(err)
	}
}

func (h *Harness) WriteTrack(rel string, content []byte) string {
	h.t.Helper()
	p := filepath.Join(h.MusicRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if content == nil {
		content = []byte{0}
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		h.t.Fatal(err)
	}
	return p
}

func repoRoot() string {
	if v := os.Getenv("EVOPLAYER_ROOT"); v != "" {
		return v
	}
	wd, _ := os.Getwd()
	for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	return "."
}

// RepoRoot returns the evoplayer repository root for fixtures.
func RepoRoot() string {
	return repoRoot()
}
