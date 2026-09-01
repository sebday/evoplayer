package library

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sebday/evoplayer/server/config"
)

type genreConfig struct {
	aliases map[string]string
	folders map[string]string
}

var genreConfigCache sync.Map // config path -> genreConfig

func genreConfigFor(env Env) genreConfig {
	path := env.MusicConfig
	if cached, ok := genreConfigCache.Load(path); ok {
		return cached.(genreConfig)
	}
	aliases, err := config.GenreAliases(path)
	if err != nil {
		aliases = config.DefaultGenreAliases()
	}
	folders, err := config.GenreFolders(path)
	if err != nil {
		folders = config.DefaultGenreFolders()
	}
	cfg := genreConfig{aliases: aliases, folders: folders}
	genreConfigCache.Store(path, cfg)
	return cfg
}

func canonicalGenreKey(name string, aliases map[string]string) string {
	norm := NormalizeGenreKey(name)
	if norm == "" {
		return ""
	}
	if canon, ok := aliases[norm]; ok {
		return canon
	}
	return norm
}

func folderCanonicalKey(folderName string, aliases map[string]string) string {
	return canonicalGenreKey(folderName, aliases)
}

// MatchLibraryGenre returns the first library folder matching any candidate genre
// or tag via direct folder name or canonical genre alias resolution.
func MatchLibraryGenre(env Env, names ...string) string {
	for _, name := range names {
		if folder := matchLibraryFolderExact(env, name); folder != "" {
			return folder
		}
	}
	cfg := genreConfigFor(env)
	canon := ""
	for _, name := range names {
		if canon = canonicalGenreKey(name, cfg.aliases); canon != "" {
			break
		}
	}
	if canon == "" {
		return ""
	}
	if folder := cfg.folders[canon]; folder != "" {
		if dirExists(filepath.Join(env.MusicRoot, folder)) {
			return folder
		}
	}
	for _, folder := range GenreChoices(env) {
		if folderCanonicalKey(folder, cfg.aliases) == canon {
			return folder
		}
	}
	return ""
}

func matchLibraryFolderExact(env Env, name string) string {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return ""
	}
	for _, folder := range GenreChoices(env) {
		if strings.ToLower(folder) == want {
			return folder
		}
	}
	return ""
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
