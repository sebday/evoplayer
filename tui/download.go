package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
)

const (
	dlCtrlURL = iota
	dlCtrlDownload
	dlCtrlImport
	dlCtrlCount
)

func (m model) renderDownload() string {
	width := m.playlistInnerWidth()
	var b strings.Builder
	b.WriteString(styleMuted().Render("YouTube or SoundCloud URL"))
	b.WriteByte('\n')
	b.WriteString(styleMuted().Render("https://www.youtube.com/watch?v=… or soundcloud.com/…/…"))
	b.WriteString("\n\n")
	b.WriteString(m.renderDownloadField(width))
	b.WriteByte('\n')
	b.WriteString(m.renderDownloadAction(dlCtrlDownload, "⬇", "Download", width))
	b.WriteByte('\n')
	b.WriteString(styleMuted().Render(strings.Repeat("─", max(8, width))))
	b.WriteByte('\n')
	b.WriteString(m.renderDownloadAction(dlCtrlImport, "↥", "Import incoming", width))
	if status := m.downloadStatus(); status != "" {
		b.WriteString("\n\n")
		if m.job.Status == "error" || (m.err != "" && m.job.Status != "running") {
			b.WriteString(styleWarn().Render(status))
		} else {
			b.WriteString(styleMuted().Render(status))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) renderDownloadField(width int) string {
	inner := max(8, width-4)
	m.downloadURL.Width = inner
	border := colBorder
	if m.downloadControlSelected(dlCtrlURL) {
		border = colAccent
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(width)
	return box.Render(m.downloadURL.View())
}

func (m model) renderDownloadAction(idx int, icon, label string, width int) string {
	cursor := "  "
	text := icon + "  " + label
	busy := m.jobBusy()
	if m.downloadControlSelected(idx) {
		cursor = styleSelected().Render("> ")
		text = styleSelected().Render(text)
	} else if busy {
		text = styleMuted().Render(text)
	}
	row := cursor + text
	if lipgloss.Width(row) > width {
		return clipEllipsis(row, width)
	}
	return row
}

func (m model) downloadControlSelected(idx int) bool {
	return m.downloadSelected() && m.focus == focusPlaylist && m.playlistIdx == idx
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
				pct := st.Progress.Done
				if st.Progress.Total != 100 {
					pct = st.Progress.Done * 100 / st.Progress.Total
				}
				if st.Progress.Phase != "" {
					return fmt.Sprintf("%s  %d%%  %s", st.Name, pct, st.Progress.Phase)
				}
				return fmt.Sprintf("%s  %d%%", st.Name, pct)
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

func (m model) startImportIncoming() (tea.Model, tea.Cmd) {
	if m.jobBusy() {
		return m, nil
	}
	m.err = ""
	m.downloadURL.Blur()
	return m, startLibraryJob(m.env, "library.import", nil)
}

func (m model) activateDownloadControl() (tea.Model, tea.Cmd) {
	switch m.playlistIdx {
	case dlCtrlURL, dlCtrlDownload:
		return m.startURLDownload()
	case dlCtrlImport:
		return m.startImportIncoming()
	default:
		return m, nil
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
