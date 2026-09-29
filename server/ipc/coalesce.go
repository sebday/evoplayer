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
	if !s.coalesceArmed {
		s.coalesceArmed = true
		time.AfterFunc(coalesceWindow, s.flushCoalesced)
	}
	s.coalesceMu.Unlock()
}

func (s *Server) flushCoalesced() {
	s.coalesceMu.Lock()
	pending := s.coalescePending
	s.coalescePending = nil
	s.coalesceMu.Unlock()
	for _, ev := range pending {
		s.broadcastImmediate(ev)
	}
	s.coalesceMu.Lock()
	if len(s.coalescePending) > 0 {
		time.AfterFunc(coalesceWindow, s.flushCoalesced)
	} else {
		s.coalesceArmed = false
	}
	s.coalesceMu.Unlock()
}
