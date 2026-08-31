package tui

import (
	"testing"

	"github.com/sebday/evoplayer/server/jobs"
)

func TestMergeJobStateKeepsLogOnEmptyUpdate(t *testing.T) {
	prev := jobs.State{
		ID:     "job-1",
		Name:   "download-url",
		Status: "running",
		Log:    "· fetching track list\n· 25 tracks in collection\n",
		Progress: &jobs.Progress{Phase: "fetching track list (8s)"},
	}
	next := jobs.State{
		ID:     "job-1",
		Name:   "download-url",
		Status: "running",
	}
	got := mergeJobState(prev, next)
	if got.Log != prev.Log {
		t.Fatalf("log = %q, want %q", got.Log, prev.Log)
	}
	if got.Progress == nil || got.Progress.Phase != "fetching track list (8s)" {
		t.Fatalf("progress = %+v", got.Progress)
	}
}
