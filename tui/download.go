package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
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
	var bottom string
	if m.downloadJobCancellable() {
		bottom = hint("⏎", "cancel", 3, active) + "  " + hint("x", "cancel", 3, active)
	} else {
		bottom = hint("⏎", "download", 3, active)
	}
	lines := m.buildDownloadInnerLines(innerW, innerH)
	return fieldsetBodyLines("download", "", lines, width, g.bodyH, active, panePadY, panePadX, bottom, "", 3)
}

func (m model) buildDownloadInnerLines(innerW, innerH int) []string {
	boxW := min(innerW, max(28, innerW*3/4))
	if boxW > 56 {
		boxW = 56
	}
	urlBox := m.renderDownloadSearchBox(boxW)
	urlH := lipgloss.Height(urlBox)
	urlBlock := lipgloss.Place(innerW, urlH, lipgloss.Center, lipgloss.Top, urlBox)

	var body []string
	body = append(body, strings.Split(urlBlock, "\n")...)

	usedBelow := 0
	if m.downloadJobCancellable() {
		if status := m.renderDownloadStatusLine(boxW); status != "" {
			body = append(body, status)
			usedBelow++
		}
	}

	logMax := max(1, innerH-urlH-usedBelow)
	log := m.renderDownloadJobLog(boxW, logMax)
	if log != "" {
		body = append(body, strings.Split(log, "\n")...)
	}

	if m.downloadJobCancellable() || strings.TrimSpace(m.job.Log) != "" {
		for len(body) < innerH {
			body = append(body, "")
		}
		if len(body) > innerH {
			body = body[:innerH]
		}
		return body
	}
	if len(body) >= innerH {
		return body[:innerH]
	}
	padTop := (innerH - len(body)) / 2
	lines := make([]string, innerH)
	for i := range lines {
		src := i - padTop
		if src >= 0 && src < len(body) {
			lines[i] = body[src]
		}
	}
	return lines
}

func (m model) renderDownloadJobLog(width, maxLines int) string {
	log := strings.TrimSpace(m.job.Log)
	if log == "" {
		return ""
	}
	lines := strings.Split(log, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
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
	case strings.HasPrefix(line, jobs.LogGlyphOK):
		return styleGood().Render(line)
	case strings.HasPrefix(line, jobs.LogGlyphFail), strings.HasPrefix(line, jobs.LogGlyphWarn):
		return styleWarn().Render(line)
	default:
		return styleMuted().Render(line)
	}
}

func (m model) renderDownloadSearchBox(width int) string {
	if m.downloadJobCancellable() {
		inner := max(8, width-4)
		cancel := styleDanger().Render(padExact("cancel", inner))
		return fieldsetPadAlert("url", cancel, width, 3, 0, 1, 3)
	}
	inner := max(8, width-4)
	m.downloadURL.Prompt = "> "
	m.downloadURL.Placeholder = "Paste YouTube or SoundCloud URL (track, playlist, artist, likes)…"
	m.downloadURL.Width = inner
	active := m.downloadSelected() && m.focus == focusPlaylist && !m.jobBusy()
	return fieldsetPad("url", "", m.downloadURL.View(), width, 3, active, 0, 1, "", "", 3)
}

func (m model) renderDownloadStatusLine(width int) string {
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
		name := st.Name
		if name == "" {
			name = "job"
		}
		if st.Progress != nil {
			if st.Progress.Total > 0 {
				if st.Progress.Phase != "" {
					return fmt.Sprintf("%s  %d/%d  %s", name, st.Progress.Done, st.Progress.Total, st.Progress.Phase)
				}
				return fmt.Sprintf("%s  %d/%d", name, st.Progress.Done, st.Progress.Total)
			}
			if st.Progress.Done > 0 && st.Progress.Phase != "" {
				return fmt.Sprintf("%s  %s (%d)", name, st.Progress.Phase, st.Progress.Done)
			}
			if st.Progress.Phase != "" {
				return fmt.Sprintf("%s  %s", name, st.Progress.Phase)
			}
		}
		if line := lastJobLogLine(st.Log); line != "" {
			return name + "  " + line
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
	if m.downloadJobCancellable() {
		return m.cancelDownloadJob()
	}
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
	m.job = jobs.State{
		Name:     "download-url",
		Status:   "running",
		Progress: &jobs.Progress{Phase: "starting…"},
	}
	return m, tea.Batch(
		startLibraryJob(m.env, "library.download", map[string]any{
			"url":    url,
			"import": true,
		}),
		pollJob(m.env),
	)
}

func (m model) cancelDownloadJob() (tea.Model, tea.Cmd) {
	if !m.downloadJobCancellable() {
		return m, nil
	}
	m.err = ""
	m.downloadURL.Blur()
	return m, cancelJobCmd(m.env)
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

func jobPollInterval() time.Duration {
	return 250 * time.Millisecond
}

func lastJobLogLine(log string) string {
	lines := strings.Split(strings.TrimSpace(log), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		for _, prefix := range []string{
			jobs.LogGlyphOK + " ",
			jobs.LogGlyphSkip + " ",
			jobs.LogGlyphFail + " ",
			jobs.LogGlyphInfo + " ",
			jobs.LogGlyphWarn + " ",
		} {
			line = strings.TrimPrefix(line, prefix)
		}
		return line
	}
	return ""
}

func pollJob(env paths.Env) tea.Cmd {
	return tea.Tick(jobPollInterval(), func(time.Time) tea.Msg {
		st, err := fetchJob(env)
		return jobMsg{state: st, err: err}
	})
}

func startLibraryJob(env paths.Env, method string, params map[string]any) tea.Cmd {
	return func() tea.Msg {
		st, err := runLibraryJob(env, method, params)
		return jobMsg{state: st, err: err}
	}
}

// renderDownloadLanding is kept for tests; production uses buildDownloadInnerLines.
func (m model) renderDownloadLanding(width, height int) string {
	lines := m.buildDownloadInnerLines(width, height)
	return strings.Join(lines, "\n")
}

func styleDanger() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colLiked).Bold(true)
}
