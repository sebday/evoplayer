package tui

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sebday/evoplayer/server/art"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/playlist"
)

func TestSearchQueriesFullLibrary(t *testing.T) {
	m := newModel(paths.Env{})
	m.nav = navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
		{Name: "mixes", Count: 1, Kind: "system"},
	})
	m.browseAll = []library.BrowseEntry{
		{Type: "dir", Name: "grime"},
		{Type: "dir", Name: "house"},
	}
	m.browse = m.browseAll
	m.queue = []library.Track{
		{Title: "World Series", Artist: "Sound Power"},
		{Title: "Fabriclive", Artist: "DJ Hype"},
	}
	m.search.SetValue("hype")
	cmd := m.applyFilter()
	if cmd == nil {
		t.Fatal("search should query the library")
	}
	if len(m.queueFiltered) != 2 {
		t.Fatalf("queue should stay unfiltered, got %d", len(m.queueFiltered))
	}
	if len(m.browse) != 0 {
		t.Fatalf("folder listing should clear while searching, got %#v", m.browse)
	}
	if n := len(m.sidebarPlaylists()); n != 0 {
		t.Fatalf("playlists should hide during library search, got %#v", m.sidebarPlaylists())
	}
	if n := len(m.sidebarTools()); n != 0 {
		t.Fatalf("settings and download should hide during library search, got %#v", m.sidebarTools())
	}
	m.search.SetValue("")
	if cmd = m.applyFilter(); cmd != nil {
		t.Fatal("clearing search should not query the library")
	}
	if len(m.browse) != 2 {
		t.Fatalf("cleared search should restore folder, got %#v", m.browse)
	}
}

func TestSearchIgnoresGlobalKeybinds(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusSearch
	m.search.Focus()

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = got.(model)
	if m.focus != focusSearch {
		t.Fatal("space should stay in search")
	}
	if cmd != nil {
		t.Fatal("space should not trigger playback")
	}

	got, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = got.(model)
	if m.focus != focusSearch {
		t.Fatal("q should stay in search")
	}
	if m.search.Value() != "q" {
		t.Fatalf("q should be typed into search, got %q", m.search.Value())
	}

	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = got.(model)
	if m.focus != focusBrowse {
		t.Fatal("esc should exit search")
	}
	if m.search.Value() != "" {
		t.Fatalf("esc should clear search, got %q", m.search.Value())
	}
}

func TestSearchArrowKeysMoveResults(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusSearch
	m.search.Focus()
	m.search.SetValue("wiley")
	m.searchQuery = "wiley"
	m.browse = []library.BrowseEntry{
		{Type: "track", Track: library.Track{Path: "/tmp/a.mp3", Title: "A"}},
		{Type: "track", Track: library.Track{Path: "/tmp/b.mp3", Title: "B"}},
	}
	m.browseIdx = 0

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = got.(model)
	if m.browseIdx != 1 {
		t.Fatalf("down in search should move browse caret, idx=%d", m.browseIdx)
	}
	if m.focus != focusSearch {
		t.Fatal("down should stay in search")
	}

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = got.(model)
	if m.focus != focusSearch {
		t.Fatal("d after moving selection should stay in search")
	}
	if cmd == nil {
		t.Fatal("d after moving selection should queue the selected search hit")
	}

	m.search.SetValue("wiley")
	m.searchQuery = "wiley"
	m.browseIdx = 0
	m.searchMoved = false
	got, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = got.(model)
	if m.focus != focusSearch {
		t.Fatal("enter should stay in search")
	}
	if cmd == nil {
		t.Fatal("enter should play the selected search hit")
	}
}

func TestSearchDTypingBeforeMove(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusSearch
	m.search.Focus()
	m.search.SetValue("wiley")

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = got.(model)
	if m.search.Value() != "wileyd" {
		t.Fatalf("d before moving selection should type, got %q", m.search.Value())
	}
	if cmd != nil && m.searchMoved {
		t.Fatal("d before moving selection should not queue")
	}
}

func TestSearchResultsReplaceFolder(t *testing.T) {
	m := newModel(paths.Env{})
	m.search.SetValue("wiley")
	m.searchQuery = "wiley"
	m.searchGen = 3
	got, _ := m.Update(searchMsg{
		query: "wiley",
		gen:   3,
		tracks: []library.Track{
			{Path: "/tmp/wot.mp3", Title: "Wot Do U Call It", Artist: "Wiley"},
			{Title: "missing path"},
		},
	})
	m = got.(model)
	if len(m.browse) != 1 || m.browse[0].Type != "track" || m.browse[0].Title != "Wot Do U Call It" {
		t.Fatalf("want one search hit, got %#v", m.browse)
	}
	got, _ = m.Update(searchMsg{
		query:  "wiley",
		gen:    2,
		tracks: []library.Track{{Path: "/tmp/old.mp3", Title: "stale"}},
	})
	m = got.(model)
	if m.browse[0].Title != "Wot Do U Call It" {
		t.Fatalf("stale search should be ignored, got %#v", m.browse)
	}
}

func TestSearchRowsShowArtist(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.search.SetValue("wiley")
	m.browse = []library.BrowseEntry{
		{Type: "track", Track: library.Track{Path: "/tmp/wot.mp3", Title: "Wot Do U Call It", Artist: "Wiley"}},
	}
	got := lipglossStrip(m.renderBrowseRows(40, 5))
	if !strings.Contains(got, "Wiley") || !strings.Contains(got, "Wot Do U Call It") {
		t.Fatalf("search hits should show artist and title, got %q", got)
	}
}

func TestNavFromIndex(t *testing.T) {
	got := navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
		{Name: "current", Count: 4, Kind: "system"},
		{Name: "mixes", Count: 1, Kind: "system"},
	})
	if got[0].ID != "filetree" || got[0].Label != "browse" {
		t.Fatalf("first should be browse, got %#v", got[0])
	}
	if got[1].ID != "settings" {
		t.Fatalf("second should be settings, got %#v", got[1])
	}
	if got[2].ID != "download" {
		t.Fatalf("third should be download, got %#v", got[2])
	}
	if got[3].ID != "help" {
		t.Fatalf("fourth should be help, got %#v", got[3])
	}
	if got[4].ID != "all" || got[4].Label != "likes" {
		t.Fatalf("all should display as likes, got %#v", got[4])
	}
	if got[5].ID != "current" || got[5].Label != "now playing" {
		t.Fatalf("current should display as now playing, got %#v", got[5])
	}
	if got[6].ID != "mixes" || got[6].Label != "mixes" {
		t.Fatalf("mixes should display as mixes, got %#v", got[6])
	}
	if retainNavIdx(got, "") != 0 {
		t.Fatalf("empty prev should land on filetree, got %d", retainNavIdx(got, ""))
	}
}

func TestArtTargetPathUsesNowPlaying(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.Path = "/playing.mp3"
	m.status.Art = "/art/playing.jpg"
	m.status.Year = "1994"
	m.focus = focusBrowse
	if got := m.artTargetPath(); got != "/art/playing.jpg" {
		t.Fatalf("should show playing art, got %q", got)
	}
	m.focus = focusPlaylist
	m.queueFiltered = []library.Track{
		{Path: "/playing.mp3", Art: "/art/playing.jpg"},
		{Path: "/queued.mp3", Art: "/art/queued.jpg", Year: "2001"},
	}
	m.playlistIdx = 1
	if got := m.artTargetPath(); got != "/art/playing.jpg" {
		t.Fatalf("playlist caret should not drive art path, got %q", got)
	}
	if got := m.artworkLegend(); got != "1994" {
		t.Fatalf("art legend should use now playing year, got %q", got)
	}
}

func TestLikeTargetsPlayingUnlessPlaylistCaret(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.Path = "/playing.mp3"
	m.focus = focusBrowse
	m.browse = []library.BrowseEntry{
		{Type: "track", Track: library.Track{Path: "/other.mp3"}},
	}
	m.browseIdx = 0
	got, ok := m.likeTarget()
	if !ok || got.Path != "/playing.mp3" {
		t.Fatalf("browse should like the playing track, got %#v ok=%v", got, ok)
	}
	m.focus = focusPlaylist
	m.queueFiltered = []library.Track{
		{Path: "/playing.mp3"},
		{Path: "/queued.mp3"},
	}
	m.playlistIdx = 0
	got, ok = m.likeTarget()
	if !ok || got.Path != "/playing.mp3" {
		t.Fatalf("playlist caret on playing track should like it, got %#v ok=%v", got, ok)
	}
	m.playlistIdx = 1
	got, ok = m.likeTarget()
	if !ok || got.Path != "/queued.mp3" {
		t.Fatalf("playlist caret on another track should like that track, got %#v ok=%v", got, ok)
	}
}

func TestUpperLikeAlwaysTargetsPlaying(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.Path = "/playing.mp3"
	m.focus = focusPlaylist
	m.queueFiltered = []library.Track{
		{Path: "/playing.mp3"},
		{Path: "/queued.mp3"},
	}
	m.playlistIdx = 1
	got, ok := m.playingLikeTarget()
	if !ok || got.Path != "/playing.mp3" {
		t.Fatalf("playing target = %#v ok=%v", got, ok)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	if cmd == nil {
		t.Fatal("L should toggle like for the playing track")
	}
	_ = next
}

func TestSidebarPlaylistsIncludeGenreDirs(t *testing.T) {
	m := newModel(paths.Env{})
	m.nav = navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
		{Name: "current", Count: 4, Kind: "system"},
		{Name: "mixes", Count: 1, Kind: "system"},
		{Name: "Drum & Bass", Count: 2, Kind: "fav"},
		{Name: "Grime", Count: 4, Kind: "fav"},
	})
	got := m.sidebarPlaylists()
	if len(got) != 5 {
		t.Fatalf("want now playing, likes, mixes, and genre playlists, got %#v", got)
	}
	if got[0].ID != "current" || got[0].Label != "now playing" || got[1].ID != "all" || got[2].ID != "mixes" || got[3].ID != "Drum & Bass" || got[4].ID != "Grime" {
		t.Fatalf("now playing should be first, got %#v", got)
	}
}

func TestNavFromIndexGenreLabels(t *testing.T) {
	cased := navFromIndex([]playlist.IndexItem{
		{Name: "Drum & Bass", Count: 2, Kind: "system"},
		{Name: "Grime", Count: 4, Kind: "system"},
	})
	if i := navIndex(cased, "Drum & Bass"); i < 0 || cased[i].Label != "Drum & Bass" {
		t.Fatalf("genre tabs should keep real casing, got %#v", cased)
	}
	if i := navIndex(cased, "Grime"); i < 0 || cased[i].Label != "Grime" {
		t.Fatalf("genre tabs should keep real casing, got %#v", cased)
	}
}

