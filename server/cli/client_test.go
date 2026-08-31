package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
)

func TestRestartDaemonRemovesSocket(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "evoplayer.sock")
	if err := os.WriteFile(sock, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := paths.Env{
		SocketPath: sock,
		DaemonLock: filepath.Join(dir, "missing.lock"),
	}
	restartDaemon(env)
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatalf("socket still present: %v", err)
	}
}

func TestDaemonBinaryNotStaleForRunningProcess(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pid := os.Getpid()
	env := paths.Env{DaemonLock: filepath.Join(t.TempDir(), "daemon.lock")}
	if err := os.WriteFile(env.DaemonLock, []byte(fmt.Sprintf("%d\n", pid)), 0o644); err != nil {
		t.Fatal(err)
	}
	if daemonBinaryStale(env, exe) {
		t.Fatalf("daemon should not be stale for running process exe=%s", exe)
	}
}
