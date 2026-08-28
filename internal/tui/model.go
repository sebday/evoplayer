package tui

import (
	"image"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/playback"
)

type focus int

const (
	focusNav focus = iota
	focusSearch
	focusList
)

type model struct {
	env        paths.Env
	width      int
	height     int
	focus      focus
	search     textinput.Model
	nav        []navItem
	navIdx     int
	tracks     []library.Track
	filtered   []library.Track
	listIdx    int
	listOffset int
	browsePath string
	browseAll  []library.BrowseEntry
	browse     []library.BrowseEntry
	status     playback.Status
	wfPath     string
	wfData     []int
	artPath    string
	artImg     image.Image
	logoPhase  float64
	err        string
	loading    bool
	ready      bool
}

type navMsg struct {
	items []navItem
	err   error
}

type tracksMsg struct {
	name   string
	tracks []library.Track
	err    error
}

type liveMsg struct {
	status playback.Status
	err    error
}

type browseMsg struct {
	path    string
	entries []library.BrowseEntry
	err     error
}

type errMsg struct {
	err error
}

type tickMsg struct{}

type logoTickMsg struct{}

type likeMsg struct {
	path  string
	liked bool
	err   error
}

func newModel(env paths.Env) model {
	ti := textinput.New()
	ti.Placeholder = "find a track…"
	ti.Prompt = "> "
	ti.CharLimit = 80
	ti.Width = 40
	return model{
		env:    env,
		focus:  focusList,
		search: ti,
		width:  80,
		height: 24,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadNav(m.env), pollLive(m.env), tickLogo())
}

func loadNav(env paths.Env) tea.Cmd {
	return func() tea.Msg {
		items, err := fetchNav(env)
		return navMsg{items: items, err: err}
	}
}

func loadTracks(env paths.Env, name string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := fetchTracks(env, name)
		return tracksMsg{name: name, tracks: tracks, err: err}
	}
}

func loadBrowse(env paths.Env, rel string) tea.Cmd {
	return func() tea.Msg {
		res, err := fetchBrowse(env, rel)
		if err != nil {
			return browseMsg{path: rel, err: err}
		}
		entries := make([]library.BrowseEntry, 0, len(res.Entries)+1)
		if res.Parent != nil {
			entries = append(entries, library.BrowseEntry{
				Type:  "parent",
				Name:  "..",
				Track: library.Track{Path: *res.Parent},
			})
		} else if rel != "" {
			entries = append(entries, library.BrowseEntry{Type: "parent", Name: ".."})
		}
		entries = append(entries, res.Entries...)
		return browseMsg{path: rel, entries: entries}
	}
}

func tickLogo() tea.Cmd {
	return tea.Tick(70*time.Millisecond, func(time.Time) tea.Msg {
		return logoTickMsg{}
	})
}

