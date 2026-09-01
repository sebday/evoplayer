package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

type tagEditOpenMsg struct {
	path string
	tags library.TrackTags
	err  error
}

type tagEditMsg struct {
	path  string
	track library.Track
	err   error
}

func newTagField() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 256
	ti.Width = 40
	return ti
}

func loadTagEditFields(env paths.Env, path string) tea.Cmd {
	return func() tea.Msg {
		tags, err := fetchTrackTags(env, path)
		return tagEditOpenMsg{path: path, tags: tags, err: err}
	}
}

func saveTrackTags(env paths.Env, path string, tags library.TrackTags) tea.Cmd {
	return func() tea.Msg {
		track, err := setTrackTags(env, path, tags)
		return tagEditMsg{path: path, track: track, err: err}
	}
}

func (m model) tagEditInputs() [5]*textinput.Model {
	return [5]*textinput.Model{&m.tagEditTitle, &m.tagEditArtist, &m.tagEditYear, &m.tagEditGenre, &m.tagEditLabel}
}

func (m model) blurTagEditFields() model {
	for _, ti := range m.tagEditInputs() {
		ti.Blur()
	}
	return m
}

func (m model) tagEditFocusedIndex() int {
	for i, ti := range m.tagEditInputs() {
		if ti.Focused() {
			return i
		}
	}
	return -1
}

func (m model) focusTagEditField() (model, tea.Cmd) {
	m = m.blurTagEditFields()
	if m.tagEditFocus < 0 || m.tagEditFocus >= len(m.tagEditInputs()) {
		m.tagEditFocus = 0
	}
	var cmd tea.Cmd
	switch m.tagEditFocus {
	case 0:
		cmd = m.tagEditTitle.Focus()
	case 1:
		cmd = m.tagEditArtist.Focus()
	case 2:
		cmd = m.tagEditYear.Focus()
	case 3:
		cmd = m.tagEditGenre.Focus()
	case 4:
		cmd = m.tagEditLabel.Focus()
	}
	return m, cmd
}

func (m model) configureTagEditWidths(width int) model {
	inner := max(8, width-4)
	for _, ti := range m.tagEditInputs() {
		ti.Width = max(6, inner)
	}
	return m
}

func (m model) renderTagEditor(width int) string {
	if m.tagEditBusy && m.tagEditTitle.Value() == "" && m.tagEditArtist.Value() == "" {
		return styleMuted().Render("loading…")
	}
	if m.tagEditBusy {
		return styleMuted().Render("saving…")
	}
	m = m.configureTagEditWidths(width)
	var b strings.Builder
	b.WriteString(styleMuted().Render("tab field · type to edit · enter next/save · esc cancel"))
	b.WriteByte('\n')
	if m.err != "" {
		b.WriteString(styleWarn().Render(clipWidth(m.err, width)))
		b.WriteByte('\n')
	}
	labels := []string{"title", "artist", "year", "genre", "label"}
	for i, ti := range m.tagEditInputs() {
		active := m.tagEditFocusedIndex() == i
		if active {
			b.WriteString(styleSelected().Render(labels[i]))
		} else {
			b.WriteString(styleMuted().Render(labels[i]))
		}
		b.WriteByte('\n')
		b.WriteString("  ")
		b.WriteString(ti.View())
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) tagEditTarget() (library.Track, bool) {
	if m.focus != focusPlaylist {
		return library.Track{}, false
	}
	return m.selectedTrack()
}

func (m model) openTagEditor() (tea.Model, tea.Cmd) {
	t, ok := m.tagEditTarget()
	if !ok || t.Path == "" {
		return m, nil
	}
	if !m.tagEditor {
		m.tagEditSavedIdx = m.playlistIdx
		m.tagEditSavedOffset = m.playlistOffset
		m.tagEditSavedFocus = m.focus
		m.tagEditSaved = true
	}
	m.tagEditor = true
	m.tagEditPath = t.Path
	m.tagEditBusy = true
	m.tagEditFocus = 0
	m.tagEditTitle = newTagField()
	m.tagEditArtist = newTagField()
	m.tagEditYear = newTagField()
	m.tagEditGenre = newTagField()
	m.tagEditLabel = newTagField()
	m.err = ""
	m.focus = focusPlaylist
	m.search.Blur()
	m.settingsPath.Blur()
	m = m.blurTagEditFields()
	env := m.env
	path := t.Path
	return m, loadTagEditFields(env, path)
}

func (m model) restoreTagEditorCursor() model {
	m.tagEditor = false
	m.tagEditBusy = false
	m.tagEditPath = ""
	m = m.blurTagEditFields()
	if m.tagEditSaved {
		m.playlistIdx = m.tagEditSavedIdx
		m.playlistOffset = m.tagEditSavedOffset
		m.focus = m.tagEditSavedFocus
		m.tagEditSaved = false
	}
	return m
}

func (m model) closeTagEditor() (tea.Model, tea.Cmd) {
	m.err = ""
	m = m.restoreTagEditorCursor()
	return m, nil
}

func (m model) applyTagEditor() (tea.Model, tea.Cmd) {
	if m.tagEditPath == "" || m.tagEditBusy {
		return m, nil
	}
	tags := library.TrackTags{
		Title:  m.tagEditTitle.Value(),
		Artist: m.tagEditArtist.Value(),
		Year:   m.tagEditYear.Value(),
		Genre:  m.tagEditGenre.Value(),
		Label:  m.tagEditLabel.Value(),
	}
	m.tagEditBusy = true
	env := m.env
	path := m.tagEditPath
	return m, saveTrackTags(env, path, tags)
}

func (m model) tagEditAdvance(delta int) (model, tea.Cmd) {
	idx := m.tagEditFocusedIndex()
	if idx < 0 {
		idx = m.tagEditFocus
	}
	m.tagEditFocus = idx + delta
	if m.tagEditFocus >= len(m.tagEditInputs()) {
		return m, nil
	}
	if m.tagEditFocus < 0 {
		m.tagEditFocus = len(m.tagEditInputs()) - 1
	}
	return m.focusTagEditField()
}

func (m model) handleTagEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.tagEditBusy && m.tagEditTitle.Value() == "" && m.tagEditArtist.Value() == "" && msg.String() != "esc" {
		return m, nil
	}
	idx := m.tagEditFocusedIndex()
	if idx < 0 {
		switch msg.String() {
		case "esc":
			return m.closeTagEditor()
		case "tab", "enter", "down":
			m.tagEditFocus = 0
			return m.focusTagEditField()
		case "ctrl+c", "q":
			return m, m.quitCmd()
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		return m.closeTagEditor()
	case "enter":
		if idx >= len(m.tagEditInputs())-1 {
			return m.applyTagEditor()
		}
		return m.tagEditAdvance(1)
	case "tab", "down":
		if idx >= len(m.tagEditInputs())-1 {
			return m.applyTagEditor()
		}
		return m.tagEditAdvance(1)
	case "shift+tab", "up":
		return m.tagEditAdvance(-1)
	case "ctrl+c", "q":
		return m, m.quitCmd()
	}

	var cmd tea.Cmd
	switch idx {
	case 0:
		m.tagEditTitle, cmd = m.tagEditTitle.Update(msg)
	case 1:
		m.tagEditArtist, cmd = m.tagEditArtist.Update(msg)
	case 2:
		m.tagEditYear, cmd = m.tagEditYear.Update(msg)
	case 3:
		m.tagEditGenre, cmd = m.tagEditGenre.Update(msg)
	case 4:
		m.tagEditLabel, cmd = m.tagEditLabel.Update(msg)
	}
	return m, cmd
}
