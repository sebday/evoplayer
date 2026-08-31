//go:build !linux && !darwin && !freebsd

package worker

func DeprioritizeProcess() {}
