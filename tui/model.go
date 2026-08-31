package tui

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sebday/evoplayer/server/art"
	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
)

type focus int

const (
	focusSearch focus = iota
	focusBrowse
	focusPlaylist
)

type model struct {
	env                paths.Env
	width              int
	height             int
	focus              focus
	search             textinput.Model
	nav                []navItem
	navIdx             int
	browseIdx          int
	browseOffset       int
	queue              []library.Track
	queueFiltered      []library.Track
	playlistIdx        int
	playlistOffset     int
	browsePath         string
	browseAll          []library.BrowseEntry
	browse             []library.BrowseEntry
	searchQuery        string
	searchGen          int
	searchMoved        bool
	status             playback.Status
	artPath            string
	artImg             image.Image
	art                *artCache
	artPicker          bool
	artHits            []art.Result
	artPickPath        string
	artPickQuery       string
	artPickBusy        bool
	artPickSavedIdx    int
	artPickSavedOffset int
	artPickSavedFocus  focus
	artPickSaved       bool
	artPickPreviewURL  string
	artPickPreviewIdx  int
	artPickPreviewGen  int
	artPreviewImg      image.Image
	artPreviewCache    map[string]image.Image
	movePicker         bool
	moveFolders        []string
	movePickPath       string
	movePickBusy       bool
	movePickSavedIdx   int
	movePickSavedOffset int
	movePickSavedFocus focus
	movePickSaved      bool
	frames             *frameCache
	wavePeaks          []int
	wavePath           string
	waveFile           string
	waveBusy           string
	vizSub             bool
	vizMode            vizMode
	paint              *vizPainter
	settingsPath       textinput.Model
	settings           config.JSONView
	job                jobs.State
	err                string
	loading            bool
	ready              bool
	pendingBrowse      int
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

type searchMsg struct {
	query  string
	gen    int
	tracks []library.Track
	err    error
}

type errMsg struct {
	err error
}

type tickMsg struct{}

type likeMsg struct {
	path  string
	liked bool
	err   error
}

type moveMsg struct {
	from   string
	to     string
	folder string
	err    error
}

type artSearchMsg struct {
	path    string
	query   string
	results []art.Result
	err     error
}

type artApplyMsg struct {
	path string
	art  string
	err  error
}

type artPreviewMsg struct {
	gen int
	idx int
	url string
	img image.Image
	err error
}

type artPrefetchMsg struct {
	path string
	gen  int
	urls []string
	imgs []image.Image
}

type jobMsg struct {
	state jobs.State
	err   error
}

func newModel(env paths.Env) model {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.Prompt = ""
	ti.CharLimit = 80
	ti.Width = 40
	sp := textinput.New()
	sp.Placeholder = "/path/to/music"
	sp.Prompt = "> "
	sp.CharLimit = 512
	sp.Width = 40
	frames := &frameCache{}
	return model{
		env:          env,
		focus:        focusBrowse,
		search:       ti,
		settingsPath: sp,
		width:        80,
		height:       24,
		art:          &artCache{},
		frames:       frames,
		paint:        newVizPainter(env, frames),
		ready:        true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadNav(m.env), loadBrowse(m.env, ""), loadTracks(m.env, "current"), pollLive(m.env), pollJob(m.env))
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

func (m model) refreshQueueUI() tea.Cmd {
	return tea.Batch(loadNav(m.env), loadTracks(m.env, "current"))
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
	if m.vizSub {
		m.vizSub = false
		return unsubViz(m.env)
	}
	return nil
}

func (m model) quitCmd() tea.Cmd {
	m.beginQuit()
	if m.vizSub {
		return tea.Batch(unsubViz(m.env), tea.Quit)
	}
	return tea.Quit
}

func (m *model) beginQuit() {
	m.clearStoredArtOverlay()
	if m.frames != nil {
		m.frames.mu.Lock()
		m.frames.quitting = true
		m.frames.mu.Unlock()
	}
	if m.art != nil {
		m.art.shown = false
	}
	if m.paint != nil {
		m.paint.setTransport(false, 0, 0, false)
	}
	_ = termimg.ClearAll()
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

func (m model) canPatchLists() bool {
	return !m.artPicker && !m.movePicker && !m.helpSelected() &&
		m.frames != nil && m.frames.view != "" && m.frames.browseRow > 0
}

func (m model) patchFocusedList() (tea.Model, tea.Cmd) {
	if !m.canPatchLists() {
		return m, nil
	}
	m.freezeFrame()
	if m.focus == focusPlaylist {
		m.patchPlaylist()
	} else {
		m.patchBrowse()
	}
	return m, nil
}

func (m model) canFreeze() bool {
	return !m.artPicker && !m.movePicker && m.frames != nil && m.frames.view != "" && m.frames.vizRow > 0
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg, vizSubMsg:
	default:
		m.unfreezeFrame()
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.search.Width = max(8, m.browseInnerWidth())
		m.settingsPath.Width = max(8, paneInnerWidth(m.playerGeom().playlistW)-4)
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
		cmds := []tea.Cmd{loadTracks(m.env, "current")}
		if len(m.browseAll) == 0 && m.browsePath == "" {
			m.loading = true
			cmds = append(cmds, loadBrowse(m.env, ""))
		}
		return m, tea.Batch(cmds...)
	case browseMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		pathChanged := msg.path != m.browsePath
		m.browsePath = msg.path
		m.browseAll = msg.entries
		if pathChanged {
			m.browseIdx = 0
			m.browseOffset = 0
			m.pendingBrowse = 0
		}
		cmd := m.applyFilter()
		if m.pendingBrowse != 0 {
			delta := m.pendingBrowse
			m.pendingBrowse = 0
			m.moveTree(delta)
		}
		return m, cmd
	case searchMsg:
		if msg.gen != m.searchGen || msg.query != m.searchQuery {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			m.browse = nil
			m.browseIdx = 0
			m.browseOffset = 0
			return m, nil
		}
		m.err = ""
		m.browse = tracksToBrowse(msg.tracks)
		m.browseIdx = 0
		m.browseOffset = 0
		return m, nil
	case tracksMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.loading = false
			return m, nil
		}
		if msg.name == "current" {
			m.queue = msg.tracks
			return m, m.applyFilter()
		}
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
				m.ensureWaveFill(vizWaveWidth(m.frames.vizW), vizPaintRows)
			}
		}
		if m.paint != nil && liveVizEnabled {
			m.paint.setTransport(m.playing(), m.status.Position, m.status.Duration, len(m.wavePeaks) > 0)
		}
		if m.canFreeze() {
			m.freezeFrame()
		}
		return m, nil
	case liveMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, tickStatus()
		}
		same := m.status.Path == msg.status.Path && m.status.QueueRevision == msg.status.QueueRevision
		oldPath := m.status.Path
		oldArt := m.artPath
		oldStatusArt := m.status.Art
		revChanged := m.status.QueueRevision != msg.status.QueueRevision
		m.status = msg.status
		m.refreshNowPlayingAssets()
		artChanged := oldStatusArt != m.status.Art || oldArt != m.artPath
		if m.focus == focusPlaylist && oldPath != m.status.Path {
			if m.scrollPlaylistForPlayingTrack() && m.canPatchLists() {
				m.patchPlaylist()
			}
		}
		if same && !artChanged && m.canFreeze() {
			m.freezeFrame()
			m.patchNowPlaying()
		}
		cmds := []tea.Cmd{tickStatus(), m.syncWaveform(), m.syncViz()}
		if revChanged {
			cmds = append(cmds, loadTracks(m.env, "current"))
		}
		return m, tea.Batch(cmds...)
	case jobMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			if m.job.Status == "running" {
				return m, pollJob(m.env)
			}
			return m, nil
		}
		wasScanning := m.libraryScanning()
		if msg.state.Status != "" || msg.state.Name != "" {
			m.job = mergeJobState(m.job, msg.state)
		}
		if m.job.Status == "idle" || (m.job.Status == "" && m.job.Name == "") {
			m.job = jobs.State{}
		}
		if wasScanning && !m.libraryScanning() {
			return m, m.refreshQueueUI()
		}
		if wasScanning != m.libraryScanning() {
			return m, nil
		}
		cmd := tea.Cmd(nil)
		if m.job.Status == "running" {
			cmd = pollJob(m.env)
		}
		if m.canFreeze() {
			m.freezeFrame()
		}
		return m, cmd
	case settingsMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.settings = msg.view
		if !m.settingsPath.Focused() {
			m.settingsPath.SetValue(strings.TrimSpace(msg.view.Paths.Root))
		}
		if m.settingsSelected() && m.canPatchLists() {
			m.freezeFrame()
			m.patchPlaylist()
			return m, nil
		}
		return m, nil
	case settingsSaveMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, m.settingsFocusCmd()
		}
		m.settings = msg.view
		m.env.MusicRoot = strings.TrimSpace(msg.view.Paths.Root)
		if !m.settingsPath.Focused() {
			m.settingsPath.SetValue(m.env.MusicRoot)
		}
		m.browsePath = ""
		m.browseAll = nil
		m.browse = nil
		m.loading = true
		return m, tea.Batch(loadBrowse(m.env, ""), m.settingsFocusCmd())
	case errMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		return m, nil
	case tickMsg:
		if m.canFreeze() {
			m.freezeFrame()
		}
		cmds := []tea.Cmd{pollLive(m.env)}
		if m.job.Status == "running" {
			cmds = append(cmds, pollJob(m.env))
		}
		return m, tea.Batch(cmds...)
	case vizSubMsg:
		if msg.err != nil {
			m.vizSub = false
			return m, nil
		}
		m.vizSub = msg.subscribed
		return m, nil
	case likeMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		for i := range m.browseAll {
			if m.browseAll[i].Path == msg.path {
				m.browseAll[i].Liked = msg.liked
			}
		}
		for i := range m.browse {
			if m.browse[i].Path == msg.path {
				m.browse[i].Liked = msg.liked
			}
		}
		for i := range m.queue {
			if m.queue[i].Path == msg.path {
				m.queue[i].Liked = msg.liked
			}
		}
		if m.status.Path == msg.path {
			m.status.Liked = msg.liked
			m.patchNowPlaying()
			if m.canPatchLists() {
				m.patchPlaylist()
			}
		}
		return m, m.applyFilter()
	case moveMsg:
		m.movePickBusy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.restoreMovePickerCursor()
		repath := func(path string) string {
			if path == msg.from {
				return msg.to
			}
			return path
		}
		for i := range m.browseAll {
			m.browseAll[i].Path = repath(m.browseAll[i].Path)
		}
		for i := range m.browse {
			m.browse[i].Path = repath(m.browse[i].Path)
			if m.browse[i].Type == "track" {
				m.browse[i].Track.Path = repath(m.browse[i].Track.Path)
				if m.browse[i].Track.Path == msg.to {
					m.browse[i].Track.Genre = msg.folder
				}
			}
		}
		for i := range m.queue {
			m.queue[i].Path = repath(m.queue[i].Path)
			if m.queue[i].Path == msg.to {
				m.queue[i].Genre = msg.folder
			}
		}
		if m.status.Path == msg.from {
			m.status.Path = msg.to
			m.status.Genre = msg.folder
			m.patchNowPlaying()
		}
		if m.artPath == msg.from {
			m.artPath = msg.to
		}
		if m.wavePath == msg.from {
			m.wavePath = msg.to
		}
		if m.canPatchLists() {
			m.patchPlaylist()
		}
		return m, tea.Batch(m.applyFilter(), loadBrowse(m.env, m.browsePath))
	case artSearchMsg:
		if !m.artPicker || msg.path != m.artPickPath {
			return m, nil
		}
		m.artPickBusy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.artHits = nil
			return m, nil
		}
		m.err = ""
		m.artPickQuery = msg.query
		m.artHits = msg.results
		m.playlistIdx = 0
		m.playlistOffset = 0
		if len(m.artHits) == 0 {
			m.artPickBusy = false
			return m, nil
		}
		m.artPickBusy = true
		m.artPickPreviewGen++
		return m, prefetchArtHits(m.artHits, m.artPickPath, m.artPickPreviewGen)
	case artPrefetchMsg:
		if !m.artPicker || msg.path != m.artPickPath || msg.gen != m.artPickPreviewGen {
			return m, nil
		}
		m.artPickBusy = false
		for i, img := range msg.imgs {
			if img == nil {
				continue
			}
			if i < len(msg.urls) {
				m.storeArtPreview(msg.urls[i], img)
			}
			if i < len(m.artHits) {
				m.storeArtPreview(art.PreviewURL(m.artHits[i]), img)
			}
		}
		if m.playlistIdx >= 0 && m.playlistIdx < len(msg.imgs) && msg.imgs[m.playlistIdx] != nil {
			m.artPreviewImg = msg.imgs[m.playlistIdx]
			if m.playlistIdx < len(msg.urls) {
				m.artPickPreviewURL = msg.urls[m.playlistIdx]
			}
			m.artPickPreviewIdx = m.playlistIdx
			m.refreshArtPickPreview()
		} else {
			m.showCachedArtPick(m.playlistIdx)
		}
		return m, nil
	case artPreviewMsg:
		if msg.err == nil && msg.url != "" && msg.img != nil {
			m.storeArtPreview(msg.url, msg.img)
		}
		if !m.artPicker || msg.gen != m.artPickPreviewGen || msg.idx != m.playlistIdx {
			return m, nil
		}
		if msg.err != nil {
			return m, nil
		}
		m.artPickPreviewURL = msg.url
		m.artPickPreviewIdx = msg.idx
		m.artPreviewImg = msg.img
		m.refreshArtPickPreview()
		return m, nil
	case artApplyMsg:
		m.artPickBusy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.restoreArtPickerCursor()
		if msg.art != "" {
			m.status.Art = msg.art
		}
		m.artPath = ""
		m.artImg = nil
		if m.art != nil {
			m.art.layout = ""
			m.art.seq = ""
			m.art.path = ""
		}
		m.refreshNowPlayingAssets()
		return m, pollLive(m.env)
	case tea.FocusMsg, tea.BlurMsg:
		return m, nil
	case tea.MouseMsg:
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	if m.focus == focusSearch {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, tea.Batch(cmd, m.applyFilter())
	}
	return m, nil
}

