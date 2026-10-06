package library

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

// SharedDumpCovers lists dump-folder tracks whose cached cover is the same
// file as another track in those folders. A cover that belongs to one track
// is left alone.
func SharedDumpCovers(env Env) (dirs []string, tracks []string, err error) {
	if env.MusicRoot == "" {
		return nil, nil, fmt.Errorf("music root is empty")
	}
	// WalkDir does not follow a symlinked library root.
	root, err := filepath.EvalSymlinks(env.MusicRoot)
	if err != nil || root == "" {
		root = env.MusicRoot
	}
	groups := map[string][]string{}
	dirSet := map[string]struct{}{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		logical := logicalMusicPath(env.MusicRoot, root, path)
		dir := filepath.Dir(logical)
		if !isDumpFolder(dir) || !playback.IsSupportedPath(logical) {
			return nil
		}
		dirSet[dir] = struct{}{}
		art := artPathTrack(env, logical)
		dev, ino, ok := fileInode(art)
		if !ok {
			return nil
		}
		key := fmt.Sprintf("%d:%d", dev, ino)
		groups[key] = append(groups[key], logical)
		return nil
	})
	dirs = make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		tracks = append(tracks, group...)
	}
	sort.Strings(tracks)
	return dirs, tracks, err
}

// ClearDumpFolderCover removes the directory-level cover for a dump folder
// so a missing track image cannot fall back to it.
func ClearDumpFolderCover(env Env, dir string) {
	if !isDumpFolder(dir) {
		return
	}
	files := dirAudioPaths(dir)
	if len(files) == 0 {
		return
	}
	_ = os.Remove(artPathFolder(env, files[0]))
	forgetFolderArt(dir)
}

// ReplaceTrackCover stores imagePath as this track's cover only, then writes
// that image back into the audio file so a later extract cannot revive a
// folder stamp.
func ReplaceTrackCover(env Env, trackPath, imagePath string) error {
	if _, err := InstallImage(env, trackPath, imagePath, "track"); err != nil {
		return err
	}
	art := artPathTrack(env, trackPath)
	if err := embedAudio(trackPath, art); err != nil {
		return err
	}
	_, _ = EnsureThumb(env, art)
	return nil
}

// StripTrackCover removes a dumped cover from one audio file and its cache
// alias. The shared image inode is left in place for any track still linked
// to it.
func StripTrackCover(env Env, trackPath string) error {
	if err := stripEmbeddedArt(trackPath); err != nil {
		return err
	}
	_ = os.Remove(artPathTrack(env, trackPath))
	rememberTrackArt(env, trackPath, "")
	forgetFolderArt(filepath.Dir(trackPath))
	return nil
}

func stripEmbeddedArt(audioPath string) error {
	st, err := os.Stat(audioPath)
	if err != nil {
		return err
	}
	ext := filepath.Ext(audioPath)
	f, err := os.CreateTemp(filepath.Dir(audioPath), ".arttmp-*"+ext)
	if err != nil {
		return err
	}
	tmp := f.Name()
	f.Close()
	args := []string{"-y", "-loglevel", "error", "-i", audioPath, "-map", "0:a", "-map_metadata", "0", "-c:a", "copy"}
	if strings.EqualFold(ext, ".mp3") {
		args = append(args, "-id3v2_version", "3")
	}
	args = append(args, tmp)
	err = exec.Command("ffmpeg", args...).Run()
	if err == nil {
		err = os.Chmod(tmp, st.Mode().Perm())
	}
	if err == nil {
		err = os.Rename(tmp, audioPath)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if strings.EqualFold(ext, ".mp3") {
		if err := tags.ClearMP3Picture(audioPath); err != nil {
			return err
		}
	}
	return nil
}

func logicalMusicPath(musicRoot, resolvedRoot, path string) string {
	if resolvedRoot == "" || musicRoot == "" {
		return path
	}
	rel, err := filepath.Rel(resolvedRoot, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.Join(musicRoot, rel)
}

func fileInode(path string) (dev, ino uint64, ok bool) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, 0, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		return 0, 0, false
	}
	return uint64(st.Dev), uint64(st.Ino), true
}
