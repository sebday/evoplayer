package tui

import (
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playlist"
)

func TestFilterTracks(t *testing.T) {
	tracks := []library.Track{
		{Title: "World Series", Artist: "Sound Power", Genre: "misc"},
		{Title: "Fabriclive", Artist: "DJ Hype", Genre: "Drum & Bass"},
	}
	got := filterTracks(tracks, "hype")
	if len(got) != 1 || got[0].Title != "Fabriclive" {
		t.Fatalf("got %#v", got)
	}
	if n := len(filterTracks(tracks, "")); n != 2 {
		t.Fatalf("empty query = %d", n)
	}
}

func TestNavFromIndex(t *testing.T) {
	got := navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
		{Name: "mixes", Count: 1, Kind: "system"},
	})
	if got[0].ID != "filetree" || got[0].Label != "filetree" {
		t.Fatalf("first should be filetree, got %#v", got[0])
	}
	if got[1].ID != "download" {
		t.Fatalf("second should be download, got %#v", got[1])
	}
	if got[2].ID != "help" {
		t.Fatalf("third should be help, got %#v", got[2])
	}
	if got[3].ID != "current" || got[3].Label != "current" {
		t.Fatalf("fourth should be current, got %#v", got[3])
	}
	if got[4].ID != "all" || got[4].Label != "likes" {
		t.Fatalf("all should display as likes, got %#v", got[4])
	}
	if got[5].ID != "mixes" || got[5].Label != "Mixes" {
		t.Fatalf("mixes should display as Mixes, got %#v", got[5])
	}
	if retainNavIdx(got, "") != 3 {
		t.Fatalf("empty prev should land on current, got %d", retainNavIdx(got, ""))
	}
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
	got := lipglossStrip(m.renderBrowse(10))
	if !strings.Contains(got, "2008/") {
		t.Fatalf("want folder row, got %q", got)
	}
}

func TestViewFreezeKeepsCachedFrame(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "current", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "current")
	first := m.View()
	if m.frames.view == "" || m.frames.vizRow < 1 || m.frames.vizW < 8 {
		t.Fatalf("full view should cache viz geom, row=%d w=%d", m.frames.vizRow, m.frames.vizW)
	}
	m.wavePeaks = []int{255, 80, 20}
	m.frames.freeze = true
	second := m.View()
	if first != second {
		t.Fatal("frozen view should return the last full frame")
	}
}

func TestLiveStatusFreezesView(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "current", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "current")
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

func TestVizPainterStartsOnPlay(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "current", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "current")
	_ = m.View()
	m.status.State = "playing"
	m.syncViz()
	if !m.paint.running() {
		t.Fatal("playing should start the vis painter")
	}
	m.status.State = "paused"
	m.syncViz()
	if m.paint.running() {
		t.Fatal("pause should stop the vis painter")
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

func TestPlayingTrackRowIsGreen(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 2, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "all")
	m.listIdx = 1
	m.filtered = []library.Track{
		{Path: "/tmp/x.mp3", Title: "World Series", Artist: "Sound Power", Duration: 201, Year: "2004", Genre: "Grime"},
		{Path: "/tmp/y.mp3", Title: "Fabriclive 36", Artist: "DJ Hype", Duration: 369, Year: "2006", Genre: "Drum & Bass"},
	}
	m.status.Path = "/tmp/x.mp3"
	got := m.renderTracks(5)
	plain := lipglossStrip(got)
	if !strings.Contains(plain, "▶") {
		t.Fatalf("playing row should keep the play marker, got %q", plain)
	}
	green := "38;2;134;239;172"
	if !strings.Contains(got, green) {
		t.Fatalf("playing row should use active green, got %q", got)
	}
	lines := strings.Split(got, "\n")
	if len(lines) < 1 || !strings.Contains(lines[0], green) {
		t.Fatalf("playing title row should be green, got %q", got)
	}
	row := lipglossStrip(lines[0])
	if !strings.Contains(row, "Sound Power") {
		t.Fatalf("missing playing title: %q", row)
	}
	if strings.Count(lines[0], green) < 2 {
		t.Fatalf("name and meta should both be green, got %q", lines[0])
	}
}