func (m model) browseFocused() bool {
	return m.focus == focusBrowse || m.focus == focusSearch
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusSearch {
		switch msg.String() {
		case "esc":
			m.search.Blur()
			m.search.SetValue("")
			m.searchMoved = false
			m.focus = focusBrowse
			return m, m.applyFilter()
		case "enter":
			m.searchMoved = false
			return m.playSelected()
		case "d":
			if m.searchMoved && m.browseLen() > 0 {
				m.searchMoved = false
				return m.addSelected()
			}
		case "up":
			if m.browseLen() == 0 || m.browseIdx <= 0 {
				return m, nil
			}
			m.searchMoved = true
			m.moveFocusedList(-1)
			return m.patchFocusedList()
		case "down":
			m.searchMoved = true
			m.moveFocusedList(1)
			return m.patchFocusedList()
		case "pgup":
			m.searchMoved = true
			m.moveFocusedList(-m.browseListVisible())
			return m.patchFocusedList()
		case "pgdown":
			m.searchMoved = true
			m.moveFocusedList(m.browseListVisible())
			return m.patchFocusedList()
		}
		m.searchMoved = false
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, tea.Batch(cmd, m.applyFilter())
	}

	if m.settingsPath.Focused() {
		switch msg.String() {
		case "esc":
			return m.closeOverlay()
		case "enter":
			return m.saveSettingsRoot()
		case "shift+tab", "up":
			m.settingsPath.Blur()
			m.focus = focusBrowse
			m.playlistIdx = 0
			return m, nil
		case "ctrl+c":
			return m, m.quitCmd()
		}
		var cmd tea.Cmd
		m.settingsPath, cmd = m.settingsPath.Update(msg)
		return m, cmd
	}

	if m.movePicker {
		switch msg.String() {
		case "esc":
			return m.closeMovePicker()
		case "enter":
			if m.movePickBusy {
				return m, nil
			}
			return m.applyMovePick()
		case "up":
			m.moveQueue(-1)
			return m, nil
		case "down":
			m.moveQueue(1)
			return m, nil
		case "ctrl+c", "q":
			return m, m.quitCmd()
		}
	}

	if m.artPicker {
		switch msg.String() {
		case "esc":
			return m.closeArtPicker()
		case "enter":
			if m.artPickBusy {
				return m, nil
			}
			return m.applyArtPick("album")
		case "s":
			if m.artPickBusy {
				return m, nil
			}
			return m.applyArtPick("track")
		case "a":
			return m.openArtPicker()
		case "up":
			m.moveQueue(-1)
			return m, m.armArtPickPreview()
		case "down":
			m.moveQueue(1)
			return m, m.armArtPickPreview()
		case "ctrl+c", "q":
			return m, m.quitCmd()
		}
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, m.quitCmd()
	case "esc":
		if m.movePicker {
			return m.closeMovePicker()
		}
		if m.artPicker {
			return m.closeArtPicker()
		}
		if m.helpSelected() || m.settingsSelected() {
			return m.closeOverlay()
		}
		return m, nil
	case "tab":
		cmd := m.cycleFocusDir(1)
		if cmd != nil {
			return m, cmd
		}
		return m, m.settingsFocusCmd()
	case "shift+tab":
		cmd := m.cycleFocusDir(-1)
		if cmd != nil {
			return m, cmd
		}
		return m, m.settingsFocusCmd()
	case "/":
		m.focus = focusSearch
		return m, m.search.Focus()
	case "?":
		if m.helpSelected() {
			return m.closeOverlay()
		}
		return m.openHelp()
	case "d":
		if m.helpSelected() && m.focus == focusPlaylist {
			return m, nil
		}
		return m.addDir()
	case "f":
		path := m.folderOpenTarget()
		if path == "" {
			return m, nil
		}
		target := path
		return m, func() tea.Msg {
			if err := openInFileManager(target); err != nil {
				return errMsg{err: err}
			}
			return tickMsg{}
		}
	case " ":
		return m, m.playbackCmd(togglePlayback)
	case ".":
		return m, m.playbackCmd(func(env paths.Env) error { return skipTrack(env, "playback.next") })
	case ",":
		return m, m.playbackCmd(func(env paths.Env) error { return skipTrack(env, "playback.prev") })
	case "<", "shift+,":
		return m, m.seekBy(-10)
	case ">", "shift+.":
		return m, m.seekBy(10)
	case "-", "_":
		return m, m.volumeDelta(-5)
	case "=":
		return m, m.volumeDelta(5)
	case "l":
		t, ok := m.likeTarget()
		if !ok || t.Path == "" {
			return m, nil
		}
		return m, m.likePathCmd(t.Path)
	case "m":
		if m.helpSelected() || m.settingsSelected() || m.artPicker || m.movePicker {
			return m, nil
		}
		if m.focus != focusPlaylist {
			return m, nil
		}
		return m.openMovePicker()
	case "L":
		t, ok := m.playingLikeTarget()
		if !ok || t.Path == "" {
			return m, nil
		}
		return m, m.likePathCmd(t.Path)
	case "v":
		return m.cycleViz(1)
	case "V":
		return m.cycleViz(-1)
	case "backspace":
		if m.canLeaveBrowseFolder() {
			return m.leaveFolder()
		}
		return m, nil
	case "left":
		if m.focus == focusPlaylist && m.settingsSelected() {
			m.settingsPath.Blur()
			m.cycleFocusDir(-1)
			return m, nil
		}
		if m.canLeaveBrowseFolder() {
			return m.leaveFolder()
		}
		return m, nil
	case "right":
		if m.focus == focusBrowse && !m.searching() {
			if _, ok := m.currentTool(); ok {
				return m, m.cycleFocusDir(1)
			}
			return m.enterFolder()
		}
		return m, nil
	case "enter":
		if m.helpSelected() && m.focus == focusPlaylist {
			return m, nil
		}
		if m.settingsSelected() && m.focus == focusPlaylist {
			if !m.settingsPath.Focused() {
				return m, m.settingsPath.Focus()
			}
			return m.saveSettingsRoot()
		}
		return m.playSelected()
	case "a":
		if m.helpSelected() && m.focus == focusPlaylist {
			return m, nil
		}
		if m.settingsSelected() {
			return m, nil
		}
		return m.openArtPicker()
	case "up", "down", "pgup", "pgdown":
		if msg.String() == "up" {
			if m.focus == focusBrowse && (m.browseLen() == 0 || m.browseIdx <= 0) {
				return m, nil
			}
			if m.focus == focusPlaylist && (m.playlistLen() == 0 || m.playlistIdx <= 0) {
				return m, nil
			}
			return m.moveFocusedWithTools(-1)
		}
		if msg.String() == "down" {
			return m.moveFocusedWithTools(1)
		}
		if msg.String() == "pgup" {
			return m.moveFocusedWithTools(-m.focusedVisible())
		}
		return m.moveFocusedWithTools(m.focusedVisible())
	}
	return m, nil
}

