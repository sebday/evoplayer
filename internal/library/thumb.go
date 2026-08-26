package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const thumbSize = 72

func thumbDir(env Env) string {
	if env.CacheDir != "" {
		return filepath.Join(env.CacheDir, "thumbs")
	}
	return ""
}

func thumbKey(artPath string) string {
	st, err := os.Stat(artPath)
	if err != nil {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(artPath))
	fmt.Fprintf(h, "\n%d\n%d", st.ModTime().UnixNano(), st.Size())
	return hex.EncodeToString(h.Sum(nil))[:20]
}

func thumbPathForArt(env Env, artPath string) string {
	key := thumbKey(artPath)
	if key == "" {
		return ""
	}
	return filepath.Join(thumbDir(env), key+".jpg")
}

func resolveThumb(env Env, artPath string) string {
	if artPath == "" || !isDecodableArtFile(artPath) {
		return ""
	}
	dest := thumbPathForArt(env, artPath)
	if dest != "" && isDecodableArtFile(dest) {
		return dest
	}
	return ""
}

func EnsureThumb(env Env, artPath string) (string, error) {
	if artPath == "" || !isDecodableArtFile(artPath) {
		return "", os.ErrNotExist
	}
	dir := thumbDir(env)
	if dir == "" {
		return "", fmt.Errorf("thumb cache dir unavailable")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := thumbPathForArt(env, artPath)
	if isDecodableArtFile(dest) {
		return dest, nil
	}
	tmp := dest + ".tmp"
	filter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", thumbSize, thumbSize, thumbSize, thumbSize)
	if err := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", artPath,
		"-vf", filter, "-q:v", "4", tmp).Run(); err != nil {
		if err := thumbWithMagick(artPath, tmp); err != nil {
			return "", err
		}
	}
	if !isDecodableArtFile(tmp) {
		os.Remove(tmp)
		return "", fmt.Errorf("thumb generation failed")
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

func thumbWithMagick(src, dest string) error {
	return exec.Command("magick", src,
		"-thumbnail", fmt.Sprintf("%dx%d^", thumbSize, thumbSize),
		"-gravity", "center",
		"-extent", fmt.Sprintf("%dx%d", thumbSize, thumbSize),
		"-quality", "85", dest).Run()
}

func enrichTrackAssets(env Env, row *Track) {
	if row == nil {
		return
	}
	row.Art = resolveArt(env, row.Path, row.Art)
	row.Waveform = resolveWaveform(env, row.Path, row.Waveform)
	if row.Art != "" {
		row.Thumb = resolveThumb(env, row.Art)
	}
}