func tickStatus() tea.Cmd {
	return tea.Tick(statusTick(), func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func pollLive(env paths.Env) tea.Cmd {
	return func() tea.Msg {
		st, err := fetchStatus(env)
		return liveMsg{status: st, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.search.Width = max(10, m.searchBarWidth()-8)
		return m, nil
	case navMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.nav = msg.items
		if m.navIdx >= len(m.nav) {
			m.navIdx = 0
		}
		if len(m.nav) == 0 {
			return m, nil
		}
		id := m.nav[m.navIdx].ID
		if id == "settings" || id == "nowplaying" || id == "help" {
			return m, nil
		}
		if id == "filetree" && (len(m.browseAll) > 0 || m.browsePath != "") {
			return m, nil
		}
		m.loading = true
		if id == "filetree" {
			return m, loadBrowse(m.env, "")
		}
		return m, loadTracks(m.env, id)
	case browseMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if !m.filetreeSelected() {
			return m, nil
		}
		m.err = ""
		m.browsePath = msg.path
		m.browseAll = msg.entries
		m.listIdx = 0
		m.listOffset = 0
		m.applyFilter()
		return m, nil
	case tracksMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if m.currentNavID() != msg.name {
			return m, nil
		}
		m.err = ""
		m.tracks = msg.tracks
		m.applyFilter()
		return m, nil
	case liveMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.status = msg.status
			m.refreshNowPlayingAssets()
		}
		return m, tickStatus()
	case errMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		return m, nil
	case tickMsg:
		return m, pollLive(m.env)
	case logoTickMsg:
		m.logoPhase += 0.03
		if m.logoPhase >= 1 {
			m.logoPhase -= 1
		}
		return m, tickLogo()
	case likeMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		for i := range m.tracks {
			if m.tracks[i].Path == msg.path {
				m.tracks[i].Liked = msg.liked
			}
		}
		for i := range m.browseAll {
			if m.browseAll[i].Path == msg.path {
				m.browseAll[i].Liked = msg.liked
			}
		}
		if m.status.Path == msg.path {
			m.status.Liked = msg.liked
		}
		m.applyFilter()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.focus == focusSearch {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.applyFilter()
		return m, cmd
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusSearch {
		switch msg.String() {
		case "esc":
			m.search.Blur()
			m.search.SetValue("")
			m.applyFilter()
			m.focus = focusList
			return m, nil
		case "tab":
			m.search.Blur()
			m.focus = focusList
			return m, nil
		case "enter":
			m.search.Blur()
			m.focus = focusList
			return m.addSelected()
		case " ":
			return m, func() tea.Msg {
				if err := togglePlayback(m.env); err != nil {
					return errMsg{err: err}
				}
				return tickMsg{}
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.applyFilter()
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		m.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil
	case "/":
		if m.helpSelected() {
			return m, nil
		}
		m.focus = focusSearch
		return m, m.search.Focus()
	case "?":
		return m.openHelp()
	case " ":
		return m, func() tea.Msg {
			if err := togglePlayback(m.env); err != nil {
				return errMsg{err: err}
			}
			return tickMsg{}
		}
	case ".":
		return m, func() tea.Msg {
			if err := skipTrack(m.env, "playback.next"); err != nil {
				return errMsg{err: err}
			}
			return tickMsg{}
		}
	case ",":
		return m, func() tea.Msg {
			if err := skipTrack(m.env, "playback.prev"); err != nil {
				return errMsg{err: err}
			}
			return tickMsg{}
		}
	case "l":
		t, ok := m.selectedTrack()
		if !ok {
			return m, nil
		}
		path := t.Path
		return m, func() tea.Msg {
			res, err := toggleLike(m.env, path)
			return likeMsg{path: path, liked: res.Liked, err: err}
		}
	case "backspace":
		if m.focus == focusList && m.filetreeSelected() && m.browsePath != "" {
			return m.leaveFolder()
		}
		return m, nil
	case "left":
		if m.focus == focusSearch {
			return m, nil
		}
		if m.focus == focusList && m.filetreeSelected() && m.browsePath != "" {
			return m.leaveFolder()
		}
		if m.focus == focusList {
			m.cycleFocus(-1)
		}
		return m, nil
	case "right":
		if m.focus == focusSearch {
			return m, nil
		}
		if m.focus == focusNav {
			m.cycleFocus(1)
			return m, nil
		}
		if m.focus == focusList && m.filetreeSelected() {
			return m.enterFolder()
		}
		return m, nil
	case "enter":
		if m.focus == focusNav {
			return m.selectNav()
		}
		if m.nowPlayingSelected() || m.helpSelected() {
			return m, nil
		}
		return m.addSelected()
	case "up", "k":
		if m.focus == focusNav {
			m.moveNav(-1)
			return m.selectNav()
		}
		m.moveList(-1)
		return m, nil
	case "down", "j":
		if m.focus == focusNav {
			m.moveNav(1)
			return m.selectNav()
		}
		m.moveList(1)
		return m, nil
	case "pgup":
		m.moveList(-m.listVisible())
		return m, nil
	case "pgdown":
		m.moveList(m.listVisible())
		return m, nil
	}
	return m, nil
}

func (m *model) cycleFocus(delta int) {
	m.search.Blur()
	if m.focus == focusSearch {
		m.focus = focusList
	}
	if m.nowPlayingSelected() || m.helpSelected() {
		if m.focus == focusNav {
			m.focus = focusList
		} else {
			m.focus = focusNav
		}
		return
	}
	if delta > 0 {
		if m.focus == focusNav {
			m.focus = focusList
		} else {
			m.focus = focusNav
		}
		return
	}
	if m.focus == focusList {
		m.focus = focusNav
	} else {
		m.focus = focusList
	}
}

func (m *model) moveNav(delta int) {
	if len(m.nav) == 0 {
		return
	}
	m.navIdx = clamp(m.navIdx+delta, 0, len(m.nav)-1)
}

func (m *model) moveList(delta int) {
	n := m.listLen()
	if n == 0 {
		return
	}
	m.listIdx = clamp(m.listIdx+delta, 0, n-1)
	vis := m.listVisible()
	if m.listIdx < m.listOffset {
		m.listOffset = m.listIdx
	}
	if m.listIdx >= m.listOffset+vis {
		m.listOffset = m.listIdx - vis + 1
	}
}

func (m model) selectNav() (tea.Model, tea.Cmd) {
	id := m.currentNavID()
	if id == "" || id == "settings" || id == "nowplaying" || id == "help" {
		m.loading = false
		m.tracks = nil
		m.filtered = nil
		m.browse = nil
		m.browseAll = nil
		return m, nil
	}
	m.loading = true
	m.listIdx = 0
	m.listOffset = 0
	if id == "filetree" {
		return m, loadBrowse(m.env, "")
	}
	return m, loadTracks(m.env, id)
}

func (m model) enterFolder() (tea.Model, tea.Cmd) {
	if !m.filetreeSelected() {
		return m, nil
	}
	e, ok := m.currentBrowse()
	if !ok {
		return m, nil
	}
	switch e.Type {
	case "parent":
		rel := e.Path
		if rel == "" {
			rel = parentPath(m.browsePath)
		}
		m.loading = true
		return m, loadBrowse(m.env, rel)
	case "dir":
		m.loading = true
		return m, loadBrowse(m.env, e.Path)
	}
	return m, nil
}

func (m model) leaveFolder() (tea.Model, tea.Cmd) {
	if !m.filetreeSelected() || m.browsePath == "" {
		return m, nil
	}
	m.loading = true
	return m, loadBrowse(m.env, parentPath(m.browsePath))
}

func (m model) queueSelectedFolder() (tea.Model, tea.Cmd) {
	if !m.filetreeSelected() {
		return m, nil
	}
	e, ok := m.currentBrowse()
	if !ok {
		return m, nil
	}
	rel := m.browsePath
	switch e.Type {
	case "parent":
		return m, nil
	case "dir":
		rel = e.Path
	}
	if rel == "" {
		return m, nil
	}
	env := m.env
	return m, tea.Batch(func() tea.Msg {
		if err := appendFolder(env, rel); err != nil {
			return errMsg{err: err}
		}
		return tickMsg{}
	}, loadNav(env))
}

func (m model) addSelected() (tea.Model, tea.Cmd) {
	if m.filetreeSelected() {
		e, ok := m.currentBrowse()
		if !ok {
			return m, nil
		}
		switch e.Type {
		case "parent":
			return m, nil
		case "dir":
			return m.queueSelectedFolder()
		case "track":
			path := e.Track.Path
			env := m.env
			return m, func() tea.Msg {
				if err := appendPaths(env, []string{path}); err != nil {
					return errMsg{err: err}
				}
				return tickMsg{}
			}
		}
		return m, nil
	}
	t, ok := m.selectedTrack()
	if !ok {
		return m, nil
	}
	path := t.Path
	env := m.env
	return m, func() tea.Msg {
		if err := appendPaths(env, []string{path}); err != nil {
			return errMsg{err: err}
		}
		return tickMsg{}
	}
}

func (m *model) applyFilter() {
	if m.filetreeSelected() {
		m.browse = filterBrowse(m.browseAll, m.search.Value())
	} else {
		m.filtered = filterTracks(m.tracks, m.search.Value())
	}
	n := m.listLen()
	if m.listIdx >= n {
		m.listIdx = max(0, n-1)
	}
	if m.listOffset > m.listIdx {
		m.listOffset = m.listIdx
	}
}

func (m model) currentNavID() string {
	if m.navIdx < 0 || m.navIdx >= len(m.nav) {
		return ""
	}
	return m.nav[m.navIdx].ID
}

func (m *model) refreshNowPlayingAssets() {
	if m.status.Waveform != m.wfPath || (m.wfPath != "" && m.wfData == nil) {
		m.wfPath = m.status.Waveform
		m.wfData = loadWaveformPeaks(m.wfPath)
	}
	if m.status.Art != m.artPath || (m.artPath != "" && m.artImg == nil) {
		m.artPath = m.status.Art
		m.artImg = loadArt(m.artPath)
	}
}

func (m model) selectedTrack() (library.Track, bool) {
	if m.nowPlayingSelected() {
		if m.status.Path == "" {
			return library.Track{}, false
		}
		return library.Track{
			Path:   m.status.Path,
			Title:  m.status.Title,
			Artist: m.status.Artist,
			Album:  m.status.Album,
			Genre:  m.status.Genre,
			Liked:  m.status.Liked,
		}, true
	}
	if m.filetreeSelected() {
		e, ok := m.currentBrowse()
		if !ok || e.Type != "track" {
			return library.Track{}, false
		}
		return e.Track, true
	}
	if m.listIdx < 0 || m.listIdx >= len(m.filtered) {
		return library.Track{}, false
	}
	return m.filtered[m.listIdx], true
}

func (m model) currentBrowse() (library.BrowseEntry, bool) {
	if m.listIdx < 0 || m.listIdx >= len(m.browse) {
		return library.BrowseEntry{}, false
	}
	return m.browse[m.listIdx], true
}

func (m model) listLen() int {
	if m.filetreeSelected() {
		return len(m.browse)
	}
	return len(m.filtered)
}

func (m model) filetreeSelected() bool {
	return m.currentNavID() == "filetree"
}

func (m model) nowPlayingSelected() bool {
	return m.currentNavID() == "nowplaying"
}

func (m model) helpSelected() bool {
	return m.currentNavID() == "help"
}

func (m model) openHelp() (tea.Model, tea.Cmd) {
	for i, item := range m.nav {
		if item.ID == "help" {
			m.navIdx = i
			m.focus = focusList
			m.search.Blur()
			return m.selectNav()
		}
	}
	return m, nil
}

func (m model) framePad() (x, y int) {
	if m.width < 96 || m.height < 28 {
		return 0, 0
	}
	return 6, 1
}

func (m model) contentWidth() int {
	x, _ := m.framePad()
	return max(20, m.width-2*x)
}

func (m model) contentHeight() int {
	_, y := m.framePad()
	return max(10, m.height-2*y)
}

func (m model) navWidth() int {
	return 22
}

func (m model) mainWidth() int {
	return max(20, m.contentWidth()-m.navWidth()-2)
}

func (m model) searchBarWidth() int {
	return max(16, m.contentWidth()-logoArtWidth()-headerLogoSearchGap)
}

func (m model) paneHeight() int {
	chrome := lipgloss.Height(m.renderHeader()) + lipgloss.Height(m.renderFooter()) + 5
	bodyH := max(8, m.contentHeight()-chrome)
	return max(5, bodyH)
}

func (m model) listVisible() int {
	vis := m.paneHeight() - 4
	if vis < 1 {
		return 1
	}
	return vis
}

func (m model) settingsSelected() bool {
	return m.currentNavID() == "settings"
}
