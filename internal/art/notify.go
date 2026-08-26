package art

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/paths"
)

func NotifyCache(env paths.Env, trackPath string) (string, error) {
	libEnv := library.EnvFrom(env)
	art, _, err := library.EnsureArt(libEnv, trackPath)
	if err != nil || art == "" {
		art = library.ResolveArtPath(libEnv, trackPath)
	}
	if art == "" {
		return "", os.ErrNotExist
	}
	hash, err := fileHash(art)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(env.DisplayArtDir, hash+".jpg")
	if err := os.MkdirAll(env.DisplayArtDir, 0o755); err != nil {
		return "", err
	}
	if st, err := os.Stat(dest); err == nil && !st.IsDir() {
		if sameFileBytes(art, dest) {
			return dest, nil
		}
		_ = os.Remove(dest)
	}
	tmp := dest + ".tmp"
	if err := copyFile(art, tmp); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
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

func sameFileBytes(a, b string) bool {
	ab, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	bb, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ab, bb)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