func TestQueuePlayingRowHighlight(t *testing.T) {
	m := newModel(paths.Env{})
	m.queueFiltered = []library.Track{
		{Path: "/tmp/x.mp3", Title: "World Series", Artist: "Sound Power", Duration: 201, Year: "2004"},
		{Path: "/tmp/y.mp3", Title: "Other", Artist: "Artist", Duration: 100, Year: "2005"},
	}
	m.status.Path = "/tmp/x.mp3"
	got := m.renderPlaylistTrackList(m.queueFiltered, 0, 0, 80, 5, true)
	if !strings.Contains(got, "42m") {
		t.Fatalf("playing queue row should use green background, got %q", got)
	}
	if !strings.Contains(got, "30") {
		t.Fatalf("playing queue row should invert text to terminal bg colour, got %q", got)
	}
	lines := strings.Split(got, "\n")
	if len(lines) < 2 || strings.Contains(lines[1], "42m") {
		t.Fatalf("only the playing row should be highlighted, got %q", got)
	}
	if lipgloss.Width(lines[0]) != 80 {
		t.Fatalf("playing row should fit one line at width 80, got %d: %q", lipgloss.Width(lines[0]), lipglossStrip(lines[0]))
	}
}

func TestBrowseReloadPreservesTreeIdx(t *testing.T) {
	m := newModel(paths.Env{})
	m.browseIdx = 1
	m.browseOffset = 0
	next, _ := m.Update(browseMsg{
		path: "",
		entries: []library.BrowseEntry{
			{Type: "dir", Name: "grime"},
			{Type: "dir", Name: "dubstep"},
		},
	})
	got := next.(model)
	if got.browseIdx != 1 || got.browseOffset != 0 {
		t.Fatalf("same-path browse reload should keep cursor, idx=%d offset=%d", got.browseIdx, got.browseOffset)
	}
	next, _ = got.Update(browseMsg{
		path: "grime",
		entries: []library.BrowseEntry{
			{Type: "dir", Name: "2008"},
		},
	})
	got = next.(model)
	if got.browseIdx != 0 || got.browseOffset != 0 {
		t.Fatalf("folder change should reset cursor, idx=%d offset=%d", got.browseIdx, got.browseOffset)
	}
}

func TestKeyBeforeBrowseLoadsMovesTree(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	key := tea.KeyMsg{Type: tea.KeyDown}
	if key.String() != "down" {
		t.Fatalf("key string %q", key.String())
	}
	if n := m.browseLen(); n != 3 {
		t.Fatalf("want loading playlist plus tools, len=%d", n)
	}
	next, _ := m.Update(key)
	got := next.(model)
	if got.browseIdx != 1 {
		t.Fatalf("down should move through the visible list, got %d", got.browseIdx)
	}
	next, _ = got.Update(browseMsg{path: "", entries: []library.BrowseEntry{
		{Type: "dir", Name: "grime"},
		{Type: "dir", Name: "dubstep"},
	}})
	got = next.(model)
	if got.browseIdx != 1 {
		t.Fatalf("caret should stay after browse loads, idx=%d", got.browseIdx)
	}
}

func TestKeyUnfreezesView(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	m.browse = []library.BrowseEntry{
		{Type: "dir", Name: "grime"},
		{Type: "dir", Name: "dubstep"},
	}
	m.browseAll = m.browse
	_ = m.View()
	got, _ := m.Update(liveMsg{status: m.status})
	m = got.(model)
	if !m.frames.freeze {
		t.Fatal("status poll should freeze the frame")
	}
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(model)
	if m.frames.freeze {
		t.Fatal("tab should unfreeze the frame")
	}
}

func TestArrowKeysPatchListsInPlace(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	m.browse = []library.BrowseEntry{
		{Type: "dir", Name: "grime"},
		{Type: "dir", Name: "dubstep"},
	}
	m.browseAll = m.browse
	m.focus = focusBrowse
	before := m.View()
	if m.frames.browseRow < 1 || m.frames.browseCol != paneContentCol() {
		t.Fatalf("browse patch origin row=%d col=%d", m.frames.browseRow, m.frames.browseCol)
	}
	if m.frames.playlistCol != m.playerGeom().browseW+paneContentCol() {
		t.Fatalf("playlist patch col=%d want %d", m.frames.playlistCol, m.playerGeom().browseW+paneContentCol())
	}
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = got.(model)
	if m.browseIdx != 1 {
		t.Fatalf("down should move browse, idx=%d", m.browseIdx)
	}
	if !m.frames.freeze {
		t.Fatal("arrow keys should freeze the frame and patch the list")
	}
	after := m.View()
	if before != after {
		t.Fatal("arrow keys should not rebuild View")
	}
}

func TestPlaylistArrowsKeepArtworkOverlay(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.focus = focusPlaylist
	m.status.Art = "/art/playing.jpg"
	m.queueFiltered = []library.Track{
		{Path: "/tmp/a.mp3", Title: "A", Art: "/art/a.jpg"},
		{Path: "/tmp/b.mp3", Title: "B", Art: "/art/b.jpg"},
	}
	m.playlistIdx = 0
	_ = m.View()
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = got.(model)
	if m.playlistIdx != 1 {
		t.Fatalf("down should move playlist, idx=%d", m.playlistIdx)
	}
	if !m.frames.freeze {
		t.Fatal("playlist arrows should freeze and leave artwork overlay")
	}
	if m.artPath != "" && m.artPath == "/art/b.jpg" {
		t.Fatal("playlist arrows should not swap the artwork overlay")
	}
}

func TestSettingsKeepsArtworkOverlay(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.focus = focusBrowse
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "grime"}}
	m.browseIdx = len(m.browse) + m.playlistSlotCount() - 1
	m.status.Art = "/art/playing.jpg"
	before := m.View()
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = got.(model)
	if !m.settingsSelected() {
		t.Fatalf("down should land on settings, nav=%s idx=%d", m.currentNavID(), m.browseIdx)
	}
	if !m.frames.freeze {
		t.Fatal("settings should patch like playlist and leave the artwork overlay")
	}
	if m.View() != before {
		t.Fatal("settings should not rebuild View")
	}
}

func TestPlaylistHeartSitsBeforeTime(t *testing.T) {
	m := newModel(paths.Env{})
	m.queueFiltered = []library.Track{
		{Path: "/tmp/y.mp3", Title: "Other", Artist: "Artist", Duration: 100, Year: "2005", Liked: true},
	}
	got := lipglossStrip(m.renderPlaylistTrackList(m.queueFiltered, 0, 0, 80, 5, true))
	heart := strings.Index(got, "♥")
	timeIdx := strings.Index(got, "1:40")
	title := strings.Index(got, "Other")
	if heart < 0 || timeIdx < 0 || title < 0 {
		t.Fatalf("want title, heart, and time, got %q", got)
	}
	if !(title < heart && heart < timeIdx) {
		t.Fatalf("heart should sit after the title and before time, got %q", got)
	}
}

func TestQueueSelectedLikedHeartMatchesCaret(t *testing.T) {
	m := newModel(paths.Env{})
	m.queueFiltered = []library.Track{
		{Path: "/tmp/y.mp3", Title: "Other", Artist: "Artist", Duration: 100, Year: "2005", Liked: true},
	}
	got := m.renderPlaylistTrackList(m.queueFiltered, 0, 0, 80, 5, true)
	if !strings.Contains(got, "34m") {
		t.Fatalf("selected liked heart should use accent colour, got %q", got)
	}
	if strings.Contains(got, "32m♥") {
		t.Fatalf("selected liked heart should not stay green, got %q", got)
	}
}

func TestQueueTrackListOmitsGenre(t *testing.T) {
	m := newModel(paths.Env{})
	m.queueFiltered = []library.Track{
		{Path: "/tmp/x.mp3", Title: "World Series", Artist: "Sound Power", Duration: 201, Year: "2004", Genre: "Grime"},
	}
	got := lipglossStrip(m.renderPlaylistTrackList(m.queueFiltered, 0, 0, 80, 5, true))
	if strings.Contains(got, "Grime") {
		t.Fatalf("queue rows should not show genre, got %q", got)
	}
	if !strings.Contains(got, "2004") {
		t.Fatalf("queue rows should still show year, got %q", got)
	}
}

func TestFiletreeOmitsParentRow(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "filetree")
	m.browsePath = "grime"
	m.browseAll = []library.BrowseEntry{
		{Type: "dir", Name: "2008"},
		{Type: "track", Track: library.Track{Title: "Wot Do U Call It", Artist: "Wiley"}},
	}
	m.browse = m.browseAll
	for _, e := range m.browse {
		if e.Type == "parent" {
			t.Fatal("filetree should not include a parent row")
		}
	}
	got := lipglossStrip(m.renderBrowseRows(m.browseInnerWidth(), 10))
	if !strings.Contains(got, "2008/") {
		t.Fatalf("want folder row, got %q", got)
	}
}

func TestBrowseTrackRowsShowNameOnly(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "filetree")
	m.browsePath = "grime"
	m.browse = []library.BrowseEntry{
		{Type: "track", Track: library.Track{Title: "Wot Do U Call It", Artist: "Wiley", Duration: 201, Year: "2004", Genre: "Grime"}},
	}
	m.browseIdx = 0
	m.focus = focusBrowse
	got := lipglossStrip(m.renderBrowseRows(m.browseInnerWidth(), 5))
	if !strings.Contains(got, "Wot Do U Call It") {
		t.Fatalf("want track title, got %q", got)
	}
	if strings.Contains(got, "Wiley") || strings.Contains(got, "3:21") || strings.Contains(got, "2004") {
		t.Fatalf("browse tracks should not show artist, time, or year, got %q", got)
	}
	if strings.HasPrefix(strings.TrimLeft(got, "\n"), " ") {
		t.Fatalf("selected browse track should not pad before the caret, got %q", got)
	}
	if !strings.Contains(got, ">") {
		t.Fatalf("want caret, got %q", got)
	}
}

func TestBrowseLongNameFillsPane(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "filetree")
	m.browsePath = "drum&bass/vinyl/back2basics"
	m.browse = []library.BrowseEntry{
		{Type: "track", Track: library.Track{Title: "b2b12007r-dj_taktix_very_long_filename_here"}},
	}
	m.browseAll = m.browse
	g := m.playerGeom()
	var row string
	for _, line := range strings.Split(lipglossStrip(m.renderBrowsePane(g)), "\n") {
		if strings.Contains(line, "b2b12007r") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("missing browse name row")
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(row, "│"), "│")
	gap := lipgloss.Width(inner) - lipgloss.Width(strings.TrimRight(inner, " "))
	if gap > 1 {
		t.Fatalf("browse name left %d empty cols: %q", gap, inner)
	}
}

func TestBrowsePanelIsNarrowerAndShowsFolderTick(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "filetree")
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "albums"}}
	g := m.playerGeom()
	if g.browseW != 27 {
		t.Fatalf("browse panel width=%d want 27", g.browseW)
	}
	root := lipglossStrip(m.renderBrowsePane(g))
	if strings.Contains(root, "┬") {
		t.Fatalf("root browse should not show a folder tick, got %q", root)
	}
	m.browsePath = "drum&bass"
	in := lipglossStrip(m.renderBrowsePane(g))
	if !strings.Contains(in, "┬") {
		t.Fatalf("in-folder browse should tick the top line, got %q", in)
	}
}

