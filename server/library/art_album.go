package library

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/sebday/evoplayer/server/tags"
)

// dumpFolderNames are collection directories. Their name is not an album,
// even when every file inside happens to carry the same album tag.
var dumpFolderNames = map[string]struct{}{
	"soundcloud": {},
}

func isDumpFolder(dir string) bool {
	base := strings.ToLower(filepath.Base(dir))
	_, ok := dumpFolderNames[base]
	return ok
}

// directoryIsAlbum reports whether dir should share one cover.
// The album tag has to be set on every audio file, and the directory itself
// must not be a dump folder such as soundcloud.
func directoryIsAlbum(dir string, albumOf func(string) string) bool {
	if isDumpFolder(dir) || albumOf == nil {
		return false
	}
	files := dirAudioPaths(dir)
	if len(files) == 0 {
		return false
	}
	album := strings.TrimSpace(albumOf(files[0]))
	if album == "" {
		return false
	}
	shared := pathsSharingAlbum(files[0], album, files, albumOf)
	return coversPaths(shared, files)
}

var (
	folderArtMu    sync.Mutex
	folderArtCache = map[string]bool{}
)

func folderArtAllowed(env Env, path string) bool {
	dir := filepath.Clean(filepath.Dir(path))
	folderArtMu.Lock()
	if allowed, ok := folderArtCache[dir]; ok {
		folderArtMu.Unlock()
		return allowed
	}
	folderArtMu.Unlock()
	known := albumsInDir(env, dir)
	allowed := directoryIsAlbum(dir, func(p string) string {
		if v, ok := known[p]; ok && strings.TrimSpace(v) != "" {
			return v
		}
		info, err := tags.ReadTags(p)
		if err != nil {
			return ""
		}
		return info.Album
	})
	folderArtMu.Lock()
	folderArtCache[dir] = allowed
	folderArtMu.Unlock()
	return allowed
}

func forgetFolderArt(dir string) {
	folderArtMu.Lock()
	delete(folderArtCache, filepath.Clean(dir))
	folderArtMu.Unlock()
}
