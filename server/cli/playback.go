package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/playlist"
	"github.com/sebday/evoplayer/server/status"
	"github.com/sebday/evoplayer/server/warm"
)

func CmdStart(env paths.Env, exe string) error {
	return EnsureDaemon(env, exe)
}

func CmdRestart(env paths.Env, exe string) error {
	if DaemonUp(env) {
		_ = savePlayerState(env)
	}
	restartDaemon(env)
	return EnsureDaemon(env, exe)
}

func CmdStop(env paths.Env) error {
	if !DaemonUp(env) {
		return nil
	}
	resp, err := IPC(env, "playback.stop", nil)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	for i := 0; i < 50; i++ {
		st, err := PlaybackStatus(env)
		if err == nil && st.State == "stopped" {
			_ = saveStatus(env, st)
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("evoplayer: stop timed out")
}

func CmdClose(env paths.Env) error {
	if err := savePlayerState(env); err != nil {
		return err
	}
	return CmdStop(env)
}

func CmdToggle(env paths.Env, exe string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	if !DaemonUp(env) {
		return resumeSaved(env, exe, false)
	}
	st, err := PlaybackStatus(env)
	if err != nil {
		return err
	}
	if st.State == "stopped" {
		return resumeSaved(env, exe, false)
	}
	_, err = IPC(env, "playback.toggle", nil)
	return err
}

func CmdNext(env paths.Env, exe string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	_, err := IPC(env, "playback.next", nil)
	if err == nil {
		_ = savePlayerState(env)
	}
	return err
}

func CmdPrev(env paths.Env, exe string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	_, err := IPC(env, "playback.prev", nil)
	if err == nil {
		_ = savePlayerState(env)
	}
	return err
}

func CmdSeek(env paths.Env, exe, sec string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	v, err := strconv.ParseFloat(sec, 64)
	if err != nil {
		return fmt.Errorf("invalid seek seconds: %s", sec)
	}
	_, err = IPC(env, "playback.seek", map[string]float64{"seconds": v})
	return err
}

func CmdShuffle(env paths.Env, exe, mode string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	on := false
	switch mode {
	case "toggle":
		st, err := PlaybackStatus(env)
		if err != nil {
			return err
		}
		on = !st.Shuffle
	case "on":
		on = true
	case "off":
		on = false
	default:
		return fmt.Errorf("unknown shuffle mode: %s", mode)
	}
	_, err := IPC(env, "playback.shuffle", map[string]bool{"on": on})
	return err
}

func CmdVolume(env paths.Env, exe, arg, value string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	if arg == "set" {
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid volume: %s", value)
		}
		_, err = IPC(env, "playback.volume.set", map[string]int{"volume": v})
		return err
	}
	delta, err := strconv.Atoi(arg)
	if err != nil {
		return fmt.Errorf("invalid volume delta: %s", arg)
	}
	_, err = IPC(env, "playback.volume.delta", map[string]int{"delta": delta})
	return err
}

func CmdLoad(env paths.Env, exe string, args []string) error {
	path := ""
	folder := false
	jsonOut := false
	for _, a := range args {
		switch a {
		case "--folder":
			folder = true
		case "--json":
			jsonOut = true
		default:
			if !strings.HasPrefix(a, "-") {
				path = a
			}
		}
	}
	if path == "" {
		return fmt.Errorf("usage: evoplayer load <path> [--folder] [--json]")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return fmt.Errorf("evoplayer: not a file: %s", path)
	}
	pathsList := []string{path}
	if folder {
		dir := filepath.Dir(path)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		pathsList = nil
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if playback.IsSupportedPath(p) {
				pathsList = append(pathsList, p)
			}
		}
		found := false
		for _, p := range pathsList {
			if p == path {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("evoplayer: track not in folder listing: %s", path)
		}
	}
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	resp, err := IPC(env, "queue.replace", map[string]interface{}{
		"paths":      pathsList,
		"start_path": path,
	})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	if err := queueSave(env, pathsList); err != nil {
		return err
	}
	st, err := waitForTrack(env, path, 5*time.Second)
	if err != nil {
		return err
	}
	_ = saveStatus(env, st)
	go warmTrack(env, path)
	if jsonOut {
		return printJSON(st)
	}
	fmt.Printf("%s — %s (%s)\n", st.Artist, st.Title, st.State)
	return nil
}

func waitForTrack(env paths.Env, path string, timeout time.Duration) (playback.Status, error) {
	deadline := time.Now().Add(timeout)
	var last playback.Status
	var lastErr error
	for time.Now().Before(deadline) {
		st, err := PlaybackStatus(env)
		if err != nil {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
			continue
		}
		last = st
		if st.Path == path && st.State != "stopped" {
			return st, nil
		}
		if st.Path == path && st.State == "stopped" {
			return st, fmt.Errorf("evoplayer: failed to start playback for %s", path)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr != nil {
		return playback.Status{}, lastErr
	}
	if last.Path == path {
		return last, nil
	}
	return playback.Status{}, fmt.Errorf("evoplayer: timed out loading %s", path)
}

func saveStatus(env paths.Env, st playback.Status) error {
	return status.Write(env, st)
}

func warmTrack(env paths.Env, path string) {
	_, _ = warm.Track(env, path)
}

func CmdStatus(env paths.Env, exe string, jsonOut bool) error {
	if DaemonUp(env) {
		st, err := PlaybackStatus(env)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(st)
		}
		if st.State == "stopped" {
			fmt.Println("stopped")
			return nil
		}
		fmt.Printf("%s — %s (%s)\n", st.Artist, st.Title, st.State)
		return nil
	}
	st := savedStatusJSON(env)
	if jsonOut {
		return printJSON(st)
	}
	if st.Path == "" {
		fmt.Println("stopped")
		return nil
	}
	fmt.Printf("%s — %s (stopped)\n", st.Artist, st.Title)
	return nil
}

func CmdOpen(env paths.Env, exe string, jsonOut bool) error {
	if DaemonUp(env) {
		st, err := PlaybackStatus(env)
		if err != nil {
			return err
		}
		if st.Path != "" {
			_ = savePlayerState(env)
			if jsonOut {
				return printJSON(st)
			}
			fmt.Println(st.Path)
			return nil
		}
	}
	st := savedStatusJSON(env)
	if jsonOut {
		return printJSON(st)
	}
	fmt.Println(st.Path)
	return nil
}

func CmdQueueAppend(env paths.Env, exe string, pathsArg []string) error {
	if len(pathsArg) == 0 {
		return fmt.Errorf("usage: evoplayer queue append <path> [...]")
	}
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	filtered := make([]string, 0, len(pathsArg))
	for _, p := range pathsArg {
		if playback.IsSupportedPath(p) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	_, err := IPC(env, "queue.append", map[string][]string{"paths": filtered})
	if err != nil {
		return err
	}
	all, _ := readCurrentQueue(env)
	all = appendUnique(all, filtered)
	return queueSave(env, all)
}

func CmdQueuePlay(env paths.Env, exe string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: evoplayer queue play <start-path> <path> [...]")
	}
	start := args[0]
	pathsList := args[1:]
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	_, err := IPC(env, "queue.replace", map[string]interface{}{
		"paths":      pathsList,
		"start_path": start,
	})
	if err != nil {
		return err
	}
	return queueSave(env, pathsList)
}

func savePlayerState(env paths.Env) error {
	st, err := PlaybackStatus(env)
	if err != nil {
		return nil
	}
	return status.Write(env, st)
}

func resumeSaved(env paths.Env, exe string, playing bool) error {
	b, err := os.ReadFile(env.PlayerState)
	if err != nil {
		return nil
	}
	var saved struct {
		Path     string  `json:"path"`
		Position float64 `json:"position"`
	}
	if json.Unmarshal(b, &saved) != nil || saved.Path == "" {
		return nil
	}
	if _, err := os.Stat(saved.Path); err != nil {
		return nil
	}
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	_, err = IPC(env, "queue.replace", map[string]interface{}{
		"paths":      []string{saved.Path},
		"start_path": saved.Path,
	})
	if err != nil {
		return err
	}
	if saved.Position > 0 {
		_, _ = IPC(env, "playback.seek", map[string]float64{"seconds": saved.Position})
	}
	if playing {
		st, _ := PlaybackStatus(env)
		if st.State == "paused" {
			_, _ = IPC(env, "playback.toggle", nil)
		}
	} else {
		st, _ := PlaybackStatus(env)
		if st.State == "playing" {
			_, _ = IPC(env, "playback.toggle", nil)
		}
	}
	return nil
}

func queueSave(env paths.Env, pathsList []string) error {
	if err := env.EnsureDirs(); err != nil {
		return err
	}
	return playlist.SaveCurrent(playlist.EnvFrom(env), pathsList)
}

func readCurrentQueue(env paths.Env) ([]string, error) {
	return playlist.ReadCurrentPaths(playlist.EnvFrom(env))
}

func CmdQueueExtend(env paths.Env, exe string, jsonOut bool) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	st, err := PlaybackStatus(env)
	if err != nil {
		return err
	}
	if st.Shuffle {
		if jsonOut {
			return printJSON(playlist.ExtendResult{})
		}
		return fmt.Errorf("evoplayer: cannot extend queue while shuffle is on")
	}
	pEnv := playlist.EnvFrom(env)
	result, added, merged, atEnd, err := playlist.ExtendCurrent(pEnv, st.Path)
	if err != nil {
		return err
	}
	if _, err := IPC(env, "queue.append", map[string][]string{"paths": added}); err != nil {
		return err
	}
	if err := playlist.SaveCurrent(pEnv, merged); err != nil {
		return err
	}
	if atEnd {
		_, _ = IPC(env, "playback.next", nil)
	}
	if jsonOut {
		return printJSON(result)
	}
	fmt.Fprintf(os.Stderr, "evoplayer: extended queue with %d track(s) from %s\n", result.Added, filepath.Base(result.Folder))
	return nil
}

func CmdQueueUpNext(env paths.Env, exe string, args []string) error {
	jsonOut := false
	limit := 5
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--limit":
			continue
		default:
			if strings.HasPrefix(a, "-") {
				continue
			}
		}
	}
	for i, a := range args {
		if a == "--limit" && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				limit = n
			}
		}
	}
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	limitArg := limit
	resp, err := IPC(env, "queue.up_next", map[string]any{"limit": limitArg})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	var items []library.Track
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	if jsonOut {
		return printJSON(items)
	}
	for _, item := range items {
		line := item.Title
		if item.Artist != "" {
			line = item.Artist + " — " + line
		}
		fmt.Println(line)
	}
	return nil
}

func appendUnique(base, add []string) []string {
	seen := make(map[string]struct{}, len(base))
	for _, p := range base {
		seen[p] = struct{}{}
	}
	for _, p := range add {
		if _, ok := seen[p]; ok {
			continue
		}
		base = append(base, p)
		seen[p] = struct{}{}
	}
	return base
}
