package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sebday/evoplayer/server/daemon"
	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/secrets"
	"github.com/sebday/evoplayer/server/status"
)

func DaemonUp(env paths.Env) bool {
	resp, err := ipc.Call(env.SocketPath, ipc.Request{ID: 1, Method: "state.get"})
	if err != nil {
		return false
	}
	return resp.OK
}

func EnsureDaemon(env paths.Env, exe string) error {
	if DaemonUp(env) {
		if daemonBinaryStale(env, exe) {
			restartDaemon(env)
		} else {
			return nil
		}
	}
	cleanupStaleDaemon(env)
	if DaemonUp(env) {
		return nil
	}

	secrets.Load()
	cmd := exec.Command(exe, "serve")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if devNull, err := os.Open(os.DevNull); err == nil {
		defer devNull.Close()
		cmd.Stdin = devNull
	}
	logFile, logStart, err := openDaemonLog(env.DaemonLog)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), "EVOPLAYER_ROOT="+env.RepoRoot)
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	for i := 0; i < 80; i++ {
		if DaemonUp(env) {
			return nil
		}
		select {
		case err := <-done:
			if DaemonUp(env) {
				return nil
			}
			if msg := daemonLogSince(env.DaemonLog, logStart); msg != "" {
				return fmt.Errorf("evoplayer: daemon exited: %s", msg)
			}
			if err != nil {
				return fmt.Errorf("evoplayer: daemon exited: %w", err)
			}
			return fmt.Errorf("evoplayer: daemon exited before socket was ready")
		default:
		}
		time.Sleep(50 * time.Millisecond)
	}

	_ = cmd.Process.Kill()
	if msg := daemonLogSince(env.DaemonLog, logStart); msg != "" {
		return fmt.Errorf("evoplayer: daemon did not start: %s", msg)
	}
	return fmt.Errorf("evoplayer: daemon did not start (log: %s)", env.DaemonLog)
}

const daemonLogMaxBytes = 1 << 20

func openDaemonLog(path string) (*os.File, int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, 0, err
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if st, err := os.Stat(path); err == nil && st.Size() > daemonLogMaxBytes {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, 0, err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, err
	}
	return f, st.Size(), nil
}

func daemonLogSince(path string, offset int64) string {
	b, err := os.ReadFile(path)
	if err != nil || int64(len(b)) <= offset {
		return ""
	}
	return strings.TrimSpace(string(b[offset:]))
}

// daemonPID returns the pid in daemon.lock only while a daemon still holds the lock.
func daemonPID(env paths.Env) int {
	if !daemon.LockHeld(env.DaemonLock) {
		return 0
	}
	pid, err := daemon.ReadLockPID(env.DaemonLock)
	if err != nil || !daemon.ProcessAlive(pid) {
		return 0
	}
	return pid
}

func daemonBinaryStale(env paths.Env, exe string) bool {
	pid := daemonPID(env)
	if pid <= 0 {
		return false
	}
	want, err := filepath.EvalSymlinks(exe)
	if err != nil {
		want = exe
	}
	wantAbs, err := filepath.Abs(want)
	if err != nil {
		wantAbs = want
	}
	run, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return true
	}
	if i := strings.Index(run, " (deleted)"); i >= 0 {
		run = run[:i]
	}
	runAbs, err := filepath.Abs(run)
	if err != nil {
		runAbs = run
	}
	return wantAbs != runAbs
}

func restartDaemon(env paths.Env) {
	if DaemonUp(env) {
		_ = savePlayerState(env)
	}
	if pid := daemonPID(env); pid > 0 {
		_ = daemon.StopProcess(pid)
		for i := 0; i < 300; i++ {
			if !daemon.ProcessAlive(pid) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if daemon.ProcessAlive(pid) {
			_ = daemon.KillProcess(pid)
			for i := 0; i < 20; i++ {
				if !daemon.ProcessAlive(pid) {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
	_ = os.Remove(env.SocketPath)
}

func cleanupStaleDaemon(env paths.Env) {
	if DaemonUp(env) {
		return
	}
	pid := daemonPID(env)
	if pid <= 0 {
		_ = os.Remove(env.SocketPath)
		return
	}
	_ = daemon.StopProcess(pid)
	for i := 0; i < 20; i++ {
		if !daemon.ProcessAlive(pid) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = os.Remove(env.SocketPath)
}

func IPC(env paths.Env, method string, params interface{}) (ipc.Response, error) {
	var raw ipc.Request
	raw.Method = method
	raw.ID = 1
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return ipc.Response{}, err
		}
		raw.Params = b
	}
	return ipc.Call(env.SocketPath, raw)
}

func PlaybackStatus(env paths.Env) (playback.Status, error) {
	resp, err := IPC(env, "state.get", nil)
	if err != nil {
		return playback.Status{}, err
	}
	if !resp.OK {
		return playback.Status{}, fmt.Errorf("%s", resp.Error)
	}
	st, err := decodeStatus(resp.Data)
	if err != nil {
		return playback.Status{}, err
	}
	return status.EnrichFull(env, st), nil
}
