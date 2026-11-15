package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	coverThumbMu      sync.Mutex
	coverThumbSeen    map[string]struct{}
	coverThumbPending []string
	coverThumbRunning bool
)

const thumbSize = 72
const coverThumbSize = 256

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
	return thumbFile(env, artPath, thumbSize)
}

func coverThumbPath(env Env, artPath string) string {
	return thumbFile(env, artPath, coverThumbSize)
}

func thumbFile(env Env, artPath string, size int) string {
	key := thumbKey(artPath)
	dir := thumbDir(env)
	if key == "" || dir == "" {
		return ""
	}
	name := key + ".jpg"
	if size != thumbSize {
		name = fmt.Sprintf("%s@%d.jpg", key, size)
	}
	return filepath.Join(dir, name)
}

func resolveThumb(env Env, artPath string) string {
	return existingThumb(env, artPath, thumbSize)
}

func resolveCoverThumb(env Env, artPath string) string {
	return existingThumb(env, artPath, coverThumbSize)
}

func existingThumb(env Env, artPath string, size int) string {
	if artPath == "" || !nonEmptyFile(artPath) {
		return ""
	}
	dest := thumbFile(env, artPath, size)
	if dest != "" && nonEmptyFile(dest) {
		return dest
	}
	return ""
}

func EnsureThumb(env Env, artPath string) (string, error) {
	return ensureThumb(env, artPath, thumbSize)
}

func ensureThumb(env Env, artPath string, size int) (string, error) {
	if artPath == "" || !nonEmptyFile(artPath) {
		return "", os.ErrNotExist
	}
	dir := thumbDir(env)
	if dir == "" {
		return "", fmt.Errorf("thumb cache dir unavailable")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := thumbFile(env, artPath, size)
	if dest == "" {
		return "", fmt.Errorf("thumb cache dir unavailable")
	}
	if nonEmptyFile(dest) {
		return dest, nil
	}
	tmp := dest + ".tmp"
	filter := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", size, size, size, size)
	args := []string{"-y", "-loglevel", "error", "-i", artPath, "-vf", filter, "-q:v", "4", tmp}
	cmd := exec.Command("ffmpeg", args...)
	if size != thumbSize {
		cmd = exec.Command("nice", append([]string{"-n", "19", "ffmpeg"}, args...)...)
	}
	if err := cmd.Run(); err != nil {
		if err := thumbWithMagick(artPath, tmp, size); err != nil {
			return "", err
		}
	}
	if !nonEmptyFile(tmp) {
		os.Remove(tmp)
		return "", fmt.Errorf("thumb generation failed")
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

func thumbWithMagick(src, dest string, size int) error {
	return exec.Command("magick", src,
		"-thumbnail", fmt.Sprintf("%dx%d^", size, size),
		"-gravity", "center",
		"-extent", fmt.Sprintf("%dx%d", size, size),
		"-quality", "85", dest).Run()
}

// queueCoverThumbs builds the larger grid thumbs without blocking the index response.
func queueCoverThumbs(env Env, arts []string) {
	if len(arts) == 0 || thumbDir(env) == "" {
		return
	}
	coverThumbMu.Lock()
	if coverThumbSeen == nil {
		coverThumbSeen = map[string]struct{}{}
	}
	for _, art := range arts {
		if art == "" {
			continue
		}
		if _, ok := coverThumbSeen[art]; ok {
			continue
		}
		coverThumbSeen[art] = struct{}{}
		coverThumbPending = append(coverThumbPending, art)
	}
	start := !coverThumbRunning && len(coverThumbPending) > 0
	if start {
		coverThumbRunning = true
	}
	coverThumbMu.Unlock()
	if start {
		go runCoverThumbs(env)
	}
}

func runCoverThumbs(env Env) {
	for {
		coverThumbMu.Lock()
		if len(coverThumbPending) == 0 {
			coverThumbRunning = false
			coverThumbMu.Unlock()
			return
		}
		art := coverThumbPending[0]
		coverThumbPending = coverThumbPending[1:]
		coverThumbMu.Unlock()
		if _, err := ensureThumb(env, art, coverThumbSize); err != nil {
			coverThumbMu.Lock()
			delete(coverThumbSeen, art)
			coverThumbMu.Unlock()
		}
	}
}

func ListAssets(env Env, path, storedArt string) (art, thumb string) {
	art = resolveArt(env, path, storedArt)
	if art != "" {
		thumb = resolveThumb(env, art)
	}
	return art, thumb
}

func enrichTrackAssets(env Env, row *Track) {
	if row == nil {
		return
	}
	row.Art, row.Thumb = ListAssets(env, row.Path, row.Art)
	row.Waveform = resolveWaveform(env, row.Path, row.Waveform)
}
