package tui

import (
	"strings"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/playlist"
)

type navItem struct {
	ID    string
	Label string
	Kind  string
	Count int
}

func navFromIndex(items []playlist.IndexItem) []navItem {
	out := []navItem{
		{ID: "filetree", Label: "browse", Kind: "filetree"},
		{ID: "settings", Label: "settings", Kind: "settings"},
		{ID: "download", Label: "download", Kind: "download"},
		{ID: "help", Label: "Help", Kind: "help"},
	}
	for _, it := range items {
		out = append(out, navItem{
			ID:    it.Name,
			Label: playlistTabLabel(it.Name),
			Kind:  it.Kind,
			Count: it.Count,
		})
	}
	return out
}

func playlistTabLabel(name string) string {
	switch name {
	case "all":
		return "likes"
	case "mixes":
		return "mixes"
	case "current":
		return "now playing"
	default:
		return name
	}
}

func navIsIcon(item navItem) bool {
	switch item.Kind {
	case "filetree", "settings", "download", "help":
		return true
	default:
		return false
	}
}

func navIsStatic(id string) bool {
	switch id {
	case "settings", "download", "help":
		return true
	default:
		return false
	}
}

func toolNavItems() []navItem {
	return []navItem{
		{ID: "settings", Label: "settings", Kind: "settings"},
		{ID: "download", Label: "download", Kind: "download"},
	}
}

func (m model) sidebarTools() []navItem {
	if m.searching() {
		return nil
	}
	return toolNavItems()
}

func (m model) playlistSlotCount() int {
	if m.searching() {
		return 0
	}
	if m.playlistsPending() {
		return 1
	}
	return len(m.sidebarPlaylists())
}

func (m model) sidebarPlaylists() []navItem {
	if m.searching() {
		return nil
	}
	var current *navItem
	rest := make([]navItem, 0, len(m.nav))
	for _, it := range m.nav {
		if navIsIcon(it) {
			continue
		}
		if it.ID == "current" {
			item := it
			current = &item
			continue
		}
		rest = append(rest, it)
	}
	if current == nil {
		return rest
	}
	return append([]navItem{*current}, rest...)
}

func (m model) playlistsPending() bool {
	if m.searching() {
		return false
	}
	return len(m.nav) == 0
}

func navIndex(nav []navItem, id string) int {
	for i, item := range nav {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func retainNavIdx(nav []navItem, prev string) int {
	if prev != "" && prev != "nowplaying" {
		if i := navIndex(nav, prev); i >= 0 {
			return i
		}
	}
	if i := navIndex(nav, "filetree"); i >= 0 {
		return i
	}
	if len(nav) == 0 {
		return 0
	}
	return 0
}

func tracksToBrowse(tracks []library.Track) []library.BrowseEntry {
	out := make([]library.BrowseEntry, 0, len(tracks))
	for _, t := range tracks {
		if t.Path == "" {
			continue
		}
		out = append(out, library.BrowseEntry{Type: "track", Name: t.Title, Track: t})
	}
	return out
}

func browseTracks(entries []library.BrowseEntry) []library.Track {
	out := make([]library.Track, 0, len(entries))
	for _, e := range entries {
		if e.Type == "track" && e.Path != "" {
			out = append(out, e.Track)
		}
	}
	return out
}

func browseLabel(e library.BrowseEntry) string {
	switch e.Type {
	case "dir":
		name := strings.TrimSpace(e.Name)
		if name == "" {
			name = e.Path
		}
		return name + "/"
	case "track":
		return browseTrackName(e)
	default:
		return trackLabel(e.Track)
	}
}

func browseTrackName(e library.BrowseEntry) string {
	if title := strings.TrimSpace(e.Title); title != "" {
		return title
	}
	return trackLabel(e.Track)
}

func parentPath(rel string) string {
	rel = strings.Trim(strings.TrimPrefix(rel, "/"), "/")
	i := strings.LastIndex(rel, "/")
	if i <= 0 {
		return ""
	}
	return rel[:i]
}

func trackLabel(t library.Track) string {
	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = t.Path
	}
	artist := strings.TrimSpace(t.Artist)
	if artist == "" {
		return title
	}
	return artist + " — " + title
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
