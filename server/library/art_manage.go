package library

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/audio"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/tags"
)

// artCacheMax is the stored cover size. The TUI panel and Discogs preview
// are about 600px; 72px thumbs stay separate for list rows.
const artCacheMax = 600

// ResolveArtPath returns cached art for a track without extracting.
func ResolveArtPath(env Env, path string) string {
	return artCacheFind(env, path)
}

type InstallResult struct {
	Art     string   `json:"art"`
	Track   string   `json:"track,omitempty"`
	Folder  string   `json:"folder,omitempty"`
	Content string   `json:"content,omitempty"`
	Scope   string   `json:"scope"`
	Paths   []string `json:"paths,omitempty"`
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
	dir := filepath.Dir(trackPath)
	dirFiles := dirAudioPaths(dir)
	paths := []string{trackPath}
	if scope == "album" {
		paths = albumAudioPaths(env, trackPath, dirFiles)
	}
	album := ""
	if info, err := tags.ReadTags(trackPath); err == nil {
		album = strings.TrimSpace(info.Album)
	}
	// A mixed folder is not an album. Publish one folder cover only when the
	// album tag is set and every audio file in the directory shares it.
	// Dump directories such as "soundcloud" are never albums.
	forgetFolderArt(dir)
	folderWide := scope == "album" && album != "" && !isDumpFolder(dir) && coversPaths(paths, dirFiles)
	destArt := artPathTrack(env, trackPath)
	if folderWide {
		destArt = artPathFolder(env, trackPath)
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
	if !nonEmptyFile(content) {
		if err := os.Rename(tmp, content); err != nil {
			return InstallResult{}, err
		}
	} else {
		_ = os.Remove(tmp)
	}
	if err := artLinkFolderAlias(destArt, content); err != nil {
		return InstallResult{}, err
	}
	for _, p := range paths {
		trackArt := artPathTrack(env, p)
		if trackArt != destArt {
			_ = artLinkFolderAlias(trackArt, content)
		}
		rememberTrackArt(env, p, trackArt)
	}
	if folderWide {
		markArtDirty(env, trackPath, "album")
	} else {
		for _, p := range paths {
			markArtDirty(env, p, "track")
		}
	}
	return InstallResult{
		Art:     destArt,
		Track:   artPathTrack(env, trackPath),
		Folder:  artPathFolder(env, trackPath),
		Content: content,
		Scope:   scope,
		Paths:   paths,
	}, nil
}

func dirAudioPaths(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if playback.IsSupportedPath(p) {
			out = append(out, p)
		}
	}
	return out
}

// albumAudioPaths is the audio files in dirFiles that share trackPath's album tag.
// An empty album tag matches nothing else: a year folder of mixes is not an album.
func albumAudioPaths(env Env, trackPath string, dirFiles []string) []string {
	info, err := tags.ReadTags(trackPath)
	album := ""
	if err == nil {
		album = info.Album
	}
	known := albumsInDir(env, filepath.Dir(trackPath))
	return pathsSharingAlbum(trackPath, album, dirFiles, func(p string) string {
		if p == trackPath {
			return album
		}
		if v, ok := known[p]; ok {
			return v
		}
		inf, err := tags.ReadTags(p)
		if err != nil {
			return ""
		}
		return inf.Album
	})
}

func albumsInDir(env Env, dir string) map[string]string {
	out := map[string]string{}
	db, err := EnsureDB(env)
	if err != nil || db == nil {
		return out
	}
	rows, err := db.Query(`SELECT path, album FROM tracks WHERE parent_dir=?`, dir)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var path, album string
		if err := rows.Scan(&path, &album); err != nil {
			continue
		}
		out[path] = album
	}
	return out
}

func pathsSharingAlbum(trackPath, album string, dirFiles []string, albumOf func(string) string) []string {
	album = strings.TrimSpace(album)
	if album == "" || albumOf == nil {
		return []string{trackPath}
	}
	out := make([]string, 0)
	for _, p := range dirFiles {
		if strings.EqualFold(strings.TrimSpace(albumOf(p)), album) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{trackPath}
	}
	return out
}