func TestIdleTrackNamesAreWhite(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "all", Count: 2, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "all")
	m.listIdx = 0
	m.filtered = []library.Track{
		{Path: "/tmp/x.mp3", Title: "World Series", Artist: "Sound Power", Duration: 201, Year: "2004", Genre: "Grime"},
		{Path: "/tmp/y.mp3", Title: "Fabriclive 36", Artist: "DJ Hype", Duration: 369, Year: "2006", Genre: "Drum & Bass"},
	}
	got := m.renderTracks(5)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("want two rows, got %q", got)
	}
	white := "38;2;255;255;255"
	if !strings.Contains(lines[1], white) {
		t.Fatalf("idle track name should be white, got %q", lines[1])
	}
	if !strings.Contains(lipglossStrip(lines[1]), "DJ Hype") {
		t.Fatalf("missing idle title: %q", lipglossStrip(lines[1]))
	}
}

func TestCurrentPaneHasTopGap(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex([]playlist.IndexItem{{Name: "current", Count: 1, Kind: "system"}})
	m.navIdx = navIndex(m.nav, "current")
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "World Series"
	got, _ := m.renderListWithArt(20)
	lines := strings.Split(lipglossStrip(got), "\n")
	idx := -1
	for i, line := range lines {
		if strings.Contains(line, "Now playing") {
			idx = i
			break
		}
	}
	if idx < 0 || idx+2 >= len(lines) {
		t.Fatalf("missing now playing legend, got %q", strings.Join(lines, "\n"))
	}
	gap, title := lines[idx+1], lines[idx+2]
	if strings.Contains(gap, "nothing playing") || strings.Contains(gap, "World Series") {
		t.Fatalf("title should not sit flush under the legend, got %q", gap)
	}
	if !strings.Contains(title, "World Series") && !strings.Contains(title, "nothing playing") {
		t.Fatalf("title should follow the gap, got %q", title)
	}
}

func TestNowPlayingPillsShareArtistRow(t *testing.T) {
	m := newModel(paths.Env{})
	m.status.Path = "/tmp/x.mp3"
	m.status.Title = "50000 Watts Loefah Remix"
	m.status.Artist = "Matty G"
	m.status.Album = "ARG009"
	m.status.Label = "Arg"
	m.status.Genre = "Dubstep"
	m.status.Year = "2007"
	m.status.DurationLabel = "5:22"
	got := m.renderNowPlayingHead(72)
	plain := lipglossStrip(got)
	if strings.Contains(plain, "\n") {
		t.Fatalf("artist and labels should share one row, got %q", plain)
	}
	artistAt := strings.Index(plain, "Matty G")
	pillAt := strings.Index(plain, "Dubstep")
	if artistAt < 0 || pillAt < 0 || artistAt > pillAt {
		t.Fatalf("artist should sit left of labels, got %q", plain)
	}
}

func TestFooterHintsHaveBorderTails(t *testing.T) {
	m := newModel(paths.Env{})
	got := lipglossStrip(m.renderFooterHints())
	if strings.Contains(got, "──") {
		t.Fatalf("hints should not use border separators, got %q", got)
	}
	if !strings.Contains(got, "browse") || !strings.Contains(got, "play") {
		t.Fatalf("want help labels, got %q", got)
	}
}

func TestFooterLegendHasTime(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.status.PositionLabel = "1:47"
	m.status.DurationLabel = "4:25"
	footer := lipglossStrip(m.renderFooter())
	lines := strings.Split(footer, "\n")
	if len(lines) < 2 {
		t.Fatalf("want a fieldset, got %q", footer)
	}
	if !strings.Contains(lines[0], "1:47/4:25") {
		t.Fatalf("time should be on the top legend, got %q", lines[0])
	}
	if strings.Contains(footer, "VOL ") || strings.Contains(footer, "▎") {
		t.Fatalf("volume indicator should be gone, got %q", footer)
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