func TestBrowseAndPlaylistPadSidesNotTop(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "albums"}}
	m.queueFiltered = []library.Track{{Title: "World Series", Artist: "Sound Power"}}
	g := m.playerGeom()
	g.bodyH = 12
	assertPane := func(t *testing.T, pane, want string) {
		t.Helper()
		lines := strings.Split(lipglossStrip(pane), "\n")
		if len(lines) < 5 {
			t.Fatalf("pane too short: %q", pane)
		}
		topPad := strings.TrimSuffix(strings.TrimPrefix(lines[1], "│"), "│")
		if strings.TrimSpace(topPad) != "" {
			t.Fatalf("first inner row should be empty pad, got %q", lines[1])
		}
		content := strings.TrimSuffix(strings.TrimPrefix(lines[2], "│"), "│")
		if !strings.Contains(content, want) {
			t.Fatalf("content should start after top pad, want %q got %q", want, lines[2])
		}
		if !strings.HasPrefix(content, " ") || !strings.HasSuffix(content, " ") {
			t.Fatalf("want 1-col left/right pad, got %q", content)
		}
		bottomPad := strings.TrimSuffix(strings.TrimPrefix(lines[len(lines)-2], "│"), "│")
		if strings.TrimSpace(bottomPad) != "" {
			t.Fatalf("last inner row should be empty pad, got %q", lines[len(lines)-2])
		}
	}
	assertPane(t, m.renderBrowsePane(g), "albums")
	assertPane(t, m.renderPlaylistPane(g), "Sound")
}

func TestBrowseDirCountIsRightAligned(t *testing.T) {
	m := newModel(paths.Env{})
	m.browse = []library.BrowseEntry{
		{Type: "dir", Name: "drum&bass", Count: 6813},
		{Type: "dir", Name: "house", Count: 7},
	}
	const width = 24
	bass := lipglossStrip(m.renderBrowseRow(0, width, false))
	house := lipglossStrip(m.renderBrowseRow(1, width, false))
	if strings.Contains(bass, "drum&bass/ 6813") {
		t.Fatalf("count should not sit next to the name, got %q", bass)
	}
	if !strings.HasSuffix(bass, "6813") || !strings.HasSuffix(house, "7") {
		t.Fatalf("counts should sit on the right edge, got %q and %q", bass, house)
	}
	if lipgloss.Width(bass) != width || lipgloss.Width(house) != width {
		t.Fatalf("rows should fill the pane, got %d and %d", lipgloss.Width(bass), lipgloss.Width(house))
	}
}

func TestBrowseInFolderHasNoSectionRulesOrGap(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
		{Name: "mixes", Count: 1, Kind: "system"},
	})
	m.navIdx = navIndex(m.nav, "filetree")
	m.browsePath = "drum&bass"
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "albums"}}
	m.browseIdx = 0
	got := lipglossStrip(m.renderBrowseInner(m.browseInnerWidth(), 12))
	if strings.Contains(got, "├─") || strings.Contains(got, "─ drum&bass") || strings.Contains(got, "─ playlists") {
		t.Fatalf("in-folder browse should not draw section rules, got %q", got)
	}
	if !strings.Contains(got, "drum&bass") || !strings.Contains(got, "albums/") || !strings.Contains(got, "likes") {
		t.Fatalf("want path, folders, then playlists, got %q", got)
	}
	lines := strings.Split(got, "\n")
	pathIdx, likesIdx, folderIdx := -1, -1, -1
	for i, line := range lines {
		if pathIdx < 0 && strings.Contains(line, "drum&bass") && !strings.Contains(line, "albums") {
			pathIdx = i
		}
		if strings.Contains(line, "albums/") {
			folderIdx = i
		}
		if likesIdx < 0 && strings.Contains(line, "likes") {
			likesIdx = i
		}
	}
	if pathIdx < 0 || likesIdx < 0 || folderIdx < 0 {
		t.Fatalf("missing path, playlist, or folder rows, got %q", got)
	}
	if !(pathIdx < folderIdx && folderIdx < likesIdx) {
		t.Fatalf("want path, folders, then playlists, got %q", got)
	}
}

func TestBrowseSelectsOneRow(t *testing.T) {
	m := newModel(paths.Env{})
	m.focus = focusBrowse
	m.nav = navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
		{Name: "current", Count: 4, Kind: "system"},
		{Name: "mixes", Count: 1, Kind: "system"},
	})
	m.browse = []library.BrowseEntry{
		{Type: "dir", Name: "albums"},
		{Type: "dir", Name: "mixes"},
		{Type: "dir", Name: "grime"},
	}
	m.browseIdx = len(m.browse) + 2 // now playing, likes, mixes
	got := lipglossStrip(m.renderBrowseInner(40, 12))
	if strings.Count(got, ">") != 1 {
		t.Fatalf("want one caret, got %q", got)
	}
	if !strings.Contains(got, ">mixes") {
		t.Fatalf("caret should sit on the playlist row, got %q", got)
	}
	if strings.Contains(got, ">grime") {
		t.Fatalf("folder row should not share the playlist caret, got %q", got)
	}
}

func TestLeftLeavesEmptyBrowseFolder(t *testing.T) {
	m := newModel(paths.Env{})
	m.focus = focusBrowse
	m.browsePath = "drum&bass/youtube"
	m.browseIdx = 0
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = got.(model)
	if cmd == nil {
		t.Fatal("left should leave an empty browse folder")
	}
	if !m.loading {
		t.Fatal("left should queue a browse reload")
	}
}

func TestLeftStaysOnPlaylistRow(t *testing.T) {
	m := newModel(paths.Env{})
	m.focus = focusBrowse
	m.browsePath = "grime"
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "albums"}}
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 3, Kind: "system"}})
	m.browseIdx = len(m.browse)
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd != nil {
		t.Fatalf("left on playlist row should not leave folder, got cmd=%#v", cmd)
	}
	if got.(model).browsePath != "grime" {
		t.Fatalf("browse path should stay, got %q", got.(model).browsePath)
	}
}

func TestBrowsePlaylistsHaveSectionRule(t *testing.T) {
	m := newModel(paths.Env{})
	m.focus = focusBrowse
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 3, Kind: "system"}})
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "grime"}}
	got := lipglossStrip(m.renderBrowseInner(30, 12))
	grimeIdx := strings.Index(got, "grime")
	likesIdx := strings.Index(got, "likes")
	if grimeIdx < 0 || likesIdx < 0 {
		t.Fatalf("missing browse or playlist rows, got %q", got)
	}
	between := got[grimeIdx:likesIdx]
	if strings.Contains(between, "╭") || strings.Contains(between, "╰") {
		t.Fatalf("playlists should not use a nested fieldset, got %q", between)
	}
	if !strings.Contains(between, "─") {
		t.Fatalf("playlists should be separated by a horizontal rule, got %q", between)
	}
}