func (m *model) cycleFocusDir(dir int) tea.Cmd {
	m.search.Blur()
	m.settingsPath.Blur()
	order := []focus{focusBrowse, focusPlaylist}
	i := 0
	for j, f := range order {
		if m.focus == f {
			i = j
			break
		}
	}
	if m.focus == focusSearch {
		i = 0
	}
	n := len(order)
	i = (i + dir%n + n) % n
	m.focus = order[i]
	if m.focus == focusPlaylist && m.settingsSelected() {
		m.playlistIdx = 0
		return m.settingsPath.Focus()
	}
	if m.focus == focusPlaylist {
		m.focusPlaylistOnPlayingOrCaret()
	}
	m.refreshArt()
	return nil
}

func (m *model) focusPlaylistOnPlayingOrCaret() {
	if m.helpSelected() || m.settingsSelected() || m.artPicker || m.movePicker {
		return
	}
	path := strings.TrimSpace(m.status.Path)
	if path != "" {
		for i, t := range m.queueFiltered {
			if t.Path == path {
				m.playlistIdx = i
				m.ensurePlaylistVisible()
				return
			}
		}
	}
	n := m.playlistLen()
	if n == 0 {
		m.playlistIdx = 0
		m.playlistOffset = 0
		return
	}
	m.playlistIdx = clamp(m.playlistIdx, 0, n-1)
	m.ensurePlaylistVisible()
}

