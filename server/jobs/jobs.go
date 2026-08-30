package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const progressNotifyInterval = 250 * time.Millisecond
const maxJobLogLines = 24

type State struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	StartedAt string    `json:"started_at,omitempty"`
	EndedAt   string    `json:"ended_at,omitempty"`
	Progress  *Progress `json:"progress,omitempty"`
	Log       string    `json:"log,omitempty"`
	Result    any       `json:"result,omitempty"`
}

type Manager struct {
	mu                 sync.Mutex
	current            *State
	cancel             func()
	seq                int
	onChange           func()
	lastProgressNotify time.Time
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) SetOnChange(fn func()) {
	m.mu.Lock()
	m.onChange = fn
	m.mu.Unlock()
}

func (m *Manager) Start(name string, fn func(ctx context.Context) error) (State, error) {
	m.mu.Lock()
	if m.current != nil && m.current.Status == "running" {
		cur := *m.current
		m.mu.Unlock()
		return cur, fmt.Errorf("job already running: %s", cur.Name)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.seq++
	id := fmt.Sprintf("job-%d", m.seq)
	st := State{
		ID:        id,
		Name:      name,
		Status:    "running",
		StartedAt: time.Now().Format(time.RFC3339),
		Log:       "",
	}
	m.current = &st
	m.cancel = cancel
	m.lastProgressNotify = time.Time{}
	startCb := m.onChange
	m.mu.Unlock()
	if startCb != nil {
		startCb()
	}

	go func() {
		err := fn(ctx)
		cancel()
		m.mu.Lock()
		if m.current == nil || m.current.ID != id {
			m.mu.Unlock()
			return
		}
		m.current.EndedAt = time.Now().Format(time.RFC3339)
		if err != nil {
			m.current.Status = "error"
			m.current.Error = err.Error()
		} else {
			m.current.Status = "done"
		}
		m.cancel = nil
		cb := m.onChange
		m.mu.Unlock()
		if cb != nil {
			cb()
		}
	}()
	return st, nil
}

func (m *Manager) Status() *State {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return nil
	}
	copy := *m.current
	return &copy
}

func (m *Manager) Cancel() bool {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (m *Manager) AppendLog(line string) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "evoplayer:")
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	m.mu.Lock()
	if m.current == nil {
		m.mu.Unlock()
		return
	}
	var lines []string
	if m.current.Log != "" {
		lines = strings.Split(strings.TrimRight(m.current.Log, "\n"), "\n")
	}
	lines = append(lines, line)
	if len(lines) > maxJobLogLines {
		lines = lines[len(lines)-maxJobLogLines:]
	}
	next := strings.Join(lines, "\n") + "\n"
	if next == m.current.Log {
		m.mu.Unlock()
		return
	}
	m.current.Log = next
	cb := m.onChange
	m.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func (m *Manager) SetProgress(p Progress) {
	m.mu.Lock()
	if m.current == nil {
		m.mu.Unlock()
		return
	}
	copy := p
	phaseChanged := m.current.Progress == nil || m.current.Progress.Phase != p.Phase
	m.current.Progress = &copy
	shouldNotify := phaseChanged || time.Since(m.lastProgressNotify) >= progressNotifyInterval
	var cb func()
	if shouldNotify {
		m.lastProgressNotify = time.Now()
		cb = m.onChange
	}
	m.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func (m *Manager) ClearProgress() {
	m.mu.Lock()
	if m.current != nil {
		m.current.Progress = nil
	}
	m.mu.Unlock()
}

func (m *Manager) BroadcastEvent() map[string]any {
	st := m.Status()
	if st == nil {
		return nil
	}
	ev := map[string]any{
		"id":     st.ID,
		"name":   st.Name,
		"status": st.Status,
		"error":  st.Error,
		"log":    st.Log,
	}
	if st.Progress != nil {
		ev["progress"] = st.Progress
	}
	if st.Result != nil {
		ev["result"] = st.Result
	}
	return ev
}

func (m *Manager) SetResult(v any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return
	}
	m.current.Result = v
}