func TestCompleteTracksPageLoadsRemainder(t *testing.T) {
	first := playlist.TracksPage{Total: 450, Items: make([]library.Track, 400)}
	got, err := completeTracksPage(first, func(total int) (playlist.TracksPage, error) {
		if total != 450 {
			t.Fatalf("reload limit = %d", total)
		}
		return playlist.TracksPage{Total: 450, Items: make([]library.Track, 450)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 450 {
		t.Fatalf("want full playlist, got %d", len(got))
	}
	small := playlist.TracksPage{Total: 3, Items: make([]library.Track, 3)}
	got, err = completeTracksPage(small, func(int) (playlist.TracksPage, error) {
		t.Fatal("should not reload a complete first page")
		return playlist.TracksPage{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want first page, got %d", len(got))
	}
}

func TestQueueLegendShowsFullCount(t *testing.T) {
	m := newModel(paths.Env{})
	m.queueFiltered = make([]library.Track, 450)
	got := m.playlistLegend(40)
	if got != "playlist (450)" {
		t.Fatalf("got %q", got)
	}
}

func TestLiveReloadsPlaylistOnQueueRevision(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.QueueRevision = 1
	got, cmd := m.Update(liveMsg{status: playback.Status{QueueRevision: 2}})
	if cmd == nil {
		t.Fatal("queue revision change should reload playlist")
	}
	next := got.(model)
	if next.status.QueueRevision != 2 {
		t.Fatalf("revision = %d", next.status.QueueRevision)
	}
}

func TestViewFreezeKeepsCachedFrame(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	first := m.View()
	if m.frames.view == "" || m.frames.vizRow < 1 || m.frames.vizW < 8 {
		t.Fatalf("full view should cache viz geom, row=%d w=%d", m.frames.vizRow, m.frames.vizW)
	}
	if m.frames.nowPlayingRow < 1 || m.frames.nowPlayingW < 8 {
		t.Fatalf("full view should now playing geom, row=%d w=%d", m.frames.nowPlayingRow, m.frames.nowPlayingW)
	}
	if m.frames.nowPlayingRow != m.frames.vizRow {
		t.Fatalf("waveform should share the now playing row, nowPlaying=%d viz=%d", m.frames.nowPlayingRow, m.frames.vizRow)
	}
	if m.frames.vizCol <= m.frames.nowPlayingCol {
		t.Fatalf("waveform should sit to the right of now playing, nowPlayingCol=%d vizCol=%d", m.frames.nowPlayingCol, m.frames.vizCol)
	}
	g := m.playerGeom()
	wantVizCol := vizPaintCol(m.frames.nowPlayingCol, g.nowPlayingInnerW)
	if m.frames.vizCol != wantVizCol {
		t.Fatalf("waveform should align in now playing, vizCol=%d want=%d", m.frames.vizCol, wantVizCol)
	}
	if m.frames.vizW != g.vizW {
		t.Fatalf("waveform width should match layout, vizW=%d want=%d", m.frames.vizW, g.vizW)
	}
	m.wavePeaks = []int{255, 80, 20}
	m.frames.freeze = true
	second := m.View()
	if first != second {
		t.Fatal("frozen view should return the last full frame")
	}
}

func TestLiveStatusFreezesOnClockTick(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	m.status.Path = "/tmp/x.mp3"
	m.status.PositionLabel = "1:00"
	m.status.DurationLabel = "4:00"
	before := m.View()
	next := m.status
	next.Position = 61
	next.PositionLabel = "1:01"
	got, _ := m.Update(liveMsg{status: next})
	m = got.(model)
	if !m.frames.freeze {
		t.Fatal("clock tick should freeze the frame and patch now playing in place")
	}
	after := m.View()
	if before != after {
		t.Fatal("clock tick should not rebuild View")
	}
}

func TestWinampPatchStaysInLeftPane(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "World Series"
	m.status.Position = 47
	m.status.Duration = 180
	m.status.State = "playing"
	got := m.renderNowPlaying(40)
	for i, line := range strings.Split(got, "\n") {
		if lipgloss.Width(line) != 40 {
			t.Fatalf("now playing row %d width %d want 40: %q", i, lipgloss.Width(line), line)
		}
	}
}

func TestLiveStatusFreezesView(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	before := m.View()
	got, _ := m.Update(liveMsg{status: m.status})
	m = got.(model)
	if !m.frames.freeze {
		t.Fatal("same-track status poll should freeze the full frame")
	}
	after := m.View()
	if before != after {
		t.Fatal("status poll should not rebuild View")
	}
}

func TestTickMsgKeepsFrozenView(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	before := m.View()
	got, _ := m.Update(liveMsg{status: m.status})
	m = got.(model)
	got, _ = m.Update(tickMsg{})
	m = got.(model)
	if !m.frames.freeze {
		t.Fatal("status tick should keep the frame frozen")
	}
	after := m.View()
	if before != after {
		t.Fatal("status tick should not rebuild View")
	}
}

func TestVizPainterStartsOnPlay(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	_ = m.View()
	m.status.State = "playing"
	m.syncViz()
	if liveVizEnabled {
		if !m.paint.running() {
			t.Fatal("playing should start the vis painter")
		}
	} else if m.paint.running() {
		t.Fatal("live vis off should not start the painter")
	}
	m.status.State = "paused"
	m.syncViz()
	if m.paint.running() {
		t.Fatal("pause should stop the vis painter")
	}
	m.status.State = "playing"
	m.vizMode = vizModeNone
	m.syncViz()
	if m.paint.running() {
		t.Fatal("none should not start the vis painter")
	}
}

func TestPickArtCellSizePrefersIoctl(t *testing.T) {
	cw, ch := pickArtCellSize(8, 16, 12, 24)
	if cw != 12 || ch != 24 {
		t.Fatalf("ioctl cell pixels should win, got %d x %d", cw, ch)
	}
	cw, ch = pickArtCellSize(8, 16, 0, 0)
	if cw != 8 || ch != 16 {
		t.Fatalf("query fallback = %d x %d", cw, ch)
	}
	cw, ch = pickArtCellSize(0, 0, 10, 20)
	if cw != 10 || ch != 20 {
		t.Fatalf("ioctl only = %d x %d", cw, ch)
	}
}

func TestKittyArtPlacesAtCursor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	layout, seq, err := renderKittyArt(img, 4, 3)
	if err != nil {
		t.Fatalf("render kitty art: %v", err)
	}
	if !strings.Contains(seq, "\x1b_G") {
		t.Fatalf("want kitty transmit sequence, got %q", seq)
	}
	if !strings.Contains(seq, "a=T") {
		t.Fatalf("want immediate kitty placement, got %q", seq)
	}
	if strings.Contains(seq, kittyPlaceholder) || strings.Contains(layout, kittyPlaceholder) {
		t.Fatal("kitty art should not use unicode placeholders")
	}
	if strings.TrimSpace(lipglossStrip(layout)) != "" {
		t.Fatalf("art cells should be blank under the overlay, got %q", layout)
	}
	c, r, ok := kittyPlacementCells(seq)
	if !ok {
		t.Fatalf("want placement c/r in seq, got %q", seq)
	}
	if c != 4 || r != 3 {
		t.Fatalf("placement c,r = %d,%d want 4,3", c, r)
	}
}

func TestOverlayArtPositionsWithCursor(t *testing.T) {
	got := overlayArt("view", "\x1b_Gseq", 0, 0)
	if got != "view\x1b_Gseq" {
		t.Fatalf("zero cursor should append, got %q", got)
	}
	got = overlayArt("view", "\x1b_Gseq", 8, 40)
	if !strings.Contains(got, "\x1b[8;40H") {
		t.Fatalf("overlay should CUP to the artwork pane, got %q", got)
	}
}

func TestKittyPlaceSeq(t *testing.T) {
	got := kittyPlaceSeq(4, 3)
	if !strings.Contains(got, "a=p") || !strings.Contains(got, "I=1") {
		t.Fatalf("want short kitty place, got %q", got)
	}
	c, r, ok := kittyPlacementCells(got)
	if !ok || c != 4 || r != 3 {
		t.Fatalf("place c,r = %d,%d ok=%v", c, r, ok)
	}
}

func TestArtRestorerRepaintsAfterWrite(t *testing.T) {
	frames := &frameCache{artRow: 8, artCol: 40, artPlace: "PLACE", artSeq: "FULL"}
	var buf bytes.Buffer
	w := &artRestorer{w: &buf, frames: frames}
	if _, err := w.Write([]byte("line")); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "line") || !strings.Contains(got, "PLACE") {
		t.Fatalf("write should restore art, got %q", got)
	}
	if strings.Contains(got, "FULL") {
		t.Fatal("should use short place after the image is already loaded")
	}
	buf.Reset()
	frames.artDirty = true
	if _, err := w.Write([]byte("next")); err != nil {
		t.Fatal(err)
	}
	got = buf.String()
	if !strings.Contains(got, "FULL") {
		t.Fatalf("dirty write should retransmit, got %q", got)
	}
}

func TestArtRestorerSkipsRestoreOnQuit(t *testing.T) {
	frames := &frameCache{artRow: 8, artCol: 40, artPlace: "PLACE", quitting: true}
	var buf bytes.Buffer
	w := &artRestorer{w: &buf, frames: frames}
	if _, err := w.Write([]byte("line")); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if got != "line" {
		t.Fatalf("quit should not restore art, got %q", got)
	}
}

func TestSquareArtRowsFollowsCols(t *testing.T) {
	if got := squareArtworkRows(16, 8, 16); got != 8 {
		t.Fatalf("16 cols at 8x16 cells -> 8 rows, got %d", got)
	}
}

func TestBoundImageCapsAt600(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1200, 800))
	got := boundImage(src, 600)
	b := got.Bounds()
	if b.Dx() != 600 || b.Dy() != 400 {
		t.Fatalf("bound 1200x800 -> %dx%d, want 600x400", b.Dx(), b.Dy())
	}
}

func TestPlayingTrackRowIsGreen(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.queueFiltered = []library.Track{
		{Path: "/tmp/x.mp3", Title: "World Series", Artist: "Sound Power", Duration: 201, Year: "2004"},
		{Path: "/tmp/y.mp3", Title: "Fabriclive 36", Artist: "DJ Hype", Duration: 369, Year: "2006"},
	}
	m.status.Path = "/tmp/x.mp3"
	got := m.renderPlaylistTrackList(m.queueFiltered, 0, 0, 80, 5, false)
	plain := lipglossStrip(got)
	if !strings.Contains(plain, "▶") {
		t.Fatalf("playing row should keep the play marker, got %q", plain)
	}
	if !strings.Contains(got, "42m") {
		t.Fatalf("playing row should use green background, got %q", got)
	}
}

func TestIdleTrackNamesAreWhite(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.queueFiltered = []library.Track{
		{Path: "/tmp/x.mp3", Title: "World Series", Artist: "Sound Power", Duration: 201, Year: "2004"},
		{Path: "/tmp/y.mp3", Title: "Fabriclive 36", Artist: "DJ Hype", Duration: 369, Year: "2006"},
	}
	got := m.renderPlaylistTrackList(m.queueFiltered, 0, 0, 80, 5, false)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("want two rows, got %q", got)
	}
	white := "[37m"
	if !strings.Contains(lines[1], white) {
		t.Fatalf("idle track name should use terminal white, got %q", lines[1])
	}
	if !strings.Contains(lipglossStrip(lines[1]), "DJ Hype") {
		t.Fatalf("missing idle title: %q", lipglossStrip(lines[1]))
	}
}

func TestFourPanelLayout(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 3, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "grime"}}
	m.queueFiltered = []library.Track{{Title: "World Series", Artist: "Sound Power"}}
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "World Series"
	m.status.Artist = "Sound Power"
	m.status.State = "playing"
	got := lipglossStrip(m.View())
	if !strings.Contains(got, "now playing") {
		t.Fatalf("want now playing fieldset, got %q", got)
	}
	if !strings.Contains(got, "browse") {
		t.Fatalf("want browse column, got %q", got)
	}
	if !strings.Contains(got, "playlist (1)") {
		t.Fatalf("want playlist legend, got %q", got)
	}
	if !strings.Contains(got, "grime/") {
		t.Fatalf("want filetree entries, got %q", got)
	}
	if !strings.Contains(got, "likes") {
		t.Fatalf("want playlists in browse, got %q", got)
	}
	if strings.Index(got, "likes") < strings.Index(got, "grime/") {
		t.Fatalf("playlists should sit below filetree entries, got %q", got)
	}
	if strings.Contains(got, "EVOPLAYER") {
		t.Fatalf("top menu bar should be gone, got %q", got)
	}
	footer := lipglossStrip(m.renderFooter())
	if footer != "" {
		t.Fatalf("footer should be omitted when empty, got %q", footer)
	}
	lines := strings.Split(got, "\n")
	idx := -1
	for i, line := range lines {
		if strings.Contains(line, "now playing") {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("missing now playing legend, got %q", got)
	}
	raw := m.View()
	for _, label := range []string{"now playing", "browse", "playlist (1)"} {
		if !strings.Contains(raw, "\x1b[1m") && !strings.Contains(raw, "\x1b[1;") {
			t.Fatalf("legends should be bold, missing bold in %q", raw)
		}
		if !strings.Contains(lipglossStrip(raw), label) {
			t.Fatalf("missing legend %q", label)
		}
	}
	m.status.Year = "1994"
	raw = m.View()
	if !strings.Contains(lipglossStrip(raw), "1994") {
		t.Fatalf("art legend should show year, got %q", lipglossStrip(raw))
	}
	if !strings.Contains(raw, "\x1b[1m") && !strings.Contains(raw, "\x1b[1;") {
		t.Fatalf("art year legend should be bold, got %q", raw)
	}
	found := false
	for _, line := range lines[idx+1:] {
		if strings.Contains(line, "WORLD SERIES") || strings.Contains(line, "█▀█") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("now playing display should follow the legend, got %q", got)
	}
	if strings.Contains(got, "_ □") {
		t.Fatalf("now playing should not draw window chrome, got %q", got)
	}
}

func TestTabCyclesTreeQueue(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	if m.focus != focusBrowse {
		t.Fatalf("start on filetree, got %v", m.focus)
	}
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(model)
	if m.focus != focusPlaylist {
		t.Fatalf("tab should move to playlist, got %v", m.focus)
	}
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(model)
	if m.focus != focusBrowse {
		t.Fatalf("tab should return to filetree, got %v", m.focus)
	}
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = got.(model)
	if m.focus != focusPlaylist {
		t.Fatalf("shift-tab should reverse to playlist, got %v", m.focus)
	}
	found := false
	for _, row := range helpKeys() {
		if row.Key == "tab" && strings.Contains(row.Label, "playlist") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("help should say tab cycles files / playlist")
	}
	addDir := false
	artKey := false
	for _, row := range helpKeys() {
		if row.Key == "d" && strings.Contains(row.Label, "add dir") {
			addDir = true
		}
		if row.Key == "a" && strings.Contains(row.Label, "art") {
			artKey = true
		}
		if row.Key == "a" && strings.Contains(row.Label, "add") {
			t.Fatal("help should not bind a to add")
		}
	}
	if !addDir {
		t.Fatal("help should bind d to add dir")
	}
	if !artKey {
		t.Fatal("help should bind a to art")
	}
}

