package tui

import (
	"fmt"
	"image"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

type focus int

const (
	focusNav focus = iota
	focusSearch
	focusList
)

type model struct {
	env         paths.Env
	width       int
	height      int
	focus       focus
	search      textinput.Model
	nav         []navItem
	navIdx      int
	tracks      []library.Track
	filtered    []library.Track
	listIdx     int
	listOffset  int
	browsePath  string
	browseAll   []library.BrowseEntry
	browse      []library.BrowseEntry
	status      playback.Status
	artPath     string
	artImg      image.Image
	art         *artCache
	frames      *frameCache
	wavePeaks   []int
	wavePath    string
	waveFile    string
	waveBusy    string
	vizLevels   []float64
	vizPeaks    []float64
	vizHold     []int
	vizSub      bool
	vizMode     vizMode
	paint       *vizPainter
	upNext      []library.Track
	upNextRev   uint64
	upNextPath  string
	upNextOK    bool
	pulsePhase  float64
	chromeTick  bool
	downloadURL textinput.Model
	job         jobs.State
	err         string
	loading     bool
	ready       bool
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

type waveformMsg struct {
	path  string
	file  string
	peaks []int
	err   error
}

type vizSubMsg struct {
	subscribed bool
	err        error
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

type jobMsg struct {
	state jobs.State
	err   error
}

type upNextMsg struct {
	tracks []library.Track
	rev    uint64
	path   string
	err    error
}

func newModel(env paths.Env) model {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.Prompt = ""
	ti.CharLimit = 80
	ti.Width = 40
	dl := textinput.New()
	dl.Placeholder = ""
	dl.Prompt = ""
	dl.CharLimit = 512
	dl.Width = 40
	frames := &frameCache{}
	return model{
		env:         env,
		focus:       focusNav,
		search:      ti,
		downloadURL: dl,
		width:       80,
		height:      24,
		art:         &artCache{},
		frames:      frames,
		paint:       newVizPainter(env, frames),
		chromeTick:  true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadNav(m.env), pollLive(m.env), pollJob(m.env), tickLogo())
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
		return browseMsg{path: rel, entries: res.Entries}
	}
}

func tickLogo() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
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

func loadUpNext(env paths.Env, rev uint64, path string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := fetchUpNext(env, 8)
		return upNextMsg{tracks: tracks, rev: rev, path: path, err: err}
	}
}

func (m model) refreshQueueUI() tea.Cmd {
	env := m.env
	if m.currentNavID() == "current" {
		return tea.Batch(loadNav(env), loadTracks(env, "current"))
	}
	return loadNav(env)
}

func loadWaveform(env paths.Env, path, file string) tea.Cmd {
	return func() tea.Msg {
		peaks, used := readWaveformPeaks(file)
		if len(peaks) > 0 {
			return waveformMsg{path: path, file: used, peaks: peaks}
		}
		out, err := warmWaveform(env, path)
		if err != nil {
			return waveformMsg{path: path, err: err}
		}
		peaks, used = readWaveformPeaks(out)
		return waveformMsg{path: path, file: used, peaks: peaks}
	}
}

func (m model) artOverlaying() bool {
	return m.art != nil && m.art.overlay && m.art.seq != ""
}

func (m *model) continueChromeTick() tea.Cmd {
	if m.artOverlaying() || m.frameFrozen() {
		m.chromeTick = false
		return nil
	}
	return tickLogo()
}

func (m *model) ensureChromeTick() tea.Cmd {
	if m.chromeTick || m.artOverlaying() || m.frameFrozen() {
		return nil
	}
	m.chromeTick = true
	return tickLogo()
}

func (m *model) syncWaveform() tea.Cmd {
	path := m.status.Path
	if path == "" {
		m.wavePeaks = nil
		m.wavePath = ""
		m.waveFile = ""
		m.waveBusy = ""
		return nil
	}
	if path == m.wavePath {
		return nil
	}
	if m.waveBusy == path {
		return nil
	}
	m.wavePeaks = nil
	m.waveFile = ""
	m.waveBusy = path
	return loadWaveform(m.env, path, m.status.Waveform)
}

func (m model) playing() bool {
	return m.status.State == "playing"
}

func (m *model) syncViz() tea.Cmd {
	if !liveVizEnabled || m.vizMode == vizModeNone {
		if m.paint != nil {
			m.paint.setTransport(false, m.status.Position, m.status.Duration, false)
		}
		m.vizLevels = nil
		m.vizPeaks = nil
		m.vizHold = nil
		if m.vizSub {
			m.vizSub = false
			return unsubViz(m.env)
		}
		return nil
	}
	if m.paint != nil {
		m.paint.setMode(m.vizMode)
		m.paint.setTransport(m.playing(), m.status.Position, m.status.Duration, len(m.wavePeaks) > 0)
	}
	if m.playing() {
		if m.vizSub {
			return nil
		}
		m.vizSub = true
		return subscribeViz(m.env)
	}
	m.vizLevels = nil
	m.vizPeaks = nil
	m.vizHold = nil
	if m.vizSub {
		m.vizSub = false
		return unsubViz(m.env)
	}
	return nil
}

func (m model) quitCmd() tea.Cmd {
	_ = termimg.ClearAll()
	if m.paint != nil {
		m.paint.setTransport(false, 0, 0, false)
	}
	if m.vizSub {
		return tea.Batch(unsubViz(m.env), tea.Quit)
	}
	return tea.Quit
}

func (m *model) freezeFrame() {
	if m.frames != nil {
		m.frames.freeze = true
	}
}

func (m *model) unfreezeFrame() {
	if m.frames != nil {
		m.frames.freeze = false
	}
}

func (m model) canFreeze() bool {
	return m.frames != nil && m.frames.view != "" && m.frames.vizRow > 0
}

func (m model) frameFrozen() bool {
	return m.frames != nil && m.frames.freeze
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg, logoTickMsg, vizSubMsg:
	default:
		m.unfreezeFrame()
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.search.Width = max(8, m.searchBarWidth())
		m.downloadURL.Width = max(8, m.listInnerWidth()-4)
		if m.art != nil {
			m.art.cols = 0
		}
		return m, nil
	case navMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		prev := m.currentNavID()
		m.nav = msg.items
		m.navIdx = retainNavIdx(m.nav, prev)
		if len(m.nav) == 0 {
			return m, nil
		}
		id := m.nav[m.navIdx].ID
		if navIsStatic(id) {
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
	case waveformMsg:
		if msg.path != m.status.Path {
			return m, nil
		}
		m.waveBusy = ""
		m.wavePath = msg.path
		m.waveFile = msg.file
		m.wavePeaks = msg.peaks
		if m.frames != nil {
			m.frames.mu.Lock()
			m.frames.wavePath = ""
			m.frames.mu.Unlock()
			if m.frames.vizW >= 8 {
				m.ensureWaveFill(max(8, m.frames.vizW-2), vizPaintRows)
			}
		}
		if m.paint != nil && liveVizEnabled {
			m.paint.setTransport(m.playing(), m.status.Position, m.status.Duration, len(m.wavePeaks) > 0)
		}
		return m, nil
	case liveMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, tickStatus()
		}
		same := m.status.Path == msg.status.Path && m.status.QueueRevision == msg.status.QueueRevision
		oldArt := m.artPath
		timeChanged := m.status.PositionLabel != msg.status.PositionLabel || m.status.DurationLabel != msg.status.DurationLabel
		m.status = msg.status
		m.refreshNowPlayingAssets()
		if same && oldArt == m.artPath && m.canFreeze() && !timeChanged {
			m.freezeFrame()
		}
		cmds := []tea.Cmd{tickStatus(), m.ensureChromeTick(), m.syncWaveform(), m.syncViz()}
		if !m.upNextOK || m.upNextRev != msg.status.QueueRevision || m.upNextPath != msg.status.Path {
			cmds = append(cmds, loadUpNext(m.env, msg.status.QueueRevision, msg.status.Path))
		}
		return m, tea.Batch(cmds...)
	case upNextMsg:
		if msg.err != nil {
			m.upNext = nil
		} else {
			m.upNext = msg.tracks
		}
		m.upNextRev = msg.rev
		m.upNextPath = msg.path
		m.upNextOK = true
		return m, nil
	case jobMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		wasScanning := m.libraryScanning()
		m.job = msg.state
		if wasScanning && !m.libraryScanning() {
			return m, m.refreshQueueUI()
		}
		return m, nil
	case errMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		return m, nil
	case tickMsg:
		if m.canFreeze() {
			m.freezeFrame()
		}
		return m, tea.Batch(pollLive(m.env), pollJob(m.env))
	case logoTickMsg:
		m.pulsePhase += 0.05
		if m.pulsePhase >= 1 {
			m.pulsePhase -= 1
		}
		if m.canFreeze() {
			m.freezeFrame()
		}
		return m, m.continueChromeTick()
	case vizSubMsg:
		if msg.err != nil {
			m.vizSub = false
			return m, nil
		}
		m.vizSub = msg.subscribed
		if !msg.subscribed {
			m.vizLevels = nil
			m.vizPeaks = nil
			m.vizHold = nil
		}
		return m, nil
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
			if m.staticSelected() {
				return m, nil
			}
			return m.playSelected()
		case " ":
			return m, m.playbackCmd(togglePlayback)
		case "ctrl+c", "q":
			return m, m.quitCmd()
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.applyFilter()
		return m, cmd
	}

	if m.downloadURL.Focused() {
		switch msg.String() {
		case "esc":
			m.downloadURL.Blur()
			m.focus = focusNav
			m.listIdx = 0
			return m, nil
		case "enter":
			return m.startURLDownload()
		case "tab", "down":
			m.downloadURL.Blur()
			m.listIdx = dlCtrlDownload
			return m, nil
		case "shift+tab", "up":
			m.downloadURL.Blur()
			m.focus = focusNav
			m.listIdx = 0
			return m, nil
		case "ctrl+c":
			return m, m.quitCmd()
		}
		var cmd tea.Cmd
		m.downloadURL, cmd = m.downloadURL.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, m.quitCmd()
	case "tab":
		m.cycleFocus()
		return m, m.downloadFocusCmd()
	case "shift+tab":
		m.cycleFocus()
		return m, m.downloadFocusCmd()
	case "/":
		m.focus = focusSearch
		return m, m.search.Focus()
	case "?":
		return m.openHelp()
	case " ":
		return m, m.playbackCmd(togglePlayback)
	case ".":
		return m, m.playbackCmd(func(env paths.Env) error { return skipTrack(env, "playback.next") })
	case ",":
		return m, m.playbackCmd(func(env paths.Env) error { return skipTrack(env, "playback.prev") })
	case "-", "_":
		return m, m.volumeDelta(-5)
	case "=":
		return m, m.volumeDelta(5)
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
	case "v":
		return m.cycleViz(1)
	case "V":
		return m.cycleViz(-1)
	case "backspace":
		if m.focus == focusList && m.filetreeSelected() && m.browsePath != "" {
			return m.leaveFolder()
		}
		return m, nil
	case "left":
		if m.focus == focusNav {
			m.moveNav(-1)
			return m.selectNav()
		}
		if m.focus == focusList && m.filetreeSelected() && m.browsePath != "" {
			return m.leaveFolder()
		}
		return m, nil
	case "right":
		if m.focus == focusNav {
			m.moveNav(1)
			return m.selectNav()
		}
		if m.focus == focusList && m.filetreeSelected() {
			return m.enterFolder()
		}
		return m, nil
	case "enter":
		if m.focus == focusNav {
			return m.selectNav()
		}
		if m.downloadSelected() {
			return m.activateDownloadControl()
		}
		if m.staticSelected() {
			return m, nil
		}
		return m.playSelected()
	case "+":
		if m.staticSelected() {
			return m, nil
		}
		return m.addSelected()
	case "up", "k":
		if m.focus == focusNav {
			return m, nil
		}
		if m.downloadSelected() {
			return m.moveDownload(-1)
		}
		if m.focus == focusList && (m.listLen() == 0 || m.listIdx <= 0) {
			m.focus = focusNav
			return m, nil
		}
		m.moveList(-1)
		return m, nil
	case "down", "j":
		if m.focus == focusNav {
			m.focus = focusList
			if m.downloadSelected() {
				m.listIdx = dlCtrlURL
				return m, m.downloadURL.Focus()
			}
			return m, nil
		}
		if m.downloadSelected() {
			return m.moveDownload(1)
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

func (m *model) cycleFocus() {
	m.search.Blur()
	m.downloadURL.Blur()
	if m.focus == focusNav {
		m.focus = focusList
		if m.downloadSelected() {
			m.listIdx = dlCtrlURL
		}
		return
	}
	m.focus = focusNav
}

func (m *model) moveNav(delta int) {
	if len(m.nav) == 0 || delta == 0 {
		return
	}
	step := 1
	n := delta
	if delta < 0 {
		step = -1
		n = -delta
	}
	for i := 0; i < n; i++ {
		next := m.navIdx + step
		for next >= 0 && next < len(m.nav) && !navOnMenu(m.nav[next]) {
			next += step
		}
		if next < 0 || next >= len(m.nav) {
			return
		}
		m.navIdx = next
	}
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
	if id == "" || navIsStatic(id) {
		m.loading = false
		m.tracks = nil
		m.filtered = nil
		m.browse = nil
		m.browseAll = nil
		m.listIdx = 0
		m.listOffset = 0
		m.downloadURL.Blur()
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
	}, m.refreshQueueUI())
}

func (m model) playSelectedFolder(rel string) (tea.Model, tea.Cmd) {
	if rel == "" {
		return m, nil
	}
	env := m.env
	return m, tea.Batch(func() tea.Msg {
		paths, err := fetchBrowseQueue(env, rel)
		if err != nil {
			return errMsg{err: err}
		}
		if len(paths) == 0 {
			return errMsg{err: fmt.Errorf("no tracks")}
		}
		if err := playPaths(env, paths, paths[0]); err != nil {
			return errMsg{err: err}
		}
		return tickMsg{}
	}, m.refreshQueueUI())
}

func (m model) cycleViz(dir int) (tea.Model, tea.Cmd) {
	if dir < 0 {
		m.vizMode = m.vizMode.prev()
	} else {
		m.vizMode = m.vizMode.next()
	}
	if m.paint != nil {
		m.paint.setMode(m.vizMode)
	}
	return m, m.syncViz()
}

func (m model) volumeDelta(delta int) tea.Cmd {
	return m.playbackCmd(func(env paths.Env) error { return adjustVolume(env, delta) })
}

func (m model) playbackCmd(fn func(paths.Env) error) tea.Cmd {
	env := m.env
	return func() tea.Msg {
		if err := fn(env); err != nil {
			return errMsg{err: err}
		}
		return tickMsg{}
	}
}

func (m model) playSelected() (tea.Model, tea.Cmd) {
	if m.filetreeSelected() {
		e, ok := m.currentBrowse()
		if !ok {
			return m, nil
		}
		switch e.Type {
		case "parent":
			return m, nil
		case "dir":
			return m.playSelectedFolder(e.Path)
		case "track":
			path := e.Track.Path
			env := m.env
			return m, tea.Batch(func() tea.Msg {
				if err := playPaths(env, []string{path}, path); err != nil {
					return errMsg{err: err}
				}
				return tickMsg{}
			}, m.refreshQueueUI())
		}
		return m, nil
	}
	t, ok := m.selectedTrack()
	if !ok || t.Path == "" {
		return m, nil
	}
	tracks := m.tracks
	path := t.Path
	env := m.env
	return m, tea.Batch(func() tea.Msg {
		if err := playTracks(env, tracks, path); err != nil {
			return errMsg{err: err}
		}
		return tickMsg{}
	}, m.refreshQueueUI())
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
			return m, tea.Batch(func() tea.Msg {
				if err := appendPaths(env, []string{path}); err != nil {
					return errMsg{err: err}
				}
				return tickMsg{}
			}, m.refreshQueueUI())
		}
		return m, nil
	}
	t, ok := m.selectedTrack()
	if !ok {
		return m, nil
	}
	path := t.Path
	env := m.env
	return m, tea.Batch(func() tea.Msg {
		if err := appendPaths(env, []string{path}); err != nil {
			return errMsg{err: err}
		}
		return tickMsg{}
	}, m.refreshQueueUI())
}

