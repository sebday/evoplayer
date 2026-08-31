package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/paths"
)

func absLibraryPath(env paths.Env, rel string) string {
	rel = strings.Trim(strings.TrimPrefix(rel, "/"), "/")
	if rel == "" {
		return filepath.Clean(env.MusicRoot)
	}
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(env.MusicRoot, filepath.FromSlash(rel))
}

func playlistFolderPath(env paths.Env, id string) string {
	switch id {
	case "all", "mixes", "current":
		return filepath.Clean(env.MusicRoot)
	default:
		return absLibraryPath(env, id)
	}
}

func (m model) folderOpenTarget() string {
	if m.browseFocused() {
		if item, ok := m.currentPlaylist(); ok {
			return playlistFolderPath(m.env, item.ID)
		}
		e, ok := m.currentBrowse()
		if !ok {
			return ""
		}
		switch e.Type {
		case "dir":
			return absLibraryPath(m.env, e.Path)
		case "track":
			if e.Path != "" {
				return filepath.Dir(e.Path)
			}
		}
		return ""
	}
	if t, ok := m.selectedTrack(); ok && t.Path != "" {
		return filepath.Dir(t.Path)
	}
	if path := strings.TrimSpace(m.status.Path); path != "" {
		return filepath.Dir(path)
	}
	return ""
}

func openInFileManager(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	if fm := strings.TrimSpace(os.Getenv("EVOPLAYER_FILE_MANAGER")); fm != "" {
		return exec.Command(fm, target).Start()
	}
	if _, err := exec.LookPath("thunar"); err == nil {
		cmd := exec.Command("thunar", target)
		cmd.Env = append(os.Environ(), "GDK_WAYLAND_APP_ID=floating-window")
		return cmd.Start()
	}
	if _, err := exec.LookPath("nautilus"); err == nil {
		return exec.Command("nautilus", target).Start()
	}
	if _, err := exec.LookPath("xdg-open"); err == nil {
		return exec.Command("xdg-open", target).Start()
	}
	return fmt.Errorf("no file manager found")
}
