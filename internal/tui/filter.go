package tui

import (
	"strings"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/playlist"
)

type navItem struct {
	ID    string
	Label string
	Kind  string
	Count int
}

func navFromIndex(items []playlist.IndexItem) []navItem {
	out := make([]navItem, 0, len(items)+5)
	out = append(out, navItem{ID: "nowplaying", Label: "Now playing", Kind: "nowplaying"})
	out = append(out, navItem{ID: "filetree", Label: "Filetree", Kind: "filetree"})
	current := navItem{ID: "current", Label: "Current", Kind: "system"}
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
		label := it.Name
		switch it.Name {
		case "all":
			label = "Likes"
		case "mixes":
			label = "Mixes"
		}
		out = append(out, navItem{
			ID:    it.Name,
			Label: label,
			Kind:  it.Kind,
			Count: it.Count,
		})
	}
	out = append(out, navItem{ID: "settings", Label: "Settings", Kind: "settings"})
	out = append(out, navItem{ID: "help", Label: "Help", Kind: "help"})
	return out
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
		if e.Type == "parent" {
			out = append(out, e)
			continue
		}
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
