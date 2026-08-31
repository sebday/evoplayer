package library

import "strings"

// MatchLibraryGenre returns the first library folder name matching any candidate
// genre or tag (case-insensitive; "&" and spaces ignored).
func MatchLibraryGenre(env Env, names ...string) string {
	for _, name := range names {
		if folder := matchLibraryFolder(env, name); folder != "" {
			return folder
		}
	}
	wantSeen := map[string]bool{}
	for _, name := range names {
		want := NormalizeGenreKey(name)
		if want == "" || wantSeen[want] {
			continue
		}
		wantSeen[want] = true
		for _, folder := range GenreChoices(env) {
			if NormalizeGenreKey(folder) == want {
				return folder
			}
		}
	}
	return ""
}

// NormalizeGenreKey folds genre names for fuzzy folder matching.
func NormalizeGenreKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "&", " and ")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