func TestMouseWheelDoesNotScroll(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusBrowse
	m.browse = []library.BrowseEntry{
		{Type: "dir", Name: "a"},
		{Type: "dir", Name: "b"},
		{Type: "dir", Name: "c"},
	}
	got, _ := m.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	})
	next := got.(model)
	if next.browseIdx != 0 {
		t.Fatalf("mouse wheel should not move the list, idx=%d", next.browseIdx)
	}
}

func TestYOpensDownload(t *testing.T) {
	m := newModel(paths.Env{})
	m.nav = navFromIndex(nil)
	m.ready = true
	m.focus = focusBrowse
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = got.(model)
	if !m.downloadSelected() {
		t.Fatal("y should open download")
	}
	if m.focus != focusPlaylist {
		t.Fatalf("download should focus playlist pane, got %v", m.focus)
	}
	if cmd == nil {
		t.Fatal("y should focus the url field")
	}
}

func TestDownloadPaneTabFocusesURL(t *testing.T) {
	m := newModel(paths.Env{})
	m.nav = navFromIndex(nil)
	m.ready = true
	m.selectSidebarTool("download")
	m.focus = focusBrowse
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(model)
	if m.focus != focusPlaylist {
		t.Fatalf("tab should move to download pane, got %v", m.focus)
	}
	if cmd == nil {
		t.Fatal("tab should focus the url field")
	}
}

func TestDownloadPaneIgnoresArrowKeys(t *testing.T) {
	m := newModel(paths.Env{})
	m.nav = navFromIndex(nil)
	m.ready = true
	got, _ := m.openDownload()
	m = got.(model)
	m.downloadURL.Focus()
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = got.(model)
	if m.focus != focusPlaylist {
		t.Fatalf("up should stay on download pane, got %v", m.focus)
	}
	if !m.downloadURL.Focused() {
		t.Fatal("up should keep the url field focused")
	}
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = got.(model)
	if m.focus != focusPlaylist || !m.downloadURL.Focused() {
		t.Fatal("down should keep focus on the url field")
	}
}

func TestBrowseToDownloadFocusesURL(t *testing.T) {
	m := newModel(paths.Env{})
	m.nav = navFromIndex(nil)
	m.ready = true
	m.focus = focusBrowse
	m.browse = nil
	tools := m.sidebarTools()
	downloadIdx := len(m.browse) + m.playlistSlotCount()
	for i, t := range tools {
		if t.ID == "download" {
			downloadIdx += i
			break
		}
	}
	m.browseIdx = downloadIdx
	got, cmd := m.moveFocusedWithTools(0)
	m = got.(model)
	if !m.downloadSelected() {
		t.Fatal("browse should select download tool")
	}
	if m.focus != focusPlaylist {
		t.Fatalf("download tool should focus playlist pane, got %v", m.focus)
	}
	if cmd == nil {
		t.Fatal("download tool should focus the url field")
	}
}

func TestBrowseToolsSitUnderPlaylists(t *testing.T) {
	m := newModel(paths.Env{})
	m.focus = focusBrowse
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 3, Kind: "system"}})
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "grime"}}
	got := lipglossStrip(m.renderBrowseInner(30, 16))
	grimeIdx := strings.Index(got, "grime")
	likesIdx := strings.Index(got, "likes")
	settingsIdx := strings.Index(got, "settings")
	downloadIdx := strings.Index(got, "download")
	if grimeIdx < 0 || likesIdx < 0 || settingsIdx < 0 || downloadIdx < 0 {
		t.Fatalf("want folders, playlists, then tools, got %q", got)
	}
	if !(grimeIdx < likesIdx && likesIdx < settingsIdx && settingsIdx < downloadIdx) {
		t.Fatalf("want folders, playlists, settings, download, got %q", got)
	}
	lines := strings.Split(got, "\n")
	likesLine, settingsLine := -1, -1
	for i, line := range lines {
		if likesLine < 0 && strings.Contains(line, "likes") {
			likesLine = i
		}
		if settingsLine < 0 && strings.Contains(line, "settings") {
			settingsLine = i
		}
	}
	if likesLine < 0 || settingsLine != likesLine+2 {
		t.Fatalf("want a 1-row gap under playlists, got likes=%d settings=%d in %q", likesLine, settingsLine, got)
	}
}

func TestSettingsReplacesPlaylistPane(t *testing.T) {
	m := newModel(paths.Env{MusicRoot: "/music"})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 3, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "settings")
	m.settings.Paths.Root = "/music"
	m.settings.Soundcloud.User = "seb-day"
	m.settingsPath.SetValue("/music")
	m.queueFiltered = []library.Track{{Title: "World Series"}}
	got := lipglossStrip(m.View())
	if !strings.Contains(got, "settings") {
		t.Fatalf("settings should replace the playlist pane, got %q", got)
	}
	if !strings.Contains(got, "library") || !strings.Contains(got, "/music") {
		t.Fatalf("settings should show the library root, got %q", got)
	}
	if !strings.Contains(got, "save") {
		t.Fatalf("settings should show save hint, got %q", got)
	}
	if strings.Contains(got, "playlist (1)") {
		t.Fatalf("playlist legend should hide while settings is open, got %q", got)
	}
	if !strings.Contains(got, "a art") {
		t.Fatalf("settings should leave the artwork pane, got %q", got)
	}
}

func TestSettingsSaveRequiresPath(t *testing.T) {
	m := newModel(paths.Env{MusicRoot: "/music"})
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "settings")
	m.focus = focusPlaylist
	m.settingsPath.SetValue("   ")
	got, cmd := m.saveSettingsRoot()
	m = got.(model)
	if cmd != nil {
		t.Fatal("empty path should not issue ipc")
	}
	if m.err != "library path required" {
		t.Fatalf("err = %q", m.err)
	}
}

func TestSettingsEnterFocusesPathField(t *testing.T) {
	m := newModel(paths.Env{MusicRoot: "/music"})
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "settings")
	m.focus = focusPlaylist
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should focus the library path field")
	}
}

func TestSettingsOpenLoadsPath(t *testing.T) {
	m := newModel(paths.Env{MusicRoot: "/music"})
	m.ready = true
	m.settings.Paths.Root = "/music"
	got, cmd := m.openSettings()
	m = got.(model)
	if cmd == nil {
		t.Fatal("openSettings should load settings and focus path field")
	}
	if m.settingsPath.Value() != "/music" {
		t.Fatalf("settings path = %q", m.settingsPath.Value())
	}
}

func TestDownloadReplacesPlaylistAndArt(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	got := lipglossStrip(m.View())
	if strings.Contains(got, "youtube and soundcloud") || strings.Contains(got, "youtube · soundcloud") {
		t.Fatalf("download should not show the tagline, got %q", got)
	}
	if !strings.Contains(got, "Paste YouTube") {
		t.Fatalf("download should show the url field, got %q", got)
	}
	if strings.Contains(got, "playlist (") {
		t.Fatalf("download should replace the playlist pane, got %q", got)
	}
	if strings.Contains(got, "a art") {
		t.Fatalf("download should replace the artwork pane, got %q", got)
	}
	if strings.Contains(got, "import soundcloud") || strings.Contains(got, "import incoming") {
		t.Fatalf("download should not show separate import actions, got %q", got)
	}
	landing := lipglossStrip(m.renderDownloadLanding(60, 20))
	if !strings.Contains(landing, "Paste YouTube") {
		t.Fatalf("download should show url field, got %q", landing)
	}
}

func TestDownloadShowsCancelInURLBoxWhenRunning(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	m.job = jobs.State{Name: "download-url", Status: "running"}
	landing := lipglossStrip(m.renderDownloadLanding(60, 20))
	if !strings.Contains(landing, "cancel") {
		t.Fatalf("running download should show cancel in url box, got %q", landing)
	}
	if strings.Contains(landing, "import soundcloud") || strings.Contains(landing, "import incoming") {
		t.Fatalf("download should not show import actions while running, got %q", landing)
	}
}

func TestDownloadCancelNotShownWhenIdle(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	landing := lipglossStrip(m.renderDownloadLanding(60, 20))
	if strings.Contains(strings.ToLower(landing), "cancel") {
		t.Fatalf("idle download pane should not show cancel in url box, got %q", landing)
	}
}

func TestDownloadCancelIssuesCmd(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	m.focus = focusPlaylist
	m.job = jobs.State{Name: "download", Status: "running"}
	_, cmd := m.cancelDownloadJob()
	if cmd == nil {
		t.Fatal("cancel should issue ipc")
	}
}

func TestURLDownloadStartsJobWithImport(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	m.focus = focusPlaylist
	m.downloadURL.SetValue("https://soundcloud.com/a/b")
	got, cmd := m.startURLDownload()
	m = got.(model)
	if cmd == nil {
		t.Fatal("enter should start the url download job")
	}
	_ = m
	_ = cmd
}

func TestDownloadJobLogRenders(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	m.job = jobs.State{
		Name:   "download",
		Status: "running",
		Log:    "· soundcloud auth from pass\n✓ artist - track.mp3\n",
	}
	got := lipglossStrip(m.renderDownloadLanding(60, 20))
	if !strings.Contains(got, "✓ artist - track.mp3") {
		t.Fatalf("download log should render, got %q", got)
	}
}

func TestDownloadPaneDoesNotFreezeDuringImport(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	m.job = jobs.State{Name: "import", Status: "running", Log: "· importing .incoming\n"}
	got, _ := m.Update(tickMsg{})
	m = got.(model)
	if m.frames.freeze {
		t.Fatal("import job ticks should not freeze the log")
	}
}

func TestDownloadPaneDoesNotFreeze(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	_ = m.View()
	got, _ := m.Update(liveMsg{status: m.status})
	m = got.(model)
	if m.frames.freeze {
		t.Fatal("download pane should keep redrawing instead of freezing")
	}
	m.job = jobs.State{Name: "download", Status: "running", Log: "· fetching likes\n"}
	got, _ = m.Update(tickMsg{})
	m = got.(model)
	if m.frames.freeze {
		t.Fatal("download job ticks should not freeze the log")
	}
	got, _ = m.Update(jobMsg{state: jobs.State{
		Name:   "download",
		Status: "running",
		Log:    "· soundcloud auth from brave\n· fetching likes\n· fetched 200 likes\n",
		Progress: &jobs.Progress{Phase: "fetching likes (200)", Done: 200},
	}})
	m = got.(model)
	if m.frames.freeze {
		t.Fatal("job updates should redraw the download log")
	}
	gotView := lipglossStrip(m.View())
	if !strings.Contains(gotView, "· fetched 200 likes") {
		t.Fatalf("download log should update live, got %q", gotView)
	}
}

func TestTabToPlaylistSelectsPlayingTrack(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.focus = focusBrowse
	m.status.Path = "/tmp/playing.mp3"
	m.queueFiltered = []library.Track{
		{Path: "/tmp/a.mp3"},
		{Path: "/tmp/playing.mp3"},
		{Path: "/tmp/c.mp3"},
	}
	m.playlistIdx = 0
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(model)
	if m.focus != focusPlaylist {
		t.Fatalf("tab should move to playlist, got %v", m.focus)
	}
	if m.playlistIdx != 1 {
		t.Fatalf("playlist caret should land on the playing track, idx=%d", m.playlistIdx)
	}
}

