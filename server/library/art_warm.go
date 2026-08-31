package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func artPathFolder(env Env, path string) string {
	return filepath.Join(env.ArtDir, artFolderKey(env.MusicRoot, path)+".jpg")
}

func artPathLegacy(env Env, path string) string {
	return filepath.Join(env.ArtDir, cacheKey(env.MusicRoot, path)+".jpg")
}

func artPathContent(env Env, hash string) string {
	return filepath.Join(env.ArtDir, hash+".jpg")
}

func artImageHash(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func artLinkFolderAlias(folderPath, contentPath string) error {
	if st, err := os.Stat(contentPath); err != nil || st.IsDir() {
		return fmt.Errorf("missing art content")
	}
	if st, err := os.Stat(folderPath); err == nil && !st.IsDir() {
		if sameFile(folderPath, contentPath) {
			return nil
		}
		_ = os.Remove(folderPath)
	}
	if err := os.Link(contentPath, folderPath); err == nil {
		return nil
	}
	in, err := os.Open(contentPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(folderPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func sameFile(a, b string) bool {
	sa, err := os.Stat(a)
	if err != nil {
		return false
	}
	sb, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(sa, sb)
}

// EnsureArt extracts or resolves cached album art for a track.
func EnsureArt(env Env, path string) (string, bool, error) {
	if path == "" {
		return "", false, os.ErrNotExist
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return "", false, os.ErrNotExist
	}
	if existing := artCacheFind(env, path); existing != "" {
		return existing, false, nil
	}
	if err := os.MkdirAll(env.ArtDir, 0o755); err != nil {
		return "", false, err
	}
	trackArt := artPathLegacy(env, path)
	if isDecodableArtFile(trackArt) {
		return trackArt, false, nil
	}
	tmp, err := os.CreateTemp(env.ArtDir, ".art.*.jpg")
	if err != nil {
		return "", false, err
	}
	tmpName := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpName)
	if err := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", path,
		"-an", "-vcodec", "copy", tmpName).Run(); err != nil {
		return "", false, nil
	}
	if !isDecodableArtFile(tmpName) {
		return "", false, nil
	}
	hash, err := artImageHash(tmpName)
	if err != nil || hash == "" {
		return "", false, err
	}
	content := artPathContent(env, hash)
	if !isDecodableArtFile(content) {
		if err := os.Rename(tmpName, content); err != nil {
			return "", false, err
		}
	} else {
		_ = os.Remove(tmpName)
	}
	if err := artLinkFolderAlias(trackArt, content); err != nil {
		return "", false, err
	}
	if isDecodableArtFile(trackArt) {
		return trackArt, true, nil
	}
	return "", false, nil
}
