package playlist

import (
	"sort"
	"strings"
)

// sortPlaylists orders tiles A–Z by the name shown in the grid.
// Likes and Mixes use those titles, not the stored ids "all" and "mixes".
func sortPlaylists(items []IndexItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a := playlistSortKey(items[i].Name)
		b := playlistSortKey(items[j].Name)
		if a != b {
			return a < b
		}
		return items[i].Name < items[j].Name
	})
}

func playlistSortKey(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "all":
		return "likes"
	case "current":
		return "now playing"
	case "mixes":
		return "mixes"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}