func coversPaths(paths, files []string) bool {
	if len(files) == 0 || len(paths) != len(files) {
		return false
	}
	have := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		have[p] = struct{}{}
	}
	for _, f := range files {
		if _, ok := have[f]; !ok {
			return false
		}
	}
	return true
}

func rememberTrackArt(env Env, path, art string) {
	db, err := EnsureDB(env)
	if err != nil || db == nil || path == "" || art == "" {
		return
	}
	_, _ = db.Exec(`UPDATE tracks SET art=? WHERE path=?`, art, path)
}

func ApplyImageURL(env Env, trackPath, imageURL, scope string) (InstallResult, error) {
	tmp := filepath.Join(env.ArtDir, fmt.Sprintf(".fetch.%d.jpg", time.Now().UnixNano()))
	if err := downloadArtURL(imageURL, tmp); err != nil {
		return InstallResult{}, err
	}
	res, err := InstallImage(env, trackPath, tmp, scope)
	os.Remove(tmp)
	return res, err
}

func downloadArtURL(imageURL, dest string) error {
	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "evoplayer/1.0 (local music player)")
	if strings.Contains(imageURL, "discogs.com") {
		req.Header.Set("Referer", "https://www.discogs.com/")
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("art fetch %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := validateImageBytes(body); err != nil {
		return err
	}
	return os.WriteFile(dest, body, 0o644)
}

func validateImageBytes(body []byte) error {
	if len(body) == 0 {
		return fmt.Errorf("empty art response")
	}
	_, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("not an image: %w", err)
	}
	return nil
}

func ClearArt(env Env, trackPath string) error {
	trackArt := artPathTrack(env, trackPath)
	folder := artPathFolder(env, trackPath)
	if trackArt != folder && nonEmptyFile(trackArt) {
		_ = os.Remove(trackArt)
	} else if nonEmptyFile(folder) {
		_ = os.Remove(folder)
	}
	markArtDirty(env, trackPath, "track")
	return nil
}

func normalizeJPG(src, dest string) error {
	if err := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", src,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", artCacheMax, artCacheMax),
		"-q:v", "2", dest).Run(); err == nil {
		if st, err := os.Stat(dest); err == nil && st.Size() > 0 {
			return nil
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := validateImageBytes(data); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

func Maintain(env Env) error {
	snap, err := artDirtySnapshot(env)
	if err != nil {
		return err
	}
	for dir := range snap.Dirs {
		_ = embedFolder(env, dir)
	}
	for track := range snap.Tracks {
		art := artCacheFind(env, track)
		if art == "" {
			continue
		}
		_ = embedAudio(track, art)
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
	return WriteFileAtomic(dirtyPath(env), raw, 0o644)
}

func embedFolder(env Env, dir string) error {
	forgetFolderArt(dir)
	if !directoryIsAlbum(dir, func(p string) string {
		info, err := tags.ReadTags(p)
		if err != nil {
			return ""
		}
		return info.Album
	}) {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
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
		return nil
	}
	art := artPathFolder(env, sample)
	if !nonEmptyFile(art) {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if playback.IsSupportedPath(p) {
			_ = embedAudio(p, art)
		}
	}
	return nil
}

func embedAudio(audioPath, artPath string) error {
	st, err := os.Stat(audioPath)
	if err != nil {
		return err
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(audioPath), "."))
	f, err := os.CreateTemp(filepath.Dir(audioPath), ".arttmp-*."+ext)
	if err != nil {
		return err
	}
	tmp := f.Name()
	f.Close()
	switch ext {
	case "mp3":
		err = exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", audioPath, "-i", artPath,
			"-map", "0:a", "-map", "1:v", "-c:a", "copy", "-c:v", "mjpeg",
			"-id3v2_version", "3", tmp).Run()
	default:
		err = exec.Command("ffmpeg", "-y", "-loglevel", "error", "-i", audioPath, "-i", artPath,
			"-map", "0", "-map", "1", "-c", "copy", "-disposition:v:0", "attached_pic", tmp).Run()
	}
	if err == nil {
		err = os.Chmod(tmp, st.Mode().Perm())
	}
	if err == nil {
		err = os.Rename(tmp, audioPath)
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}
