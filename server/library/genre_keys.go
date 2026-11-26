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

// FolderForGenreTag returns the on-disk library folder for a genre tag.
// A tag that names a folder uses that folder. Otherwise [genre_aliases] maps
// the tag onto a canonical genre, and [genres] names the folder. The tag
// string itself is not rewritten.
func FolderForGenreTag(env Env, names ...string) string {
	if folder := MatchLibraryGenre(env, names...); folder != "" {
		return folder
	}
	if env.MusicConfig == "" {
		return ""
	}
	aliases, err := config.GenreAliases(env.MusicConfig)
	if err != nil || len(aliases) == 0 {
		return ""
	}
	folders, err := config.GenreFolders(env.MusicConfig)
	if err != nil {
		return ""
	}
	for _, name := range names {
		key := genreTagKey(name)
		if key == "" {
			continue
		}
		canon := aliases[key]
		if canon == "" {
			continue
		}
		folderName := strings.TrimSpace(folders[canon])
		if folderName == "" {
			continue
		}
		if folder := MatchLibraryGenre(env, folderName); folder != "" {
			return folder
		}
	}
	return ""
}