func TestSkipScrollsPlaylistForPlayingTrack(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.focus = focusPlaylist
	vis := m.playlistListVisible()
	if vis < 3 {
		t.Fatalf("need a visible playlist window, got %d", vis)
	}
	tracks := make([]library.Track, vis+10)
	for i := range tracks {
		tracks[i] = library.Track{Path: fmt.Sprintf("/tmp/%d.mp3", i), Title: fmt.Sprintf("t%d", i)}
	}
	m.queue = tracks
	m.queueFiltered = tracks

	t.Run("next", func(t *testing.T) {
		m.playlistIdx = 0
		m.playlistOffset = 3
		playingIdx := m.playlistOffset + vis - 1
		m.status.Path = tracks[playingIdx].Path
		next := m.status
		next.Path = tracks[playingIdx+1].Path
		got, _ := m.Update(liveMsg{status: next})
		m = got.(model)
		if m.playlistOffset != 4 {
			t.Fatalf("next track off-screen should scroll down 1 row, offset=%d want 4", m.playlistOffset)
		}
		if m.playlistIdx != 0 {
			t.Fatalf("caret should stay put, idx=%d", m.playlistIdx)
		}
	})

	t.Run("prev", func(t *testing.T) {
		m.playlistIdx = 0
		m.playlistOffset = 5
		playingIdx := 5
		m.status.Path = tracks[playingIdx].Path
		prev := m.status
		prev.Path = tracks[playingIdx-1].Path
		got, _ := m.Update(liveMsg{status: prev})
		m = got.(model)
		if m.playlistOffset != 4 {
			t.Fatalf("previous track off-screen should scroll up 1 row, offset=%d want 4", m.playlistOffset)
		}
		if m.playlistIdx != 0 {
			t.Fatalf("caret should stay put, idx=%d", m.playlistIdx)
		}
	})
}

func TestTabToPlaylistKeepsCaretWithoutPlayingTrack(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.focus = focusBrowse
	m.queueFiltered = []library.Track{
		{Path: "/tmp/a.mp3"},
		{Path: "/tmp/b.mp3"},
		{Path: "/tmp/c.mp3"},
	}
	m.playlistIdx = 2
	m.playlistOffset = 1
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = got.(model)
	if m.playlistIdx != 2 {
		t.Fatalf("without a playing track, keep the caret, idx=%d", m.playlistIdx)
	}
	if m.playlistOffset != 1 {
		t.Fatalf("caret scroll should stay, offset=%d", m.playlistOffset)
	}
}

func TestSeekTargetClamps(t *testing.T) {
	if got := seekTarget(5, 60, -10); got != 0 {
		t.Fatalf("seek back should clamp to 0, got %v", got)
	}
	if got := seekTarget(55, 60, 10); got != 60 {
		t.Fatalf("seek forward should clamp to duration, got %v", got)
	}
	if got := seekTarget(20, 60, 10); got != 30 {
		t.Fatalf("seek forward 10s, got %v", got)
	}
}

func TestShiftCommaPeriodSeek(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}})
	if cmd != nil {
		t.Fatal("seek without a track should be a no-op")
	}
	m.status.Path = "/tmp/x.mp3"
	m.status.Position = 40
	m.status.Duration = 120
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}})
	if cmd == nil {
		t.Fatal("shift+, should seek back 10s")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'>'}})
	if cmd == nil {
		t.Fatal("shift+. should seek forward 10s")
	}
}

func TestPanelHintLegends(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.State = "playing"
	g := m.playerGeom()
	now := lipglossStrip(m.renderNowPlayingBar())
	if !strings.Contains(now, "find") || !strings.Contains(now, "help") {
		t.Fatalf("now playing should show find and help on the bottom right, got %q", now)
	}
	if !strings.Contains(now, "space") || !strings.Contains(now, "pause") {
		t.Fatalf("now playing should show space pause on the bottom right, got %q", now)
	}
	if strings.Contains(now, "quit") {
		t.Fatalf("now playing should not show quit, got %q", now)
	}
	tree := lipglossStrip(m.renderBrowsePane(g))
	if strings.Contains(tree, "find") || strings.Contains(tree, "quit") || strings.Contains(tree, "space") {
		t.Fatalf("browse panel should not keep now playing hints, got %q", tree)
	}
	if !strings.Contains(tree, "browse") {
		t.Fatalf("browse panel should show title, got %q", tree)
	}
	if !strings.Contains(tree, "play") {
		t.Fatalf("browse panel should show play legend, got %q", tree)
	}
	if !strings.Contains(tree, "d add dir") {
		t.Fatalf("browse panel should show d add dir, got %q", tree)
	}
	if strings.Contains(tree, "a add") {
		t.Fatalf("browse panel should not show a add, got %q", tree)
	}
	queue := lipglossStrip(m.renderPlaylistPane(g))
	if strings.Contains(queue, "browse") || strings.Contains(queue, "add dir") {
		t.Fatalf("playlist should not keep moved hints, got %q", queue)
	}
	if !strings.Contains(queue, "like") {
		t.Fatalf("playlist should show like, got %q", queue)
	}
	art, _ := m.renderArtworkPane(g)
	artPlain := lipglossStrip(art)
	if strings.Contains(artPlain, "like") {
		t.Fatalf("artwork panel should not keep like hint, got %q", artPlain)
	}
	if !strings.Contains(artPlain, "a art") {
		t.Fatalf("artwork panel should show a art, got %q", artPlain)
	}
}

func TestHelpOverlaysQueue(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "help")
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "grime"}}
	got := lipglossStrip(m.View())
	if !strings.Contains(got, "browse") {
		t.Fatalf("help should leave the browse column, got %q", got)
	}
	if !strings.Contains(got, "Help") {
		t.Fatalf("help should overlay the playlist column, got %q", got)
	}
	if !strings.Contains(got, "files /") || !strings.Contains(got, "playlist") {
		t.Fatalf("help body should list tab cycle, got %q", got)
	}
}

func TestLCDGlyphsBottomRowAligned(t *testing.T) {
	for digit := '0'; digit <= '9'; digit++ {
		g := lcdGlyphs[digit]
		if strings.Contains(g[2], "█") {
			t.Fatalf("digit %q bottom row should use ▀ not █ for baseline alignment: %q", digit, g[2])
		}
	}
	if lcdGlyphs['8'] == lcdGlyphs['9'] {
		t.Fatal("9 should be distinct from 8")
	}
	if strings.HasPrefix(lcdGlyphs['9'][1], "█") {
		t.Fatalf("9 middle row should not extend the left stem: %q", lcdGlyphs['9'][1])
	}
}