func (m *model) applyFilter() {
	if m.downloadSelected() {
		return
	}
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
	if m.status.Art != m.artPath || (m.artPath != "" && m.artImg == nil) {
		m.artPath = m.status.Art
		m.artImg = loadArt(m.artPath)
		if m.art != nil {
			m.art.layout = ""
			m.art.seq = ""
			m.art.path = ""
		}
	}
}

func (m model) selectedTrack() (library.Track, bool) {
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
	if m.downloadSelected() {
		return dlCtrlCount
	}
	if m.filetreeSelected() {
		return len(m.browse)
	}
	return len(m.filtered)
}

func (m model) filetreeSelected() bool {
	return m.currentNavID() == "filetree"
}

func (m model) currentSelected() bool {
	return m.currentNavID() == "current"
}

func (m model) helpSelected() bool {
	return m.currentNavID() == "help"
}

func (m model) downloadSelected() bool {
	return m.currentNavID() == "download"
}

func (m model) staticSelected() bool {
	return navIsStatic(m.currentNavID())
}

func (m model) downloadFocusCmd() tea.Cmd {
	if m.downloadSelected() && m.focus == focusList && m.listIdx == dlCtrlURL {
		return m.downloadURL.Focus()
	}
	return nil
}

func (m model) moveDownload(delta int) (tea.Model, tea.Cmd) {
	next := m.listIdx + delta
	if next < 0 {
		m.downloadURL.Blur()
		m.focus = focusNav
		m.listIdx = 0
		return m, nil
	}
	if next >= dlCtrlCount {
		return m, nil
	}
	m.listIdx = next
	if m.listIdx == dlCtrlURL {
		return m, m.downloadURL.Focus()
	}
	m.downloadURL.Blur()
	return m, nil
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
	return 0, 0
}

func (m model) contentWidth() int {
	x, _ := m.framePad()
	return max(20, m.width-2*x)
}

func (m model) contentHeight() int {
	_, y := m.framePad()
	return max(10, m.height-2*y)
}

func (m model) mainWidth() int {
	return max(20, m.contentWidth())
}

func (m model) searchBarWidth() int {
	return min(32, max(18, m.contentWidth()/5))
}

func (m model) bodyRowHeight(headerH, footerH int) int {
	return max(5, m.contentHeight()-headerH-footerH)
}

func (m model) paneHeight() int {
	if !m.ready {
		return 10
	}
	headerH := lipgloss.Height(m.renderHeader())
	footerH := lipgloss.Height(m.renderFooter())
	return m.bodyRowHeight(headerH, footerH)
}

func (m model) listVisible() int {
	h := m.paneHeight()
	vis := h - 4
	if m.paneSubtitle() != "" {
		vis -= 2
	}
	if m.currentSelected() {
		vis -= m.nowPlayingHeadRows()
	}
	return max(1, vis)
}