func (m *model) ensurePlaylistVisible() {
	n := m.playlistLen()
	if n == 0 {
		m.playlistOffset = 0
		return
	}
	vis := m.playlistListVisible()
	if m.playlistIdx < m.playlistOffset {
		m.playlistOffset = m.playlistIdx
	}
	if m.playlistIdx >= m.playlistOffset+vis {
		m.playlistOffset = m.playlistIdx - vis + 1
	}
	if m.playlistOffset < 0 {
		m.playlistOffset = 0
	}
}

func (m model) playingTrackIndex() int {
	path := strings.TrimSpace(m.status.Path)
	if path == "" {
		return -1
	}
	for i, t := range m.queueFiltered {
		if t.Path == path {
			return i
		}
	}
	return -1
}

func (m *model) scrollPlaylistForPlayingTrack() bool {
	if m.focus != focusPlaylist || m.helpSelected() || m.settingsSelected() || m.artPicker || m.movePicker {
		return false
	}
	idx := m.playingTrackIndex()
	if idx < 0 {
		return false
	}
	n := m.playlistLen()
	vis := m.playlistListVisible()
	if n == 0 || vis <= 0 {
		return false
	}
	before := m.playlistOffset
	if idx < m.playlistOffset {
		m.playlistOffset--
	} else if idx >= m.playlistOffset+vis {
		m.playlistOffset++
	}
	if m.playlistOffset < 0 {
		m.playlistOffset = 0
	}
	maxOffset := max(0, n-vis)
	if m.playlistOffset > maxOffset {
		m.playlistOffset = maxOffset
	}
	return m.playlistOffset != before
}