func TestWinampBarKeepsWaveformBesideClock(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "Beyond Control"
	m.status.Artist = "Razor Boy"
	m.status.State = "playing"
	bar := m.renderNowPlayingBar()
	if h := lipgloss.Height(bar); h != vizPaintRows+2 {
		t.Fatalf("now playing should be %d rows, got %d", vizPaintRows+2, h)
	}
	plain := lipglossStrip(bar)
	if strings.Count(plain, "BEYOND CONTROL") > 1 {
		t.Fatalf("title should not wrap onto a second line, got %q", plain)
	}
	if strings.Count(plain, "SHUFFLE") != 1 {
		t.Fatalf("transport row should appear once, got %q", plain)
	}
	lines := strings.Split(plain, "\n")
	if len(lines) < 4 {
		t.Fatalf("now playing too short: %q", plain)
	}
	if strings.Contains(lines[1], "BEYOND") || strings.Contains(lines[1], "RAZOR") {
		t.Fatalf("clock should sit one row below the legend, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "RAZOR") {
		t.Fatalf("title should start on the second inner row, got %q", lines[2])
	}
	if !strings.HasPrefix(strings.TrimPrefix(lines[2], "│"), "  ") {
		t.Fatalf("clock should sit 1 col in from the border, got %q", lines[2])
	}
}

func TestWinampTransportShowsWhenPaused(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "Track"
	m.status.Artist = "Artist"
	m.status.State = "paused"
	got := lipglossStrip(m.renderNowPlayingBar())
	if !strings.Contains(got, "⏸") {
		t.Fatalf("paused transport should show pause glyph, got %q", got)
	}
	if strings.Count(got, "SHUFFLE") != 1 {
		t.Fatalf("paused transport row should stay visible once, got %q", got)
	}
}

func TestWinampPatchKeepsTransportRows(t *testing.T) {
	frames := &frameCache{nowPlayingRow: 4, nowPlayingCol: 3, nowPlayingW: 40}
	m := model{frames: frames, status: playback.Status{Path: "/tmp/x.mp3", State: "paused"}}
	m.patchNowPlaying()
	if !strings.Contains(frames.lastNowPlaying, "⏸") {
		t.Fatalf("patch cache should include transport row, got %q", frames.lastNowPlaying)
	}
	if strings.Count(frames.lastNowPlaying, "\n") != vizPaintRows-1 {
		t.Fatalf("patch cache should keep %d rows, got %q", vizPaintRows, frames.lastNowPlaying)
	}
	if strings.Count(lipglossStrip(frames.lastNowPlaying), "SHUFFLE") != 1 {
		t.Fatalf("patch cache should not duplicate transport, got %q", frames.lastNowPlaying)
	}
	first := strings.Split(lipglossStrip(frames.lastNowPlaying), "\n")[0]
	if strings.TrimSpace(first) != "" {
		t.Fatalf("patch cache should keep a 1-row gap above the clock, got %q", frames.lastNowPlaying)
	}
	for _, line := range strings.Split(lipglossStrip(frames.lastNowPlaying), "\n") {
		if strings.Contains(line, "SHUFFLE") && !strings.HasPrefix(line, " ") {
			t.Fatalf("transport should keep a 1-col left pad, got %q", line)
		}
	}
}

func TestWinampChromeShowsTransport(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "So Distinguished (Instrumental)"
	m.status.Artist = "Crazyee Bandit"
	m.status.Position = 3*60 + 47
	m.status.Duration = 5 * 60
	m.status.Volume = 80
	m.status.Shuffle = true
	m.status.State = "playing"
	m.status.Year = "2008"
	m.status.Label = "Dub Police"
	m.status.Album = "DPZ007"
	got := lipglossStrip(m.renderNowPlaying(72))
	if strings.Contains(got, "Dubstep") {
		t.Fatalf("now playing should not keep the old pill row, got %q", got)
	}
	if strings.Contains(got, "EVOPLAYER") || strings.Contains(got, "_ □") || strings.Contains(got, "┌") {
		t.Fatalf("now playing should not draw its own window chrome, got %q", got)
	}
	if !strings.Contains(got, "CRAZYEE BANDIT") || !strings.Contains(got, "SO DISTINGUISHED") {
		t.Fatalf("marquee should show artist and title, got %q", got)
	}
	if !strings.Contains(got, "48 KHZ") {
		t.Fatalf("want format in the status column, got %q", got)
	}
	if !strings.Contains(got, "2008") || !strings.Contains(got, "DUB POLICE") || !strings.Contains(got, "DPZ007") {
		t.Fatalf("want year and label/cat no under the marquee, got %q", got)
	}
	display := strings.Split(got, "\n")
	if len(display) < 2 {
		t.Fatalf("want display rows, got %q", got)
	}
	titleRow, releaseRow, meterRow := display[0], display[1], display[2]
	if strings.Contains(titleRow, "48 KHZ") {
		t.Fatalf("48 KHZ should not sit on the title row, got %q", titleRow)
	}
	if strings.Contains(releaseRow, "EQ") {
		t.Fatalf("EQ should sit under 48 KHZ, got %q", releaseRow)
	}
	stereoAt := strings.Index(titleRow, "STEREO")
	khzAt := strings.Index(releaseRow, "48 KHZ")
	yearAt := strings.Index(releaseRow, "2008")
	if khzAt < 0 || stereoAt < 0 {
		t.Fatalf("want STEREO over 48 KHZ, title=%q release=%q", titleRow, releaseRow)
	}
	if khzAt < yearAt {
		t.Fatalf("year should sit left of 48 KHZ, got %q", releaseRow)
	}
	stereoEnd := lipgloss.Width(titleRow[:stereoAt]) + lipgloss.Width("STEREO")
	khzEnd := lipgloss.Width(releaseRow[:khzAt]) + lipgloss.Width("48 KHZ")
	if stereoEnd != khzEnd {
		t.Fatalf("48 KHZ should right-align with STEREO, title=%q release=%q", titleRow, releaseRow)
	}
	plAt := strings.Index(meterRow, "PL")
	if plAt < 0 {
		t.Fatalf("EQ PL should sit on the meter row, got %q", meterRow)
	}
	plEnd := lipgloss.Width(meterRow[:plAt]) + lipgloss.Width("PL")
	if stereoEnd != plEnd {
		t.Fatalf("EQ PL should right-align with STEREO, title=%q meter=%q", titleRow, meterRow)
	}
	if !strings.Contains(got, "STEREO") || !strings.Contains(got, "PL") {
		t.Fatalf("status chrome should live in the now playing display, got %q", got)
	}
	if !strings.Contains(got, "SHUFFLE") || !strings.Contains(got, "▶") {
		t.Fatalf("want transport and shuffle, got %q", got)
	}
	idle := lipglossStrip(model{}.renderNowPlaying(80))
	if !strings.Contains(idle, "NOTHING") {
		t.Fatalf("idle chrome should say nothing playing, got %q", idle)
	}
	if nowPlayingClock("", 0) != "00:00" {
		t.Fatalf("idle clock should start at 00:00, got %q", nowPlayingClock("", 0))
	}
	lines := strings.Split(got, "\n")
	if len(lines) < 4 {
		t.Fatalf("want display and transport rows, got %d: %q", len(lines), got)
	}
	w := lipgloss.Width(lines[0])
	for i, line := range lines {
		if lipgloss.Width(line) != w {
			t.Fatalf("row %d width %d want %d: %q", i, lipgloss.Width(line), w, line)
		}
	}
}

func TestNowPlayingLikedHeart(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "Genesis"
	m.status.Artist = "Busta Rhymes"
	got := lipglossStrip(m.renderNowPlaying(72))
	if strings.Contains(got, "♥") {
		t.Fatalf("unliked track should not show a heart, got %q", got)
	}
	m.status.Liked = true
	raw := m.renderNowPlaying(72)
	got = lipglossStrip(raw)
	if !strings.Contains(got, "♥") {
		t.Fatalf("liked track should show a heart, got %q", got)
	}
	if !strings.Contains(raw, "\x1b[31m") && !strings.Contains(raw, "\x1b[38;5;1m") {
		t.Fatalf("liked heart should use terminal colour 1, got %q", raw)
	}
}

func TestWinampSlidersKeepRowGeometry(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "Track"
	m.status.Artist = "Brockie"
	m.status.Duration = 60
	m.status.State = "playing"
	const width = 72
	base := strings.Split(m.renderNowPlaying(width), "\n")
	baseW := lipgloss.Width(base[0])
	baseN := len(base)
	for vol := 0; vol <= 100; vol += 5 {
		m.status.Volume = vol
		for _, pos := range []float64{0, 0.15, 15, 30, 59, 60} {
			m.status.Position = pos
			got := m.renderNowPlaying(width)
			lines := strings.Split(got, "\n")
			if len(lines) != baseN {
				t.Fatalf("vol=%d pos=%v rows %d want %d", vol, pos, len(lines), baseN)
			}
			for i, line := range lines {
				if strings.Contains(line, "\n") {
					t.Fatalf("row %d wrapped at vol=%d pos=%v", i, vol, pos)
				}
				if lipgloss.Width(line) != baseW {
					t.Fatalf("row %d width %d want %d at vol=%d pos=%v", i, lipgloss.Width(line), baseW, vol, pos)
				}
			}
		}
	}
	low := lipgloss.Width(nowPlayingVolume(0, 12))
	high := lipgloss.Width(nowPlayingVolume(100, 12))
	if low != high || low != 12 {
		t.Fatalf("volume bar width should stay %d, got 0=%d 100=%d", 12, low, high)
	}
	if lipgloss.Width(nowPlayingDotSlider(0, 20)) != lipgloss.Width(nowPlayingDotSlider(19, 20)) {
		t.Fatal("balance dot position should not change row width")
	}
}

func TestFooterHintsHaveBorderTails(t *testing.T) {
	m := newModel(paths.Env{})
	got := lipglossStrip(m.renderFooterHints())
	if strings.Contains(got, "──") {
		t.Fatalf("hints should not use border separators, got %q", got)
	}
	if strings.Contains(got, "browse") || strings.Contains(got, "space") || strings.Contains(got, "vis") || strings.Contains(got, "like") || strings.Contains(got, "quit") {
		t.Fatalf("moved hints should not stay in the footer, got %q", got)
	}
}

func TestFooterIsHintsOnly(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.PositionLabel = "1:47"
	m.status.DurationLabel = "4:25"
	if lipglossStrip(m.renderFooter()) != "" {
		t.Fatalf("footer should be empty when there are no hints")
	}
	m.nav = navFromIndex(nil)
	m.navIdx = navIndex(m.nav, "download")
	footer := lipglossStrip(m.renderFooter())
	if footer != "" {
		t.Fatalf("download should keep hints on the pane, not the footer, got %q", footer)
	}
	if strings.Contains(footer, "1:47/4:25") {
		t.Fatalf("time belongs in now playing, not the hints footer, got %q", footer)
	}
	if strings.Contains(footer, "VOL ") || strings.Contains(footer, "▎") {
		t.Fatalf("volume indicator should be gone, got %q", footer)
	}
}

func TestStartupShowsPlaylistLoading(t *testing.T) {
	m := newModel(paths.Env{})
	got := lipglossStrip(m.renderBrowseInner(30, 12))
	if !strings.Contains(got, "loading") {
		t.Fatalf("playlist slot should show loading before nav arrives, got %q", got)
	}
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 3, Kind: "system"}})
	got = lipglossStrip(m.renderBrowseInner(30, 12))
	if !strings.Contains(got, "likes") {
		t.Fatalf("playlists should replace loading after nav loads, got %q", got)
	}
}

func TestScanningHidesLibrary(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
	})
	m.navIdx = navIndex(m.nav, "filetree")
	m.browse = []library.BrowseEntry{{Type: "dir", Name: "grime"}}
	normal := lipglossStrip(m.renderBrowseInner(24, 12))
	if !strings.Contains(normal, "likes") {
		t.Fatalf("playlists should show before scan, got %q", normal)
	}
	m.job = jobs.State{Name: "scan", Status: "running"}
	scanning := lipglossStrip(m.renderBrowseInner(24, 12))
	if !strings.Contains(scanning, "Please wait") {
		t.Fatalf("scan should replace the filetree, got %q", scanning)
	}
	if strings.Contains(scanning, "likes") || strings.Contains(scanning, "grime") {
		t.Fatalf("playlists should hide while scanning, got %q", scanning)
	}
	m.job = jobs.State{Name: "scan", Status: "done"}
	after := lipglossStrip(m.renderBrowseInner(24, 12))
	if strings.Contains(after, "Please wait") {
		t.Fatalf("wait message should clear when scan ends, got %q", after)
	}
	if !strings.Contains(after, "likes") {
		t.Fatalf("playlists should return after scan, got %q", after)
	}
}

func TestWinampBarFitsMainWidth(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "Beyond Control"
	m.status.Artist = "Razor Boy"
	m.status.State = "playing"
	got := m.renderNowPlayingBar()
	wantW := m.mainWidth()
	for i, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w != wantW {
			t.Fatalf("row %d width %d want %d: %q", i, w, wantW, lipglossStrip(line))
		}
	}
	plain := lipglossStrip(got)
	if !strings.Contains(plain, "STEREO") || !strings.Contains(plain, "PL") {
		t.Fatalf("now playing bar should include status chrome in now playing, got %q", plain)
	}
}

func TestVizPaintStaysInsideNowPlaying(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.Path = "/tmp/x.mp3"
	m.status.State = "playing"
	_ = m.View()
	g := m.playerGeom()
	if m.frames.vizCol+g.vizW > m.mainWidth() {
		t.Fatalf("viz should end before the right border, col=%d w=%d width=%d", m.frames.vizCol, g.vizW, m.mainWidth())
	}
	wantCol := vizPaintCol(nowPlayingContentCol(), g.nowPlayingInnerW)
	if m.frames.vizCol != wantCol {
		t.Fatalf("viz col=%d want %d", m.frames.vizCol, wantCol)
	}
}

func TestWinampPaintKeepsLeftPad(t *testing.T) {
	if nowPlayingContentCol() != 2+nowPlayingPadX {
		t.Fatalf("now playing overlay should start after the fieldset pad, col=%d", nowPlayingContentCol())
	}
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.Path = "/tmp/x.mp3"
	m.status.State = "playing"
	_ = m.View()
	if m.frames.nowPlayingCol != nowPlayingContentCol() {
		t.Fatalf("now playing overlay col=%d want %d", m.frames.nowPlayingCol, nowPlayingContentCol())
	}
}

func TestChromeUsesTerminalPalette(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	got := m.renderNowPlayingBar()
	if strings.Contains(got, "38;2;") || strings.Contains(got, "48;2;") {
		t.Fatalf("header should use terminal palette colours, got %q", got)
	}
	if !strings.Contains(got, "[3") && !strings.Contains(got, "[9") {
		t.Fatalf("header should use ANSI palette colours, got %q", got)
	}
}

func lipglossStrip(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestViewFillsTerminalHeight(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "filetree")
	out := m.View()
	h := lipgloss.Height(out)
	if h != 40 {
		t.Fatalf("view height %d want 40", h)
	}
}

