package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/soundcloud"
)

const (
	downloadLogLines = 8
)

const (
	dlCtrlURL = iota
	dlCtrlSoundCloud
	dlCtrlImport
	dlCtrlCancel
	dlCtrlCount
)

func (m model) downloadPaneWidth() int {
	g := m.playerGeom()
	return g.playlistW + g.artworkW
}

func (m model) renderDownloadPane(g playerGeom) string {
	width := m.downloadPaneWidth()
	innerW := paneInnerWidth(width)
	innerH := paneInnerHeight(g.bodyH)
	active := m.focus == focusPlaylist
	bottom := hint("⏎", "download", 3, active) + "  " + hint("i", "import", 3, active)
	if m.downloadJobCancellable() {
		bottom += "  " + hint("x", "cancel", 3, active)
	}
	return fieldsetPad("download", "", m.renderDownloadLanding(innerW, innerH), width, g.bodyH, active, panePadY, panePadX, bottom, "", 3)
}

func (m model) renderDownloadLanding(width, height int) string {
	boxW := min(width, max(28, width*3/4))
	if boxW > 56 {
		boxW = 56
	}
	var b strings.Builder
	b.WriteString(m.renderDownloadSearchBox(boxW))
	b.WriteString("\n\n")
	b.WriteString(m.renderDownloadActions(boxW))
	log := m.renderDownloadJobLog(boxW)
	status := m.downloadStatusLine(boxW)
	if log != "" {
		b.WriteString("\n\n")
		b.WriteString(log)
	}
	if status != "" {
		if log != "" {
			b.WriteString("\n")
		} else {
			b.WriteString("\n\n")
		}
		b.WriteString(status)
	}
	body := strings.TrimRight(b.String(), "\n")
	if height < 2 || width < 8 {
		return clipLines(body, max(1, height))
	}
	align := lipgloss.Center
	if strings.TrimSpace(m.job.Log) != "" {
		align = lipgloss.Top
	}
	return lipgloss.Place(width, height, lipgloss.Center, align, body)
}

