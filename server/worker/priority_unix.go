//go:build darwin || freebsd

package worker

import "syscall"

func DeprioritizeProcess() {
	_ = syscall.Setpriority(syscall.PRIO_PROCESS, 0, 19)
}
