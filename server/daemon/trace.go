package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/ipc"
	"github.com/sebday/evoplayer/server/playback"
)

func ipcTraceEnabled() bool {
	v := strings.TrimSpace(os.Getenv("EVOPLAYER_TRACE_IPC"))
	return v != "" && v != "0" && !strings.EqualFold(v, "false")
}

func ipcTraceVerbose() bool {
	v := strings.TrimSpace(os.Getenv("EVOPLAYER_TRACE_IPC"))
	return strings.EqualFold(v, "all") || strings.EqualFold(v, "verbose")
}

func ipcTraceInteresting(method string) bool {
	if ipcTraceVerbose() {
		return true
	}
	switch {
	case strings.HasPrefix(method, "playback."),
		strings.HasPrefix(method, "queue."),
		method == "viz.subscribe",
		method == "viz.unsubscribe",
		method == "subscribe":
		return true
	default:
		return false
	}
}

func tracePlaybackSnap(st playback.Status) string {
	path := st.Path
	if path != "" {
		path = filepath.Base(path)
	}
	return fmt.Sprintf("path=%q state=%s pos=%.1f dur=%.1f queue=%d/%d",
		path, st.State, st.Position, st.Duration, st.PlaylistPos+1, st.PlaylistCount)
}

func traceIPCParams(method string, params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}
	switch method {
	case "queue.replace", "queue.load", "queue.play_current":
		var p struct {
			Paths     []string `json:"paths"`
			StartPath string   `json:"start_path"`
		}
		if json.Unmarshal(params, &p) != nil {
			break
		}
		start := p.StartPath
		if start != "" {
			start = filepath.Base(start)
		}
		return fmt.Sprintf("paths=%d start=%q", len(p.Paths), start)
	case "queue.play_path":
		var p struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(params, &p) == nil && p.Path != "" {
			return fmt.Sprintf("path=%q", filepath.Base(p.Path))
		}
	case "queue.append":
		var p struct {
			Paths []string `json:"paths"`
		}
		if json.Unmarshal(params, &p) == nil {
			return fmt.Sprintf("paths=%d", len(p.Paths))
		}
	case "playback.seek":
		var p struct {
			Seconds float64 `json:"seconds"`
		}
		if json.Unmarshal(params, &p) == nil {
			return fmt.Sprintf("seconds=%.1f", p.Seconds)
		}
	case "playback.volume.set", "playback.volume.delta":
		return string(params)
	}
	if ipcTraceVerbose() {
		return string(params)
	}
	return fmt.Sprintf("params=%dB", len(params))
}

func traceIPCIn(req ipc.Request, before playback.Status) {
	if !ipcTraceEnabled() || !ipcTraceInteresting(req.Method) {
		return
	}
	extra := traceIPCParams(req.Method, req.Params)
	if extra != "" {
		fmt.Fprintf(os.Stderr, "evoplayer: ipc req id=%d method=%s %s %s\n",
			req.ID, req.Method, tracePlaybackSnap(before), extra)
		return
	}
	fmt.Fprintf(os.Stderr, "evoplayer: ipc req id=%d method=%s %s\n",
		req.ID, req.Method, tracePlaybackSnap(before))
}

func traceIPCOut(req ipc.Request, before, after playback.Status, err error, dur time.Duration) {
	if !ipcTraceEnabled() || !ipcTraceInteresting(req.Method) {
		return
	}
	ok := err == nil
	errMsg := ""
	if err != nil {
		errMsg = " err=" + err.Error()
	}
	changed := ""
	if before.State != after.State || before.Path != after.Path {
		changed = " CHANGED"
	}
	fmt.Fprintf(os.Stderr, "evoplayer: ipc res id=%d method=%s ok=%v dur=%s%s %s -> %s%s\n",
		req.ID, req.Method, ok, dur.Round(time.Millisecond), changed,
		tracePlaybackSnap(before), tracePlaybackSnap(after), errMsg)
}

func traceMPRIS(action string, before playback.Status) {
	if !ipcTraceEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "evoplayer: mpris %s %s\n", action, tracePlaybackSnap(before))
}