func (m model) renderDownloadJobLog(width int) string {
	log := strings.TrimSpace(m.job.Log)
	if log == "" {
		return ""
	}
	lines := strings.Split(log, "\n")
	if len(lines) > downloadLogLines {
		lines = lines[len(lines)-downloadLogLines:]
	}
	var b strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = clipEllipsis(line, width)
		b.WriteString(styleDownloadLogLine(line))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func styleDownloadLogLine(line string) string {
	switch {
	case strings.HasPrefix(line, soundcloud.LogGlyphOK):
		return styleGood().Render(line)
	case strings.HasPrefix(line, soundcloud.LogGlyphFail), strings.HasPrefix(line, soundcloud.LogGlyphWarn):
		return styleWarn().Render(line)
	default:
		return styleMuted().Render(line)
	}
}

func (m model) renderDownloadSearchBox(width int) string {
	inner := max(8, width-4)
	m.downloadURL.Prompt = "> "
	m.downloadURL.Placeholder = "Paste a YouTube or SoundCloud URL..."
	m.downloadURL.Width = inner
	return fieldsetPad("url", "", m.downloadURL.View(), width, 3, m.downloadControlSelected(dlCtrlURL), 0, 1, "", "", 3)
}

func (m model) renderDownloadActions(width int) string {
	gap := "  "
	row := m.renderDownloadAction(dlCtrlSoundCloud, "import soundcloud") + gap +
		m.renderDownloadAction(dlCtrlImport, "import incoming")
	if lipgloss.Width(row) > width {
		row = clipEllipsis(row, width)
	}
	if !m.downloadJobCancellable() {
		return row
	}
	cancel := m.renderDownloadAction(dlCtrlCancel, "cancel")
	if lipgloss.Width(cancel) > width {
		cancel = clipEllipsis(cancel, width)
	}
	return row + "\n" + cancel
}

func (m model) renderDownloadAction(idx int, label string) string {
	cursor := "  "
	text := label
	busy := m.jobBusy()
	selected := m.downloadControlSelected(idx)
	if idx == dlCtrlCancel {
		if selected {
			cursor = styleSelected().Render("> ")
			text = styleWarn().Render(label)
		} else if m.downloadJobCancellable() {
			text = styleWarn().Render(label)
		} else {
			text = styleMuted().Render(label)
		}
		return cursor + text
	}
	if selected {
		cursor = styleSelected().Render("> ")
		text = styleSelected().Render(text)
	} else if busy {
		text = styleMuted().Render(text)
	} else {
		text = styleMuted().Render(text)
	}
	return cursor + text
}

func (m model) downloadControlSelected(idx int) bool {
	return m.downloadSelected() && m.focus == focusPlaylist && m.playlistIdx == idx
}

func (m model) downloadStatusLine(width int) string {
	status := m.downloadStatus()
	if status == "" {
		return ""
	}
	if m.job.Status == "error" || (m.err != "" && m.job.Status != "running") {
		return styleWarn().Render(clipEllipsis(status, width))
	}
	return styleMuted().Render(clipEllipsis(status, width))
}

func (m model) downloadStatus() string {
	if m.job.Status == "running" || m.job.Status == "done" || m.job.Status == "error" {
		if s := formatJob(m.job); s != "" {
			return s
		}
	}
	if m.err != "" {
		return m.err
	}
	return ""
}

func formatJob(st jobs.State) string {
	switch st.Status {
	case "running":
		if st.Progress != nil {
			if st.Progress.Total > 0 {
				if st.Progress.Phase != "" {
					return fmt.Sprintf("%s  %d/%d  %s", st.Name, st.Progress.Done, st.Progress.Total, st.Progress.Phase)
				}
				return fmt.Sprintf("%s  %d/%d", st.Name, st.Progress.Done, st.Progress.Total)
			}
			if st.Progress.Phase != "" {
				return fmt.Sprintf("%s  %s", st.Name, st.Progress.Phase)
			}
		}
		name := st.Name
		if name == "" {
			name = "job"
		}
		return name + "  running…"
	case "error":
		if st.Error != "" {
			return st.Error
		}
		return st.Name + "  failed"
	case "done":
		name := st.Name
		if name == "" {
			name = "job"
		}
		return name + "  done"
	default:
		return ""
	}
}

func (m model) downloadJobCancellable() bool {
	if m.job.Status != "running" {
		return false
	}
	switch m.job.Name {
	case "download", "download-url", "import":
		return true
	default:
		return false
	}
}

func (m model) jobBusy() bool {
	return m.job.Status == "running"
}

func (m model) libraryScanning() bool {
	return m.job.Status == "running" && m.job.Name == "scan"
}

func (m model) startURLDownload() (tea.Model, tea.Cmd) {
	url := strings.TrimSpace(m.downloadURL.Value())
	if url == "" {
		m.err = "url required"
		return m, nil
	}
	if m.jobBusy() {
		return m, nil
	}
	m.err = ""
	m.downloadURL.Blur()
	return m, startLibraryJob(m.env, "library.download", map[string]any{
		"url":    url,
		"import": false,
	})
}

func (m model) startSoundCloudDownload() (tea.Model, tea.Cmd) {
	if m.jobBusy() {
		return m, nil
	}
	m.err = ""
	m.downloadURL.Blur()
	return m, startLibraryJob(m.env, "library.soundcloud.download", nil)
}

func (m model) startImportIncoming() (tea.Model, tea.Cmd) {
	if m.jobBusy() {
		return m, nil
	}
	m.err = ""
	m.downloadURL.Blur()
	return m, startLibraryJob(m.env, "library.import", nil)
}

func (m model) cancelDownloadJob() (tea.Model, tea.Cmd) {
	if !m.downloadJobCancellable() {
		return m, nil
	}
	m.err = ""
	m.downloadURL.Blur()
	return m, cancelJobCmd(m.env)
}

func (m model) activateDownloadControl() (tea.Model, tea.Cmd) {
	switch m.playlistIdx {
	case dlCtrlURL:
		return m.startURLDownload()
	case dlCtrlSoundCloud:
		return m.startSoundCloudDownload()
	case dlCtrlImport:
		return m.startImportIncoming()
	case dlCtrlCancel:
		return m.cancelDownloadJob()
	default:
		return m, nil
	}
}

func cancelJobCmd(env paths.Env) tea.Cmd {
	return func() tea.Msg {
		if err := cancelJob(env); err != nil {
			return errMsg{err: err}
		}
		st, err := fetchJob(env)
		return jobMsg{state: st, err: err}
	}
}

func pollJob(env paths.Env) tea.Cmd {
	return func() tea.Msg {
		st, err := fetchJob(env)
		return jobMsg{state: st, err: err}
	}
}

func startLibraryJob(env paths.Env, method string, params map[string]any) tea.Cmd {
	return func() tea.Msg {
		st, err := runLibraryJob(env, method, params)
		return jobMsg{state: st, err: err}
	}
}
