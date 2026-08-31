package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sebday/evoplayer/server/paths"
)

func (m model) libraryScanning() bool {
	return m.job.Status == "running" && m.job.Name == "scan"
}

func jobPollInterval() time.Duration {
	return 250 * time.Millisecond
}

func pollJob(env paths.Env) tea.Cmd {
	return tea.Tick(jobPollInterval(), func(time.Time) tea.Msg {
		st, err := fetchJob(env)
		return jobMsg{state: st, err: err}
	})
}
