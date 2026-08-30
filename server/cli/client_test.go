package cli

import (
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
