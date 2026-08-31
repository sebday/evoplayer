package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sebday/evoplayer/server/cli"
	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/secrets"
)

type settingsMsg struct {
	view config.JSONView
	err  error
}

type settingsSaveMsg struct {
	view config.JSONView
	err  error
}

func loadSettings(env paths.Env) tea.Cmd {
	return func() tea.Msg {
		view, err := config.JSON(env.MusicConfig, env.MusicRoot)
		if err != nil {
			return settingsMsg{err: err}
		}
		view.Soundcloud.OAuthSource = secrets.SoundcloudOAuth().Source
		return settingsMsg{view: view}
	}
}

func saveSettingsRoot(env paths.Env, path string) tea.Cmd {
	return func() tea.Msg {
		resp, err := cli.IPC(env, "config.set", map[string]string{
			"section": "paths",
			"key":     "root",
			"value":   strings.TrimRight(strings.TrimSpace(path), "/"),
		})
		if err != nil {
			return settingsSaveMsg{err: err}
		}
		var view config.JSONView
		if err := decodeData(resp, &view); err != nil {
			return settingsSaveMsg{err: err}
		}
		return settingsSaveMsg{view: view}
	}
}

func (m model) renderSettings(width int) string {
	user := strings.TrimSpace(m.settings.Soundcloud.User)
	if user == "" {
		user = "unset"
	}
	oauth := strings.TrimSpace(m.settings.Soundcloud.OAuthSource)
	if oauth == "" {
		oauth = "missing"
	}
	viz := m.vizMode.String()
	if viz == "none" {
		viz = "off"
	}

	inner := max(8, width-4)
	m.settingsPath.Prompt = "> "
	m.settingsPath.Placeholder = "/path/to/music"
	m.settingsPath.Width = inner
	libraryField := fieldsetPad("library", "", m.settingsPath.View(), width, 3, m.settingsSelected() && m.focus == focusPlaylist, 0, 1, "", "", 3)

	var b strings.Builder
	b.WriteString(libraryField)
	b.WriteString("\n\n")
	b.WriteString(styleMuted().Render("soundcloud"))
	b.WriteByte('\n')
	b.WriteString("  " + styleText().Render(clipEllipsis(user, max(4, width-2))))
	b.WriteByte('\n')
	b.WriteString("  " + styleMuted().Render("oauth") + "  " + styleText().Render(oauth))
	b.WriteString("\n\n")
	b.WriteString(styleMuted().Render("visualizer"))
	b.WriteByte('\n')
	b.WriteString("  " + styleText().Render(viz))
	return strings.TrimRight(b.String(), "\n")
}
