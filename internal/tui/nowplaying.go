package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderNowPlaying(height int) string {
	innerW := max(8, m.mainWidth()-4)
	innerH := max(4, height-2)
	var body string
	if m.status.Path == "" {
		body = styleMuted().Render("nothing playing")
	} else {
		body = m.renderNowPlayingBody(innerW, innerH)
	}
	return fieldset("Now playing", body, m.mainWidth(), height, m.focus == focusList)
}

func (m model) renderNowPlayingBody(innerW, innerH int) string {
	waveH := 3
	remain := max(3, innerH-waveH-1)
	artCols := min(24, max(8, innerW/3))
	artRows := max(2, artCols/2)
	if artRows > remain {
		artRows = remain
		artCols = min(innerW, artRows*2)
	}
	art := renderArt(m.artImg, artCols, artRows)
	side := innerW >= artCols+16
	if !side {
		metaLines := 4
		budget := innerH - metaLines - 5
		if budget < artRows {
			artRows = max(2, budget)
			artCols = min(innerW, artRows*2)
			art = renderArt(m.artImg, artCols, artRows)
		}
	}
	metaW := innerW
	if side {
		metaW = max(8, innerW-artCols-1)
	}
	meta := m.renderNowMeta(metaW, artRows)
	var top string
	if side {
		top = lipgloss.JoinHorizontal(lipgloss.Top, art, " ", meta)
	} else {
		top = lipgloss.JoinVertical(lipgloss.Left, art, "", meta)
	}
	waveW := innerW
	peaks := downsamplePeaks(m.wfData, waveW)
	wave := styleMuted().Render(strings.Repeat("▁", waveW))
	if len(peaks) > 0 {
		playAt := playheadIndex(m.status.Position, m.status.Duration, waveW)
		wave = renderWaveform(peaks, playAt)
	}
	times := styleMuted().Render(fmt.Sprintf("%s / %s",
		playbackLabel(m.status.PositionLabel, m.status.Position),
		playbackLabel(m.status.DurationLabel, m.status.Duration),
	))
	return lipgloss.JoinVertical(lipgloss.Left, top, "", wave, times)
}

func (m model) renderNowMeta(width, height int) string {
	width = max(8, width)
	title := strings.TrimSpace(m.status.Title)
	if title == "" {
		title = m.status.Path
	}
	lines := []string{
		styleSelected().Render(trunc(title, width)),
	}
	if a := strings.TrimSpace(m.status.Artist); a != "" {
		lines = append(lines, trunc(a, width))
	}
	if al := strings.TrimSpace(m.status.Album); al != "" {
		lines = append(lines, styleMuted().Render(trunc(al, width)))
	}
	bits := make([]string, 0, 3)
	if g := strings.TrimSpace(m.status.Genre); g != "" {
		bits = append(bits, g)
	}
	if y := strings.TrimSpace(m.status.Year); y != "" {
		bits = append(bits, y)
	}
	if l := strings.TrimSpace(m.status.Label); l != "" {
		bits = append(bits, l)
	}
	if len(bits) > 0 {
		lines = append(lines, styleMuted().Render(trunc(strings.Join(bits, " · "), width)))
	}
	state := m.status.State
	if state == "" {
		state = "stopped"
	}
	extra := state
	if m.status.PlaylistCount > 0 {
		extra = fmt.Sprintf("%s  %d/%d", state, m.status.PlaylistPos, m.status.PlaylistCount)
	}
	if m.status.Liked {
		extra += "  ♥"
		lines = append(lines, styleGood().Render(trunc(extra, width)))
	} else {
		lines = append(lines, styleMuted().Render(trunc(extra, width)))
	}
	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func playbackLabel(label string, sec float64) string {
	if label != "" {
		return label
	}
	return dur(sec)
}

func trunc(s string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
