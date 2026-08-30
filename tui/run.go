package tui

import (
	"os"

	"github.com/blacktop/go-termimg"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sebday/evoplayer/server/cli"
	"github.com/sebday/evoplayer/server/paths"
)

func Run(env paths.Env, exe string) error {
	if err := cli.EnsureDaemon(env, exe); err != nil {
		return err
	}
	probeArtProtocol()
	m := newModel(env)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(newArtRestorer(os.Stdout, m.frames)))
	_, err := p.Run()
	_ = termimg.ClearAll()
	return err
}
