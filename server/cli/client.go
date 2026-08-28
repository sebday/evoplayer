package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	cmd.Env = append(os.Environ(), "EVOPLAYER_ROOT="+env.LegacyRoot)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
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
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
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
	msg := strings.TrimSpace(stderr.String())
	if msg != "" {
		return fmt.Errorf("evoplayer: daemon did not start: %s", msg)
	}
	return fmt.Errorf("evoplayer: daemon did not start")
}

func daemonBinaryStale(env paths.Env, exe string) bool {
	pid, err := daemon.ReadLockPID(env.DaemonLock)
	if err != nil || pid <= 0 || !daemon.ProcessAlive(pid) {
		return false
	}
	wantStat, err := os.Stat(exe)
	if err != nil {
		return false
	}
	runStat, err := os.Stat(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return true
	}
	return !os.SameFile(wantStat, runStat)
}

func restartDaemon(env paths.Env) {
	pid, _ := daemon.ReadLockPID(env.DaemonLock)
	if pid > 0 && daemon.ProcessAlive(pid) {
		_ = daemon.StopProcess(pid)
		for i := 0; i < 40; i++ {
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
	pid, err := daemon.ReadLockPID(env.DaemonLock)
	if err != nil || pid <= 0 {
		_ = os.Remove(env.SocketPath)
		return
	}
	if !daemon.ProcessAlive(pid) {
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
	return status.Enrich(env, st), nil
}
