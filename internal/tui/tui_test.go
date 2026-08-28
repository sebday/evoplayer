package tui

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/playback"
	"github.com/sebday/evoplayer/internal/playlist"
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

func TestNavFromIndexAddsSettings(t *testing.T) {
	got := navFromIndex([]playlist.IndexItem{
		{Name: "all", Count: 3, Kind: "system"},
		{Name: "mixes", Count: 1, Kind: "system"},
	})
	if got[0].ID != "nowplaying" || got[0].Label != "Now playing" {
		t.Fatalf("first should be Now playing, got %#v", got[0])
	}
	if got[1].ID != "filetree" || got[1].Label != "Filetree" {
		t.Fatalf("second should be Filetree, got %#v", got[1])
	}
	if got[2].ID != "current" || got[2].Label != "Current" {
		t.Fatalf("third should be Current, got %#v", got[2])
	}
	if got[3].ID != "all" || got[3].Label != "Likes" {
		t.Fatalf("all should display as Likes, got %#v", got[3])
	}
	if got[4].ID != "mixes" || got[4].Label != "Mixes" {
		t.Fatalf("mixes should display as Mixes, got %#v", got[4])
	}
	last := got[len(got)-1]
	if last.ID != "help" || last.Label != "Help" {
		t.Fatalf("last = %#v", last)
	}
	if got[len(got)-2].ID != "settings" {
		t.Fatalf("settings should sit above Help, got %#v", got[len(got)-2])
	}
}

func TestTrackLabel(t *testing.T) {
	if got := trackLabel(library.Track{Artist: "A", Title: "B"}); got != "A — B" {
		t.Fatalf("got %q", got)
	}
}

func TestParentPath(t *testing.T) {
	if got := parentPath("grime/2008"); got != "grime" {
		t.Fatalf("got %q", got)
	}
	if got := parentPath("grime"); got != "" {
		t.Fatalf("root parent = %q", got)
	}
}

func TestFilterBrowseKeepsParent(t *testing.T) {
	entries := []library.BrowseEntry{
		{Type: "parent", Name: ".."},
		{Type: "dir", Name: "2008"},
		{Type: "track", Track: library.Track{Title: "Wot Do U Call It", Artist: "Wiley"}},
	}
	got := filterBrowse(entries, "wiley")
	if len(got) != 2 || got[0].Type != "parent" || got[1].Title != "Wot Do U Call It" {
		t.Fatalf("got %#v", got)
	}
}

func TestFieldsetPutsLegendInBorder(t *testing.T) {
	got := fieldset("Search", "query", 24, 5, false)
	first := strings.Split(got, "\n")[0]
	if !strings.Contains(first, "Search") || !strings.Contains(first, "╭") || !strings.Contains(first, "─") {
		t.Fatalf("legend should sit in the top border, got %q", first)
	}
	if !strings.Contains(got, "╰") {
		t.Fatal("missing bottom border")
	}
}

func TestNavHasNoFieldset(t *testing.T) {
	m := newModel(paths.Env{})
	m.nav = []navItem{{ID: "all", Label: "Likes", Kind: "system", Count: 3}}
	got := m.renderNav(12)
	if strings.Contains(got, "╭") || strings.Contains(got, "╮") || strings.Contains(got, "╰") {
		t.Fatalf("nav should be unboxed, got %q", got)
	}
	if !strings.Contains(got, "Likes") {
		t.Fatalf("missing label: %q", got)
	}
}

func TestFramePad(t *testing.T) {
	small := model{width: 80, height: 24}
	if x, y := small.framePad(); x != 0 || y != 0 {
		t.Fatalf("small pad = %d,%d", x, y)
	}
	big := model{width: 120, height: 40}
	if x, y := big.framePad(); x != 6 || y != 1 {
		t.Fatalf("big pad = %d,%d", x, y)
	}
}

func TestMoveListKeepsSelectionVisible(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	vis := m.listVisible()
	if vis < 3 {
		t.Fatalf("visible rows = %d", vis)
	}
	m.filtered = make([]library.Track, vis+8)
	for i := 0; i < vis+2; i++ {
		m.moveList(1)
		if m.listIdx < m.listOffset || m.listIdx >= m.listOffset+vis {
			t.Fatalf("idx %d outside window [%d, %d)", m.listIdx, m.listOffset, m.listOffset+vis)
		}
	}
}

func TestDownsamplePeaksTakesBucketMax(t *testing.T) {
	got := downsamplePeaks([]int{0, 10, 255, 1}, 2)
	if len(got) != 2 || got[0] != 10 || got[1] != 255 {
		t.Fatalf("got %#v", got)
	}
	if downsamplePeaks(nil, 8) != nil {
		t.Fatal("empty data should stay empty")
	}
}

func TestPlayheadIndex(t *testing.T) {
	if got := playheadIndex(30, 100, 10); got != 3 {
		t.Fatalf("got %d", got)
	}
	if got := playheadIndex(0, 100, 10); got != 0 {
		t.Fatalf("start = %d", got)
	}
	if got := playheadIndex(100, 100, 10); got != 9 {
		t.Fatalf("end = %d", got)
	}
}

func TestRenderWaveformUsesBlocks(t *testing.T) {
	got := renderWaveform([]int{0, 255, 64}, 1)
	plain := lipglossStrip(got)
	lines := strings.Split(plain, "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 rows, got %q", plain)
	}
	if !strings.Contains(plain, "█") && !strings.Contains(plain, "▄") {
		t.Fatalf("want amplitude blocks, got %q", plain)
	}
}

func TestRenderArtHalfBlocks(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 196, G: 181, B: 253, A: 255})
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "art.jpg")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	got := renderArt(loadArt(path), 6, 3)
	if !strings.Contains(got, "▀") {
		t.Fatalf("want half-blocks, got %q", got)
	}
	if !strings.Contains(got, "38;2;") {
		t.Fatalf("want truecolor art, got %q", got)
	}
}

func TestNowPlayingShowsSearch(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = 0
	m.status = playback.Status{State: "stopped"}
	got := m.View()
	if !strings.Contains(got, "Search") {
		t.Fatalf("now playing should show search, got %q", got)
	}
	if !strings.Contains(got, "Now playing") {
		t.Fatalf("missing now playing legend: %q", got)
	}
	if !strings.Contains(got, "nothing playing") {
		t.Fatalf("empty state missing: %q", got)
	}
}

func TestHelpListsKeysAndHidesSearch(t *testing.T) {
	m := newModel(paths.Env{})
	m.width = 120
	m.height = 40
	m.ready = true
	m.nav = navFromIndex(nil)
	m.navIdx = len(m.nav) - 1
	got := m.View()
	if strings.Contains(got, "Search") {
		t.Fatalf("help should hide search, got %q", got)
	}
	plain := lipglossStrip(got)
	if !strings.Contains(plain, "Help") {
		t.Fatalf("missing help legend: %q", plain)
	}
	if !strings.Contains(plain, "space") || !strings.Contains(plain, "like") {
		t.Fatalf("missing keys: %q", plain)
	}
	nm, _ := m.openHelp()
	hm := nm.(model)
	if !hm.helpSelected() {
		t.Fatal("? should select help")
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
