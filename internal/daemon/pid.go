package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func writeLockPID(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := fmt.Fprintf(f, "%d\n", os.Getpid())
	return err
}

func ReadLockPID(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	line := strings.TrimSpace(string(b))
	if line == "" {
		return 0, nil
	}
	pid, err := strconv.Atoi(line)
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func StopProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

func KillProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGKILL)
}