func (m *model) moveFocusedList(delta int) {
	if m.focus == focusPlaylist {
		m.moveQueue(delta)
		return
	}
	m.moveTree(delta)
}

func (m model) moveFocusedWithTools(delta int) (tea.Model, tea.Cmd) {
	prevSettings := m.settingsSelected()
	m.moveFocusedList(delta)
	cmd := m.syncBrowseTool()
	if m.focus == focusBrowse && (prevSettings || m.settingsSelected()) && m.canPatchLists() {
		m.freezeFrame()
		m.patchBrowse()
		m.patchPlaylist()
		return m, cmd
	}
	next, pcmd := m.patchFocusedList()
	return next, tea.Batch(cmd, pcmd)
}

func (m *model) syncBrowseTool() tea.Cmd {
	if !m.browseFocused() {
		return nil
	}
	was := m.currentNavID()
	if tool, ok := m.currentTool(); ok {
		if i := navIndex(m.nav, tool.ID); i >= 0 {
			m.navIdx = i
		}
		if tool.ID == "settings" && was != "settings" {
			return loadSettings(m.env)
		}
		return nil
	}
	if was == "settings" {
		m.settingsPath.Blur()
		if i := navIndex(m.nav, "filetree"); i >= 0 {
			m.navIdx = i
		}
	}
	return nil
}

func (m *model) selectSidebarTool(id string) {
	if i := navIndex(m.nav, id); i >= 0 {
		m.navIdx = i
	}
	n := len(m.browse) + m.playlistSlotCount()
	for i, t := range m.sidebarTools() {
		if t.ID != id {
			continue
		}
		m.browseIdx = n + i
		vis := m.browseListVisible()
		if m.browseIdx < m.browseOffset {
			m.browseOffset = m.browseIdx
		}
		if vis > 0 && m.browseIdx >= m.browseOffset+vis {
			m.browseOffset = m.browseIdx - vis + 1
		}
		return
	}
}

func (m *model) moveTree(delta int) {
	n := m.browseLen()
	if n == 0 {
		m.pendingBrowse += delta
		return
	}
	if m.pendingBrowse != 0 {
		delta += m.pendingBrowse
		m.pendingBrowse = 0
	}
	m.browseIdx = clamp(m.browseIdx+delta, 0, n-1)
	vis := m.browseListVisible()
	if m.browseIdx < m.browseOffset {
		m.browseOffset = m.browseIdx
	}
	if m.browseIdx >= m.browseOffset+vis {
		m.browseOffset = m.browseIdx - vis + 1
	}
}

func (m *model) moveQueue(delta int) {
	n := m.playlistLen()
	if n == 0 {
		return
	}
	m.playlistIdx = clamp(m.playlistIdx+delta, 0, n-1)
	m.ensurePlaylistVisible()
	if !m.artPicker && !m.canPatchLists() {
		m.refreshArt()
	}
}

func (m model) enterFolder() (tea.Model, tea.Cmd) {
	e, ok := m.currentBrowse()
	if !ok {
		return m, nil
	}
	switch e.Type {
	case "dir":
		m.loading = true
		return m, loadBrowse(m.env, e.Path)
	}
	return m, nil
}

func (m model) leaveFolder() (tea.Model, tea.Cmd) {
	if m.browsePath == "" {
		return m, nil
	}
	m.loading = true
	return m, loadBrowse(m.env, parentPath(m.browsePath))
}

func (m model) canLeaveBrowseFolder() bool {
	if m.focus != focusBrowse || m.browsePath == "" || m.searching() {
		return false
	}
	return len(m.browse) == 0 || m.browseIdx < len(m.browse)
}

func (m model) queueOp(fn func() error) (tea.Model, tea.Cmd) {
	return m, tea.Sequence(func() tea.Msg {
		if err := fn(); err != nil {
			return errMsg{err: err}
		}
		return tickMsg{}
	}, m.refreshQueueUI())
}

func (m model) queueSelectedFolder() (tea.Model, tea.Cmd) {
	e, ok := m.currentBrowse()
	if !ok {
		return m, nil
	}
	rel := m.browsePath
	switch e.Type {
	case "dir":
		rel = e.Path
	}
	if rel == "" {
		return m, nil
	}
	env := m.env
	return m.queueOp(func() error {
		return appendFolder(env, rel)
	})
}

func (m model) playSelectedFolder(rel string) (tea.Model, tea.Cmd) {
	if rel == "" {
		return m, nil
	}
	env := m.env
	return m.queueOp(func() error {
		paths, err := fetchBrowseQueue(env, rel)
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return fmt.Errorf("no tracks")
		}
		return playPaths(env, paths, paths[0])
	})
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

func seekTarget(pos, duration, delta float64) float64 {
	t := pos + delta
	if t < 0 {
		return 0
	}
	if duration > 0 && t > duration {
		return duration
	}
	return t
}

