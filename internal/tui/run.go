package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebday/evoplayer/internal/cli"
	"github.com/sebday/evoplayer/internal/paths"
)

func Run(env paths.Env, exe string) error {
	if err := cli.EnsureDaemon(env, exe); err != nil {
		return err
	}
	p := tea.NewProgram(newModel(env), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
