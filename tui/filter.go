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
		{ID: "filetree", Label: "filetree", Kind: "filetree"},
		{ID: "download", Label: "download", Kind: "download"},
		{ID: "help", Label: "Help", Kind: "help"},
	}
	current := navItem{ID: "current", Label: "current", Kind: "system"}
	rest := make([]playlist.IndexItem, 0, len(items))
	for _, it := range items {
		if it.Name == "current" {
			current.Count = it.Count
			continue
		}
		rest = append(rest, it)
	}
	out = append(out, current)
	for _, it := range rest {
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
		return "Mixes"
	case "current":
		return "current"
	default:
		return name
	}
}

func navIsIcon(item navItem) bool {
	switch item.Kind {
	case "filetree", "download", "help":
		return true
	default:
		return false
	}
}

func navOnMenu(item navItem) bool {
	return item.ID != "help"
}

func navIsStatic(id string) bool {
	switch id {
	case "download", "help":
		return true
	default:
		return false
	}
}

func navGlyph(item navItem) string {
	switch item.ID {
	case "filetree":
		return "⌂ files"
	case "download":
		return "⬇ download"
	default:
		return ""
	}
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
	if i := navIndex(nav, "current"); i >= 0 {
		return i
	}
	if len(nav) == 0 {
		return 0
	}
	return 0
}

func filterTracks(tracks []library.Track, query string) []library.Track {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return tracks
	}
	out := make([]library.Track, 0, len(tracks))
	for _, t := range tracks {
		hay := strings.ToLower(t.Artist + " " + t.Title + " " + t.Album + " " + t.Genre)
		if strings.Contains(hay, q) {
			out = append(out, t)
		}
	}
	return out
}

func filterBrowse(entries []library.BrowseEntry, query string) []library.BrowseEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return entries
	}
	out := make([]library.BrowseEntry, 0, len(entries))
	for _, e := range entries {
		hay := strings.ToLower(e.Name + " " + e.Artist + " " + e.Title + " " + e.Album)
		if strings.Contains(hay, q) {
			out = append(out, e)
		}
	}
	return out
}

func browseLabel(e library.BrowseEntry) string {
	switch e.Type {
	case "parent":
		return ".."
	case "dir":
		name := strings.TrimSpace(e.Name)
		if name == "" {
			name = e.Path
		}
		return name + "/"
	default:
		return trackLabel(e.Track)
	}
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