func (m model) seekBy(delta float64) tea.Cmd {
	if m.status.Path == "" {
		return nil
	}
	sec := seekTarget(m.status.Position, m.status.Duration, delta)
	return m.playbackCmd(func(env paths.Env) error { return seekPlayback(env, sec) })
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
	if m.browseFocused() {
		if item, ok := m.currentTool(); ok {
			return m.openTool(item.ID)
		}
		if item, ok := m.currentPlaylist(); ok {
			return m.playPlaylist(item.ID)
		}
		e, ok := m.currentBrowse()
		if !ok {
			return m, nil
		}
		switch e.Type {
		case "dir":
			return m.playSelectedFolder(e.Path)
		case "track":
			path := e.Track.Path
			tracks := []library.Track{e.Track}
			if m.searching() {
				tracks = browseTracks(m.browse)
			}
			env := m.env
			return m.queueOp(func() error {
				return playTracks(env, tracks, path)
			})
		}
		return m, nil
	}
	t, ok := m.selectedTrack()
	if !ok || t.Path == "" {
		return m, nil
	}
	tracks := m.queue
	path := t.Path
	env := m.env
	return m.queueOp(func() error {
		return playTracks(env, tracks, path)
	})
}

func (m model) playPlaylist(name string) (tea.Model, tea.Cmd) {
	if name == "" {
		return m, nil
	}
	env := m.env
	return m.queueOp(func() error {
		tracks, err := fetchTracks(env, name)
		if err != nil {
			return err
		}
		return playTracks(env, tracks, "")
	})
}

func (m model) addSelected() (tea.Model, tea.Cmd) {
	if m.browseFocused() {
		if item, ok := m.currentPlaylist(); ok {
			return m.addPlaylist(item.ID)
		}
		e, ok := m.currentBrowse()
		if !ok {
			return m, nil
		}
		switch e.Type {
		case "dir":
			return m.queueSelectedFolder()
		case "track":
			path := e.Track.Path
			env := m.env
			return m.queueOp(func() error {
				return appendPaths(env, []string{path})
			})
		}
		return m, nil
	}
	t, ok := m.selectedTrack()
	if !ok {
		return m, nil
	}
	path := t.Path
	env := m.env
	return m.queueOp(func() error {
		return appendPaths(env, []string{path})
	})
}

func (m model) addDir() (tea.Model, tea.Cmd) {
	if !m.browseFocused() {
		return m, nil
	}
	if _, ok := m.currentPlaylist(); ok {
		return m, nil
	}
	if _, ok := m.currentTool(); ok {
		return m, nil
	}
	return m.queueSelectedFolder()
}

func (m model) addPlaylist(name string) (tea.Model, tea.Cmd) {
	if name == "" {
		return m, nil
	}
	env := m.env
	return m.queueOp(func() error {
		tracks, err := fetchTracks(env, name)
		if err != nil {
			return err
		}
		pathsList := make([]string, 0, len(tracks))
		for _, t := range tracks {
			if t.Path != "" {
				pathsList = append(pathsList, t.Path)
			}
		}
		return appendPaths(env, pathsList)
	})
}

func (m model) searching() bool {
	return strings.TrimSpace(m.search.Value()) != ""
}

func (m *model) applyFilter() tea.Cmd {
	m.queueFiltered = m.queue
	m.refreshArt()
	q := strings.TrimSpace(m.search.Value())
	if q == "" {
		m.searchGen++
		m.searchQuery = ""
		m.browse = m.browseAll
		m.clampTreeCursor()
		return nil
	}
	if q == m.searchQuery {
		m.clampTreeCursor()
		return nil
	}
	m.searchQuery = q
	m.searchGen++
	gen := m.searchGen
	m.browse = nil
	m.browseIdx = 0
	m.browseOffset = 0
	env := m.env
	return func() tea.Msg {
		tracks, err := fetchSearch(env, q)
		return searchMsg{query: q, gen: gen, tracks: tracks, err: err}
	}
}

func (m *model) clampTreeCursor() {
	if n := m.browseLen(); m.browseIdx >= n {
		m.browseIdx = max(0, n-1)
	}
	if m.browseOffset > m.browseIdx {
		m.browseOffset = m.browseIdx
	}
	if n := m.playlistLen(); m.playlistIdx >= n {
		m.playlistIdx = max(0, n-1)
	}
	if m.playlistOffset > m.playlistIdx {
		m.playlistOffset = m.playlistIdx
	}
}

func (m model) currentNavID() string {
	if m.navIdx < 0 || m.navIdx >= len(m.nav) {
		return ""
	}
	return m.nav[m.navIdx].ID
}

func (m *model) refreshNowPlayingAssets() {
	m.refreshArt()
}

func (m model) artTargetPath() string {
	if m.status.Art != "" {
		return m.status.Art
	}
	if m.status.Path != "" {
		return library.ResolveArtPath(library.EnvFrom(m.env), m.status.Path)
	}
	return ""
}

func (m *model) refreshArt() {
	path := m.artTargetPath()
	if path != m.artPath || (m.artPath != "" && m.artImg == nil) {
		m.artPath = path
		m.artImg = loadArt(m.artPath)
		m.invalidateArtCache()
	}
}

func (m model) likeTarget() (library.Track, bool) {
	if m.focus == focusPlaylist {
		t, ok := m.selectedTrack()
		if ok && t.Path != "" && t.Path != m.status.Path {
			return t, true
		}
	}
	return m.playingLikeTarget()
}

func (m model) playingLikeTarget() (library.Track, bool) {
	if m.status.Path == "" {
		return library.Track{}, false
	}
	return library.Track{Path: m.status.Path, Liked: m.status.Liked}, true
}

func (m model) likePathCmd(path string) tea.Cmd {
	env := m.env
	return func() tea.Msg {
		res, err := toggleLike(env, path)
		return likeMsg{path: path, liked: res.Liked, err: err}
	}
}

func (m model) selectedTrack() (library.Track, bool) {
	if m.focus == focusPlaylist {
		if m.playlistIdx < 0 || m.playlistIdx >= len(m.queueFiltered) {
			return library.Track{}, false
		}
		return m.queueFiltered[m.playlistIdx], true
	}
	e, ok := m.currentBrowse()
	if !ok || e.Type != "track" {
		return library.Track{}, false
	}
	return e.Track, true
}

