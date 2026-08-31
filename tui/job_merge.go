package tui

import (
	"strings"

	"github.com/sebday/evoplayer/server/jobs"
)

func mergeJobState(prev, next jobs.State) jobs.State {
	if next.Status == "idle" || (next.Status == "" && next.Name == "") {
		return next
	}
	if prev.ID != "" && next.ID != "" && prev.ID != next.ID {
		return next
	}
	if prev.Status == "running" && next.Status == "running" {
		if strings.TrimSpace(next.Log) == "" && strings.TrimSpace(prev.Log) != "" {
			next.Log = prev.Log
		} else if len(next.Log) < len(prev.Log) {
			next.Log = prev.Log
		}
		if next.Progress == nil {
			next.Progress = prev.Progress
		} else if prev.Progress != nil && next.Progress.Total == 0 && prev.Progress.Total > 0 {
			next.Progress.Total = prev.Progress.Total
			if next.Progress.Done == 0 && prev.Progress.Done > 0 {
				next.Progress.Done = prev.Progress.Done
			}
		}
	}
	if next.Name == "" && prev.Name != "" {
		next.Name = prev.Name
	}
	if next.Status == "" && prev.Status != "" {
		next.Status = prev.Status
	}
	return next
}
