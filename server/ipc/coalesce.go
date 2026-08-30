package ipc

import (
	"time"
)

// coalesceWindow batches high-frequency state/viz events (latest frame wins).
const coalesceWindow = 16 * time.Millisecond

func (s *Server) coalesceBroadcast(ev Event) {
	s.coalesceMu.Lock()
	if s.coalescePending == nil {
		s.coalescePending = make(map[string]Event, 2)
	}
	s.coalescePending[ev.Event] = ev
	if s.coalesceTimer == nil {
		s.coalesceTimer = time.AfterFunc(coalesceWindow, s.flushCoalesced)
	}
	s.coalesceMu.Unlock()
}

func (s *Server) flushCoalesced() {
	s.coalesceMu.Lock()
	pending := s.coalescePending
	s.coalescePending = nil
	s.coalesceTimer = nil
	s.coalesceMu.Unlock()
	for _, ev := range pending {
		s.broadcastImmediate(ev)
	}
}
