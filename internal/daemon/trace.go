package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/internal/ipc"
	"github.com/sebday/evoplayer/internal/playback"
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
	hyp := ipcHypothesis(req.Method)
	if agentDebugEnabled() && hyp != "" {
		agentDebugLog(hyp, "daemon.go:traceIPCIn", "ipc req", map[string]any{
			"id": req.ID, "method": req.Method, "before": tracePlaybackSnap(before),
			"params": traceIPCParams(req.Method, req.Params),
		})
	}
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
	hyp := ipcHypothesis(req.Method)
	if agentDebugEnabled() && hyp != "" {
		agentDebugLog(hyp, "daemon.go:traceIPCOut", "ipc res", map[string]any{
			"id": req.ID, "method": req.Method, "ok": err == nil,
			"before": tracePlaybackSnap(before), "after": tracePlaybackSnap(after),
			"durMs": dur.Milliseconds(), "err": errString(err),
		})
	}
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

func ipcHypothesis(method string) string {
	switch {
	case method == "playback.toggle" || method == "playback.stop":
		return "H1"
	case strings.HasPrefix(method, "queue."):
		return "H2"
	case method == "viz.subscribe" || method == "viz.unsubscribe":
		return "H4"
	default:
		return ""
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func traceMPRIS(action string, before playback.Status) {
	if agentDebugEnabled() {
		agentDebugLog("H3", "daemon.go:traceMPRIS", action, map[string]any{
			"before": tracePlaybackSnap(before),
		})
	}
	if !ipcTraceEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "evoplayer: mpris %s %s\n", action, tracePlaybackSnap(before))
}
