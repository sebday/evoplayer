package library

import (
	"strings"

	"github.com/sebday/evoplayer/server/config"
)

// genreTagKey folds case and punctuation so a tag can match a library folder name.
func genreTagKey(name string) string {
	return config.NormalizeGenreKey(strings.TrimSpace(name))
}

// MatchLibraryGenre returns a library folder only when the tag is that folder's name.
func MatchLibraryGenre(env Env, names ...string) string {
	for _, name := range names {
		if folder := matchLibraryFolder(env, name); folder != "" {
			return folder
		}
	}
	return ""
}

func matchLibraryFolder(env Env, name string) string {
	want := genreTagKey(name)
	if want == "" {
		return ""
	}
	for _, folder := range GenreChoices(env) {
		if genreTagKey(folder) == want {
			return folder
		}
	}
	return ""
}
