package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebday/evoplayer/server/cli"
	"github.com/sebday/evoplayer/server/paths"
)

func Run(env paths.Env, exe string) error {
	if err := cli.EnsureDaemon(env, exe); err != nil {
		return err
	}
	probeArtProtocol()
	p := tea.NewProgram(newModel(env), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