func TestAOpensArtLookupFromBrowse(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusBrowse
	m.status.Path = "/tmp/playing.mp3"
	m.browse = []library.BrowseEntry{{
		Type:  "track",
		Track: library.Track{Path: "/tmp/other.mp3", Title: "Other"},
	}}
	m.browseIdx = 0
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = got.(model)
	if !m.artPicker {
		t.Fatal("a should open art lookup from browse")
	}
	if m.artPickPath != "/tmp/playing.mp3" {
		t.Fatalf("art lookup path=%q", m.artPickPath)
	}
	if cmd == nil {
		t.Fatal("a should search discogs")
	}
}

func TestAOpensArtLookup(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusPlaylist
	m.status.Path = "/tmp/playing.mp3"
	m.queueFiltered = []library.Track{{Path: "/tmp/playing.mp3"}}
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = got.(model)
	if !m.artPicker {
		t.Fatal("a should open art lookup from the playlist column")
	}
	if m.artPickPath != "/tmp/playing.mp3" {
		t.Fatalf("art lookup path=%q", m.artPickPath)
	}
	if cmd == nil {
		t.Fatal("a should search discogs")
	}
}

func TestAOpensArtLookupForPlayingTrackNotCaret(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusPlaylist
	m.status.Path = "/tmp/playing.mp3"
	m.queueFiltered = []library.Track{
		{Path: "/tmp/playing.mp3"},
		{Path: "/tmp/queued.mp3"},
	}
	m.playlistIdx = 1
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = got.(model)
	if !m.artPicker {
		t.Fatal("a should open art lookup for the playing track")
	}
	if m.artPickPath != "/tmp/playing.mp3" {
		t.Fatalf("art lookup should ignore the playlist caret, path=%q", m.artPickPath)
	}
	if cmd == nil {
		t.Fatal("a should search discogs")
	}
}

func TestArtPickerCloseRestoresPlaylistPlace(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusPlaylist
	m.status.Path = "/tmp/playing.mp3"
	m.queueFiltered = []library.Track{
		{Path: "/tmp/a.mp3"},
		{Path: "/tmp/b.mp3"},
		{Path: "/tmp/c.mp3"},
	}
	m.playlistIdx = 2
	m.playlistOffset = 1
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = got.(model)
	m.playlistIdx = 0
	m.playlistOffset = 0
	got, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = got.(model)
	if m.artPicker {
		t.Fatal("esc should close art lookup")
	}
	if m.focus != focusPlaylist {
		t.Fatalf("focus=%v want playlist", m.focus)
	}
	if m.playlistIdx != 2 || m.playlistOffset != 1 {
		t.Fatalf("playlist idx=%d offset=%d want 2,1", m.playlistIdx, m.playlistOffset)
	}
}

func TestArtApplyRestoresPlaylistPlace(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusPlaylist
	m.status.Path = "/tmp/playing.mp3"
	m.queueFiltered = []library.Track{
		{Path: "/tmp/a.mp3"},
		{Path: "/tmp/b.mp3"},
	}
	m.playlistIdx = 1
	m.playlistOffset = 1
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = got.(model)
	m.playlistIdx = 0
	got, _ = m.Update(artApplyMsg{path: "/tmp/b.mp3", art: "/tmp/cover.jpg"})
	m = got.(model)
	if m.artPicker {
		t.Fatal("apply should close art lookup")
	}
	if m.playlistIdx != 1 || m.playlistOffset != 1 {
		t.Fatalf("playlist idx=%d offset=%d want 1,1", m.playlistIdx, m.playlistOffset)
	}
}

func TestAArtNeedsPlayingTrack(t *testing.T) {
	m := newModel(paths.Env{})
	m.ready = true
	m.focus = focusPlaylist
	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = got.(model)
	if m.artPicker {
		t.Fatal("art lookup should need a playing track")
	}
	if cmd != nil {
		t.Fatal("art lookup without a track should not start a search")
	}
}

func TestArtPickApplyURLUsesSelectedCover(t *testing.T) {
	m := newModel(paths.Env{})
	m.artPicker = true
	m.playlistIdx = 1
	m.artHits = []art.Result{
		{URL: "https://example.com/a.jpg", Thumb: "https://example.com/a-thumb.jpg"},
		{URL: "https://example.com/b.jpg", Thumb: "https://example.com/b-thumb.jpg"},
	}
	m.artPickPreviewURL = "https://example.com/a-thumb.jpg"
	m.artPickPreviewIdx = 0
	m.artPreviewImg = image.NewRGBA(image.Rect(0, 0, 1, 1))
	if got := m.artPickApplyURL(); got != "https://example.com/b.jpg" {
		t.Fatalf("apply should use the selected cover, got %q", got)
	}
	got := art.PreviewURL(art.Result{Thumb: "https://i.discogs.com/x/fit-in/150x150/R-1.jpg"})
	if got != "https://i.discogs.com/x/fit-in/600x600/R-1.jpg" {
		t.Fatalf("apply should size discogs thumbs, got %q", got)
	}
}

func TestArtPickShowsCachedCoverOnMove(t *testing.T) {
	m := newModel(paths.Env{})
	m.artPicker = true
	m.focus = focusPlaylist
	m.artHits = []art.Result{
		{URL: "https://example.com/a.jpg", Thumb: "https://example.com/a-thumb.jpg"},
		{URL: "https://example.com/b.jpg", Thumb: "https://example.com/b-thumb.jpg"},
	}
	b := image.NewRGBA(image.Rect(0, 0, 2, 1))
	m.storeArtPreview("https://example.com/b-thumb.jpg", b)
	m.playlistIdx = 0
	cmd := m.armArtPickPreview()
	if cmd == nil {
		t.Fatal("idx 0 should still fetch when uncached")
	}
	m.moveQueue(1)
	cmd = m.armArtPickPreview()
	if cmd != nil {
		t.Fatal("cached cover should swap without a fetch")
	}
	if m.artPreviewImg != b {
		t.Fatal("cached cover should become the visible preview")
	}
	if m.artPickPreviewIdx != 1 {
		t.Fatalf("preview idx=%d", m.artPickPreviewIdx)
	}
}

func TestArtPreviewIgnoresStaleSelection(t *testing.T) {
	m := newModel(paths.Env{})
	m.artPicker = true
	m.playlistIdx = 1
	m.artPickPreviewGen = 2
	next, _ := m.Update(artPreviewMsg{
		gen: 2,
		idx: 0,
		url: "https://example.com/a-thumb.jpg",
		img: image.NewRGBA(image.Rect(0, 0, 1, 1)),
	})
	m = next.(model)
	if m.artPreviewImg != nil {
		t.Fatal("preview for a previous row should not replace the current cover")
	}
	if _, ok := m.artPreviewCache["https://example.com/a-thumb.jpg"]; !ok {
		t.Fatal("stale preview should still be cached for later")
	}
}

func TestArtPickerListsHits(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.artPicker = true
	m.artPickQuery = "wiley wot"
	m.artHits = []art.Result{{Label: "Wot Do U Call It", Year: "2004"}}
	m.focus = focusPlaylist
	got := lipglossStrip(m.renderPlaylistPane(m.playerGeom()))
	if !strings.Contains(got, "cover") {
		t.Fatalf("picker should retitle the playlist column, got %q", got)
	}
	if !strings.Contains(got, "Wot Do U Call It") || !strings.Contains(got, "2004") {
		t.Fatalf("picker should list the discogs hit, got %q", got)
	}
	if !strings.Contains(got, "set") || !strings.Contains(got, "track") {
		t.Fatalf("picker should show set/track hints, got %q", got)
	}
}

func TestArtPickerHidesListUntilCoversLoad(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.artPicker = true
	m.artPickBusy = true
	m.artPickQuery = "wiley wot"
	m.artHits = []art.Result{{Label: "Wot Do U Call It", Year: "2004"}}
	m.focus = focusPlaylist
	got := lipglossStrip(m.renderPlaylistPane(m.playerGeom()))
	if !strings.Contains(got, "loading covers") {
		t.Fatalf("picker should wait for covers before listing, got %q", got)
	}
	if strings.Contains(got, "Wot Do U Call It") {
		t.Fatal("picker should not list hits until covers are fetched")
	}
}

func TestArtSearchPrefetchesCovers(t *testing.T) {
	m := newModel(paths.Env{})
	m.artPicker = true
	m.artPickPath = "/tmp/playing.mp3"
	next, cmd := m.Update(artSearchMsg{
		path:    "/tmp/playing.mp3",
		query:   "wiley",
		results: []art.Result{{URL: "https://example.com/a.jpg", Label: "A"}},
	})
	m = next.(model)
	if !m.artPickBusy {
		t.Fatal("picker should stay busy while covers download")
	}
	if cmd == nil {
		t.Fatal("search should prefetch covers")
	}
}

func TestArtPrefetchShowsFirstCover(t *testing.T) {
	m := newModel(paths.Env{})
	m.artPicker = true
	m.artPickBusy = true
	m.artPickPath = "/tmp/playing.mp3"
	m.artPickPreviewGen = 3
	m.artHits = []art.Result{
		{URL: "https://example.com/a.jpg", Thumb: "https://example.com/a-thumb.jpg"},
		{URL: "https://example.com/b.jpg", Thumb: "https://example.com/b-thumb.jpg"},
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	next, _ := m.Update(artPrefetchMsg{
		path: "/tmp/playing.mp3",
		gen:  3,
		urls: []string{"https://example.com/a-thumb.jpg", "https://example.com/b-thumb.jpg"},
		imgs: []image.Image{img, image.NewRGBA(image.Rect(0, 0, 2, 1))},
	})
	m = next.(model)
	if m.artPickBusy {
		t.Fatal("picker should open the list after covers load")
	}
	if m.artPreviewImg != img {
		t.Fatal("first cover should fill the artwork panel")
	}
}

func TestFolderOpenTargetBrowseTrack(t *testing.T) {
	m := newModel(paths.Env{MusicRoot: "/music"})
	m.focus = focusBrowse
	m.browse = []library.BrowseEntry{{
		Type:  "track",
		Track: library.Track{Path: "/music/grime/track.mp3"},
	}}
	got := m.folderOpenTarget()
	if got != "/music/grime" {
		t.Fatalf("folder = %q", got)
	}
}

func TestFolderOpenTargetBrowseDir(t *testing.T) {
	m := newModel(paths.Env{MusicRoot: "/music"})
	m.focus = focusBrowse
	m.browse = []library.BrowseEntry{{
		Type:  "dir",
		Track: library.Track{Path: "grime/youtube"},
	}}
	got := m.folderOpenTarget()
	if got != "/music/grime/youtube" {
		t.Fatalf("folder = %q", got)
	}
}

func TestFolderOpenTargetPlaylistTrack(t *testing.T) {
	m := newModel(paths.Env{MusicRoot: "/music"})
	m.focus = focusPlaylist
	m.queueFiltered = []library.Track{{Path: "/music/dnb/a.mp3"}}
	m.playlistIdx = 0
	got := m.folderOpenTarget()
	if got != "/music/dnb" {
		t.Fatalf("folder = %q", got)
	}
}

func TestOpenInFileManagerUsesOverride(t *testing.T) {
	t.Setenv("EVOPLAYER_FILE_MANAGER", "true")
	if err := openInFileManager(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
