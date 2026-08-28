package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func agentDebugLogPath() string {
	if v := os.Getenv("EVOPLAYER_AGENT_DEBUG_LOG"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/evoplayer-agent-debug.log"
	}
	return filepath.Join(home, "Projects", "evoplayer", ".cursor", "debug-e0555c.log")
}

var agentDebugMu sync.Mutex

func agentDebugEnabled() bool {
	v := os.Getenv("EVOPLAYER_AGENT_DEBUG")
	return v != "" && v != "0" && v != "false"
}

func agentDebugLog(hypothesisID, location, message string, data map[string]any) {
	if !agentDebugEnabled() {
		return
	}
	if data == nil {
		data = map[string]any{}
	}
	entry := map[string]any{
		"sessionId":    "e0555c",
		"runId":        "pre-fix",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	agentDebugMu.Lock()
	defer agentDebugMu.Unlock()
	f, err := os.OpenFile(agentDebugLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
}