func (m model) currentBrowse() (library.BrowseEntry, bool) {
	if m.browseIdx < 0 || m.browseIdx >= len(m.browse) {
		return library.BrowseEntry{}, false
	}
	return m.browse[m.browseIdx], true
}

func (m model) currentPlaylist() (navItem, bool) {
	if m.playlistsPending() {
		return navItem{}, false
	}
	pls := m.sidebarPlaylists()
	i := m.browseIdx - len(m.browse)
	if i < 0 || i >= len(pls) {
		return navItem{}, false
	}
	return pls[i], true
}

func (m model) currentTool() (navItem, bool) {
	tools := m.sidebarTools()
	i := m.browseIdx - len(m.browse) - m.playlistSlotCount()
	if i < 0 || i >= len(tools) {
		return navItem{}, false
	}
	return tools[i], true
}

func (m model) browseLen() int {
	return len(m.browse) + m.playlistSlotCount() + len(m.sidebarTools())
}

func (m model) playlistLen() int {
	if m.artPicker {
		return len(m.artHits)
	}
	if m.movePicker {
		return len(m.moveFolders)
	}
	if m.helpSelected() || m.settingsSelected() {
		return 0
	}
	return len(m.queueFiltered)
}

func (m model) leftLoading() bool {
	return m.loading && !m.staticSelected()
}

func (m model) helpSelected() bool {
	return m.currentNavID() == "help"
}

func (m model) settingsSelected() bool {
	return m.currentNavID() == "settings"
}

func (m model) staticSelected() bool {
	return navIsStatic(m.currentNavID())
}

func (m model) openHelp() (tea.Model, tea.Cmd) {
	for i, item := range m.nav {
		if item.ID == "help" {
			m.navIdx = i
			m.focus = focusPlaylist
			m.search.Blur()
			m.settingsPath.Blur()
			return m, nil
		}
	}
	return m, nil
}

func (m model) openTool(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "settings":
		return m.openSettings()
	default:
		return m, nil
	}
}

func (m model) settingsFocusCmd() tea.Cmd {
	if m.settingsSelected() && m.focus == focusPlaylist {
		return m.settingsPath.Focus()
	}
	return nil
}

func (m model) saveSettingsRoot() (tea.Model, tea.Cmd) {
	path := strings.TrimSpace(m.settingsPath.Value())
	if path == "" {
		m.err = "library path required"
		return m, nil
	}
	m.settingsPath.Blur()
	return m, saveSettingsRoot(m.env, path)
}

func (m model) openSettings() (tea.Model, tea.Cmd) {
	m.selectSidebarTool("settings")
	m.focus = focusPlaylist
	m.search.Blur()
	root := strings.TrimSpace(m.settings.Paths.Root)
	if root == "" {
		root = strings.TrimSpace(m.env.MusicRoot)
	}
	m.settingsPath.SetValue(root)
	if m.canPatchLists() {
		m.freezeFrame()
		m.patchBrowse()
		m.patchPlaylist()
	}
	return m, tea.Batch(loadSettings(m.env), m.settingsFocusCmd())
}

func (m model) closeOverlay() (tea.Model, tea.Cmd) {
	m.settingsPath.Blur()
	m.clearArtPickPreview()
	m.artPicker = false
	m.artHits = nil
	m.artPickBusy = false
	m.restoreMovePickerCursor()
	if i := navIndex(m.nav, "filetree"); i >= 0 {
		m.navIdx = i
	}
	m.focus = focusBrowse
	return m, nil
}

func (m model) artPickTarget() string {
	return strings.TrimSpace(m.status.Path)
}

func (m model) openArtPicker() (tea.Model, tea.Cmd) {
	path := m.artPickTarget()
	if path == "" {
		return m, nil
	}
	if !m.artPicker {
		m.artPickSavedIdx = m.playlistIdx
		m.artPickSavedOffset = m.playlistOffset
		m.artPickSavedFocus = m.focus
		m.artPickSaved = true
	}
	m.artPicker = true
	m.artPickPath = path
	m.artPickQuery = ""
	m.artHits = nil
	m.artPickBusy = true
	m.clearArtPickPreview()
	m.err = ""
	m.focus = focusPlaylist
	m.playlistIdx = 0
	m.playlistOffset = 0
	m.search.Blur()
	m.settingsPath.Blur()
	env := m.env
	return m, func() tea.Msg {
		res, err := searchCover(env, path, "")
		return artSearchMsg{path: path, query: res.Query, results: res.Results, err: err}
	}
}

func (m *model) restoreArtPickerCursor() {
	m.clearArtPickPreview()
	m.artPicker = false
	m.artHits = nil
	m.artPickBusy = false
	if m.artPickSaved {
		m.playlistIdx = m.artPickSavedIdx
		m.playlistOffset = m.artPickSavedOffset
		m.focus = m.artPickSavedFocus
		m.artPickSaved = false
	}
}

func (m model) closeArtPicker() (tea.Model, tea.Cmd) {
	m.restoreArtPickerCursor()
	return m, nil
}

func (m *model) storeArtPreview(url string, img image.Image) {
	if url == "" || img == nil {
		return
	}
	if m.artPreviewCache == nil {
		m.artPreviewCache = map[string]image.Image{}
	}
	m.artPreviewCache[url] = img
}

func (m *model) cachedArtPreview(r art.Result) (image.Image, string, bool) {
	if m.artPreviewCache == nil {
		return nil, "", false
	}
	for _, url := range []string{art.PreviewURL(r), r.Thumb, r.URL} {
		if url == "" {
			continue
		}
		if img, ok := m.artPreviewCache[url]; ok && img != nil {
			return img, url, true
		}
	}
	return nil, "", false
}

func (m *model) showCachedArtPick(idx int) bool {
	if idx < 0 || idx >= len(m.artHits) {
		return false
	}
	img, url, ok := m.cachedArtPreview(m.artHits[idx])
	if !ok {
		return false
	}
	m.artPickPreviewURL = url
	m.artPickPreviewIdx = idx
	m.artPreviewImg = img
	m.refreshArtPickPreview()
	return true
}

