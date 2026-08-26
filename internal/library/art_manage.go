package library

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/internal/audio"
	"github.com/sebday/evoplayer/internal/playback"
)

// ResolveArtPath returns cached art for a track without extracting.
func ResolveArtPath(env Env, path string) string {
	return artCacheFind(env, path)
}

type InstallResult struct {
	Art     string `json:"art"`
	Track   string `json:"track,omitempty"`
	Folder  string `json:"folder,omitempty"`
	Content string `json:"content,omitempty"`
	Scope   string `json:"scope"`
}

func InstallImage(env Env, trackPath, imagePath, scope string) (InstallResult, error) {
	if scope != "album" && scope != "track" {
		scope = "track"
	}
	if !audio.IsAudio(trackPath) {
		return InstallResult{}, fmt.Errorf("not an audio file")
	}
	st, err := os.Stat(imagePath)
	if err != nil || st.IsDir() {
		return InstallResult{}, fmt.Errorf("not an image file")
	}
	if err := os.MkdirAll(env.ArtDir, 0o755); err != nil {
		return InstallResult{}, err
	}
	destArt := artPathFolder(env, trackPath)
	if scope == "track" {
		destArt = artPathLegacy(env, trackPath)
	}
	tmp := filepath.Join(env.ArtDir, fmt.Sprintf(".install.%d.jpg", time.Now().UnixNano()))
	if err := normalizeJPG(imagePath, tmp); err != nil {
		return InstallResult{}, err
	}
	defer os.Remove(tmp)
	hash, err := artImageHash(tmp)
	if err != nil {
		return InstallResult{}, err
	}
	content := artPathContent(env, hash)
	if !isDecodableArtFile(content) {
		if err := os.Rename(tmp, content); err != nil {
			return InstallResult{}, err
		}
	} else {
		_ = os.Remove(tmp)
	}
	if err := artLinkFolderAlias(destArt, content); err != nil {
		return InstallResult{}, err
	}
	markArtDirty(env, trackPath, scope)
	return InstallResult{
		Art:     destArt,
		Track:   artPathLegacy(env, trackPath),
		Folder:  artPathFolder(env, trackPath),
		Content: content,
		Scope:   scope,
	}, nil
}

func ApplyImageURL(env Env, trackPath, imageURL, scope string) (InstallResult, error) {
	tmp := filepath.Join(env.ArtDir, fmt.Sprintf(".fetch.%d.jpg", time.Now().UnixNano()))
	out, err := os.Create(tmp)
	if err != nil {
		return InstallResult{}, err
	}
	resp, err := http.Get(imageURL)
	if err != nil {
		out.Close()
		os.Remove(tmp)
		return InstallResult{}, err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmp)
		return InstallResult{}, err
	}
	out.Close()
	res, err := InstallImage(env, trackPath, tmp, scope)
	os.Remove(tmp)
	return res, err
}

func ClearArt(env Env, trackPath string) error {
	legacy := artPathLegacy(env, trackPath)
	folder := artPathFolder(env, trackPath)
	if legacy != folder && isDecodableArtFile(legacy) {
		_ = os.Remove(legacy)
	} else if isDecodableArtFile(folder) {
		_ = os.Remove(folder)
	}
	markArtDirty(env, trackPath, "track")
	return nil
}

func normalizeJPG(src, dest string) error {
	if err := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", src,
		"-vf", "scale=1200:1200:force_original_aspect_ratio=decrease",
		"-q:v", "2", dest).Run(); err == nil {
		if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
			return nil
		}
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func Maintain(env Env) error {
	snap, err := artDirtySnapshot(env)
	if err != nil {
		return err
	}
	for dir := range snap.Dirs {
		_, _, _, _ = embedFolder(env, dir)
	}
	for track := range snap.Tracks {
		art := artCacheFind(env, track)
		if art == "" {
			continue
		}
		_, _ = embedAudio(track, art)
	}
	if env.CacheDir == "" {
		return nil
	}
	return writeDirty(env, dirtySnapshot{
		Dirs:   map[string]dirtyEntry{},
		Tracks: map[string]dirtyEntry{},
	})
}

type dirtySnapshot struct {
	Dirs   map[string]dirtyEntry `json:"dirs"`
	Tracks map[string]dirtyEntry `json:"tracks"`
}

type dirtyEntry struct {
	At string `json:"at"`
}

func dirtyPath(env Env) string {
	return filepath.Join(env.CacheDir, "art-dirty.json")
}

func markArtDirty(env Env, path, scope string) {
	snap, _ := artDirtySnapshot(env)
	if snap.Dirs == nil {
		snap.Dirs = map[string]dirtyEntry{}
	}
	if snap.Tracks == nil {
		snap.Tracks = map[string]dirtyEntry{}
	}
	at := time.Now().Format(time.RFC3339)
	key := filepath.Dir(path)
	kind := snap.Dirs
	if scope == "track" || trackInGenreRoot(relUnderRoot(env.MusicRoot, path)) {
		key = path
		kind = snap.Tracks
	}
	kind[key] = dirtyEntry{At: at}
	_ = writeDirty(env, snap)
}

func artDirtySnapshot(env Env) (dirtySnapshot, error) {
	out := dirtySnapshot{Dirs: map[string]dirtyEntry{}, Tracks: map[string]dirtyEntry{}}
	raw, err := os.ReadFile(dirtyPath(env))
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func writeDirty(env Env, snap dirtySnapshot) error {
	if err := os.MkdirAll(env.CacheDir, 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return os.WriteFile(dirtyPath(env), raw, 0o644)
}

func embedFolder(env Env, dir string) (embedded, failed, skipped int, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, 0, err
	}
	var sample string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if playback.IsSupportedPath(p) {
			sample = p
			break
		}
	}
	if sample == "" {
		return 0, 0, 0, nil
	}
	art := artPathFolder(env, sample)
	if !isDecodableArtFile(art) {
		return 0, 0, 0, nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if !playback.IsSupportedPath(p) {
			continue
		}
		st, err := embedAudio(p, art)
		switch {
		case err != nil:
			failed++
		case st == 2:
			skipped++
		default:
			embedded++
		}
	}
	return embedded, failed, skipped, nil
}

func embedAudio(audioPath, artPath string) (int, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(audioPath), "."))
	tmp := audioPath + ".arttmp." + ext
	var err error
	switch ext {
	case "mp3":
		err = exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", audioPath, "-i", artPath,
			"-map", "0:a", "-map", "1:v", "-c:a", "copy", "-c:v", "mjpeg",
			"-id3v2_version", "3", tmp).Run()
	default:
		err = exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", audioPath, "-i", artPath,
			"-map", "0", "-map", "1", "-c", "copy", "-disposition:v:0", "attached_pic", tmp).Run()
	}
	if err != nil {
		os.Remove(tmp)
		return 1, err
	}
	if err := os.Rename(tmp, audioPath); err != nil {
		os.Remove(tmp)
		return 1, err
	}
	return 0, nil
}
