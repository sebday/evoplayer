package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sebday/evoplayer/server/cli"
	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/playlist"

	tea "github.com/charmbracelet/bubbletea"
)

func decodeData[T any](resp ipc.Response, out *T) error {
	if !resp.OK {
		if resp.Error != "" {
			return fmt.Errorf("%s", resp.Error)
		}
		return fmt.Errorf("ipc failed")
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func fetchNav(env paths.Env) ([]navItem, error) {
	resp, err := cli.IPC(env, "library.playlist.list", nil)
	if err != nil {
		return nil, err
	}
	var items []playlist.IndexItem
	if err := decodeData(resp, &items); err != nil {
		return nil, err
	}
	return navFromIndex(items), nil
}

func fetchTracks(env paths.Env, name string) ([]library.Track, error) {
	resp, err := cli.IPC(env, "library.playlist.tracks", map[string]any{
		"name":   name,
		"offset": 0,
		"limit":  400,
	})
	if err != nil {
		return nil, err
	}
	var page playlist.TracksPage
	if err := decodeData(resp, &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}

func fetchBrowse(env paths.Env, rel string) (library.BrowseResult, error) {
	var out library.BrowseResult
	resp, err := cli.IPC(env, "library.browse", map[string]any{
		"path":   rel,
		"offset": 0,
		"limit":  400,
	})
	if err != nil {
		return out, err
	}
	err = decodeData(resp, &out)
	return out, err
}

func fetchBrowseQueue(env paths.Env, rel string) ([]string, error) {
	resp, err := cli.IPC(env, "library.browse", map[string]any{
		"path":             rel,
		"queue":            true,
		"queue_paths_only": true,
	})
	if err != nil {
		return nil, err
	}
	var out library.BrowseResult
	if err := decodeData(resp, &out); err != nil {
		return nil, err
	}
	return out.Paths, nil
}

func fetchStatus(env paths.Env) (playback.Status, error) {
	return cli.PlaybackStatus(env)
}

func playTracks(env paths.Env, tracks []library.Track, start string) error {
	pathsList := make([]string, 0, len(tracks))
	for _, t := range tracks {
		if t.Path != "" {
			pathsList = append(pathsList, t.Path)
		}
	}
	if len(pathsList) == 0 {
		return fmt.Errorf("no tracks")
	}
	if start == "" {
		start = pathsList[0]
	}
	return playPaths(env, pathsList, start)
}

func appendFolder(env paths.Env, rel string) error {
	resp, err := cli.IPC(env, "queue.append_folder", map[string]any{"path": rel})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func playPaths(env paths.Env, pathsList []string, start string) error {
	if len(pathsList) == 0 {
		return fmt.Errorf("no tracks")
	}
	if start == "" {
		start = pathsList[0]
	}
	resp, err := cli.IPC(env, "queue.replace", map[string]any{
		"paths":      pathsList,
		"start_path": start,
	})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func appendPaths(env paths.Env, pathsList []string) error {
	if len(pathsList) == 0 {
		return fmt.Errorf("no tracks")
	}
	resp, err := cli.IPC(env, "queue.append", map[string][]string{"paths": pathsList})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

func togglePlayback(env paths.Env) error {
	_, err := cli.IPC(env, "playback.toggle", nil)
	return err
}

func skipTrack(env paths.Env, method string) error {
	_, err := cli.IPC(env, method, nil)
	return err
}

func adjustVolume(env paths.Env, delta int) error {
	_, err := cli.IPC(env, "playback.volume.delta", map[string]int{"delta": delta})
	return err
}

func toggleLike(env paths.Env, path string) (playlist.FavoriteResult, error) {
	var out playlist.FavoriteResult
	resp, err := cli.IPC(env, "library.favorite.toggle", map[string]string{"path": path})
	if err != nil {
		return out, err
	}
	err = decodeData(resp, &out)
	return out, err
}

func fetchUpNext(env paths.Env, limit int) ([]library.Track, error) {
	if limit <= 0 {
		limit = 8
	}
	resp, err := cli.IPC(env, "queue.up_next", map[string]any{"limit": limit})
	if err != nil {
		return nil, err
	}
	var tracks []library.Track
	if err := decodeData(resp, &tracks); err != nil {
		return nil, err
	}
	return tracks, nil
}

func fetchJob(env paths.Env) (jobs.State, error) {
	var st jobs.State
	resp, err := cli.IPC(env, "job.status", nil)
	if err != nil {
		return st, err
	}
	if !resp.OK {
		if resp.Error != "" {
			return st, fmt.Errorf("%s", resp.Error)
		}
		return st, fmt.Errorf("ipc failed")
	}
	err = decodeData(resp, &st)
	return st, err
}

func runLibraryJob(env paths.Env, method string, params map[string]any) (jobs.State, error) {
	var st jobs.State
	resp, err := cli.IPC(env, method, params)
	if err != nil {
		return st, err
	}
	if !resp.OK {
		if resp.Error != "" {
			return st, fmt.Errorf("%s", resp.Error)
		}
		return st, fmt.Errorf("ipc failed")
	}
	err = decodeData(resp, &st)
	return st, err
}

func warmWaveform(env paths.Env, path string) (string, error) {
	resp, err := cli.IPC(env, "library.warm.waveform", map[string]string{"path": path})
	if err != nil {
		return "", err
	}
	if !resp.OK {
		if resp.Error != "" {
			return "", fmt.Errorf("%s", resp.Error)
		}
		return "", fmt.Errorf("ipc failed")
	}
	var out struct {
		Waveform string `json:"waveform"`
	}
	if err := decodeData(resp, &out); err != nil {
		return "", err
	}
	return out.Waveform, nil
}

func readWaveformPeaks(file string) ([]int, string) {
	if file == "" {
		return nil, ""
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, file
	}
	var payload struct {
		Channels int   `json:"channels"`
		Data     []int `json:"data"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return nil, file
	}
	data := payload.Data
	if payload.Channels >= 2 {
		out := make([]int, 0, (len(data)+1)/2)
		for i := 0; i < len(data); i += 2 {
			v := data[i]
			if i+1 < len(data) && data[i+1] > v {
				v = data[i+1]
			}
			out = append(out, v)
		}
		return out, file
	}
	return data, file
}

func statusTick() time.Duration {
	return 400 * time.Millisecond
}

func subscribeViz(env paths.Env) tea.Cmd {
	return func() tea.Msg {
		_, err := cli.IPC(env, "viz.subscribe", nil)
		return vizSubMsg{subscribed: err == nil, err: err}
	}
}

func unsubViz(env paths.Env) tea.Cmd {
	return func() tea.Msg {
		_, err := cli.IPC(env, "viz.unsubscribe", nil)
		return vizSubMsg{subscribed: false, err: err}
	}
}