func (m *model) clearArtPickPreview() {
	m.artPickPreviewURL = ""
	m.artPickPreviewIdx = -1
	m.artPickPreviewGen = 0
	m.artPreviewImg = nil
	m.artPreviewCache = nil
	m.invalidateArtCache()
}

func (m *model) armArtPickPreview() tea.Cmd {
	return m.previewArtPickCmd()
}

func fetchArtHit(r art.Result) (image.Image, string, error) {
	seen := map[string]struct{}{}
	var urls []string
	for _, u := range []string{art.PreviewURL(r), r.URL, r.Thumb} {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}
	if len(urls) == 0 {
		return nil, "", fmt.Errorf("no art url")
	}
	var lastErr error
	for _, url := range urls {
		img, err := fetchArtURL(url)
		if err == nil {
			return boundImage(img, art.PreviewSize), url, nil
		}
		lastErr = err
	}
	return nil, urls[0], lastErr
}

func prefetchArtHits(hits []art.Result, path string, gen int) tea.Cmd {
	hits = append([]art.Result(nil), hits...)
	return func() tea.Msg {
		urls := make([]string, len(hits))
		imgs := make([]image.Image, len(hits))
		sem := make(chan struct{}, 6)
		var wg sync.WaitGroup
		for i, hit := range hits {
			wg.Add(1)
			go func(i int, hit art.Result) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				img, url, err := fetchArtHit(hit)
				if err != nil {
					return
				}
				urls[i] = url
				imgs[i] = img
			}(i, hit)
		}
		wg.Wait()
		return artPrefetchMsg{path: path, gen: gen, urls: urls, imgs: imgs}
	}
}

func (m *model) previewArtPickCmd() tea.Cmd {
	if !m.artPicker || m.playlistIdx < 0 || m.playlistIdx >= len(m.artHits) {
		return nil
	}
	idx := m.playlistIdx
	if m.showCachedArtPick(idx) {
		return nil
	}
	hit := m.artHits[idx]
	m.artPickPreviewGen++
	gen := m.artPickPreviewGen
	return func() tea.Msg {
		img, url, err := fetchArtHit(hit)
		return artPreviewMsg{gen: gen, idx: idx, url: url, img: img, err: err}
	}
}

func (m *model) invalidateArtCache() {
	if m.art != nil {
		m.art.layout = ""
		m.art.seq = ""
		m.art.path = ""
	}
}

func (m *model) refreshArtPickPreview() {
	m.invalidateArtCache()
	if m.art != nil {
		m.art.shown = false
	}
	m.unfreezeFrame()
}

func artPickImageURL(r art.Result) string {
	return art.PreviewURL(r)
}

func (m model) artPickApplyURL() string {
	if m.playlistIdx < 0 || m.playlistIdx >= len(m.artHits) {
		return ""
	}
	return artPickImageURL(m.artHits[m.playlistIdx])
}

func (m model) applyArtPick(scope string) (tea.Model, tea.Cmd) {
	if m.playlistIdx < 0 || m.playlistIdx >= len(m.artHits) {
		return m, nil
	}
	path := m.artPickPath
	url := m.artPickApplyURL()
	if path == "" || url == "" {
		return m, nil
	}
	m.artPickBusy = true
	env := m.env
	return m, func() tea.Msg {
		out, err := applyCover(env, path, url, scope)
		return artApplyMsg{path: path, art: out.Art, err: err}
	}
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

func (m model) browseListVisible() int {
	g := m.playerGeom()
	return m.browseVisible(paneInnerHeight(g.bodyH))
}

func (m model) playlistListVisible() int {
	g := m.playerGeom()
	return paneInnerHeight(g.bodyH)
}

func (m model) focusedVisible() int {
	if m.focus == focusPlaylist {
		return m.playlistListVisible()
	}
	return m.browseListVisible()
}

func (m model) moveTarget() (library.Track, bool) {
	if m.focus != focusPlaylist {
		return library.Track{}, false
	}
	return m.selectedTrack()
}

func (m model) openMovePicker() (tea.Model, tea.Cmd) {
	t, ok := m.moveTarget()
	if !ok || t.Path == "" {
		return m, nil
	}
	if !m.movePicker {
		m.movePickSavedIdx = m.playlistIdx
		m.movePickSavedOffset = m.playlistOffset
		m.movePickSavedFocus = m.focus
		m.movePickSaved = true
	}
	m.movePicker = true
	m.movePickPath = t.Path
	m.movePickBusy = false
	m.moveFolders = library.GenreChoices(library.EnvFrom(m.env))
	m.err = ""
	m.focus = focusPlaylist
	m.playlistIdx = 0
	m.playlistOffset = 0
	m.search.Blur()
	m.settingsPath.Blur()
	return m, nil
}

func (m *model) restoreMovePickerCursor() {
	m.movePicker = false
	m.moveFolders = nil
	m.movePickBusy = false
	if m.movePickSaved {
		m.playlistIdx = m.movePickSavedIdx
		m.playlistOffset = m.movePickSavedOffset
		m.focus = m.movePickSavedFocus
		m.movePickSaved = false
	}
}

func (m model) closeMovePicker() (tea.Model, tea.Cmd) {
	m.err = ""
	m.restoreMovePickerCursor()
	return m, nil
}

func (m model) applyMovePick() (tea.Model, tea.Cmd) {
	if m.playlistIdx < 0 || m.playlistIdx >= len(m.moveFolders) {
		return m, nil
	}
	folder := m.moveFolders[m.playlistIdx]
	path := m.movePickPath
	if folder == "" || path == "" {
		return m, nil
	}
	m.movePickBusy = true
	env := m.env
	return m, func() tea.Msg {
		res, err := moveTrack(env, path, folder)
		return moveMsg{from: path, to: res.To, folder: folder, err: err}
	}
}
