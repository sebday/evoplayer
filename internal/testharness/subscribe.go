package testharness

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/sebday/evoplayer/internal/ipc"
)

// Subscriber listens for IPC broadcast events on the harness socket.
type Subscriber struct {
	events chan ipc.Event
	done   chan struct{}
	once   sync.Once
}

func (h *Harness) StartSubscriber() *Subscriber {
	h.t.Helper()
	conn, err := net.Dial("unix", h.Socket)
	if err != nil {
		h.t.Fatal(err)
	}
	sub := &Subscriber{
		events: make(chan ipc.Event, 32),
		done:   make(chan struct{}),
	}
	if err := writeIPC(conn, ipc.Request{Method: "subscribe"}); err != nil {
		h.t.Fatal(err)
	}
	go func() {
		defer conn.Close()
		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			var ev ipc.Event
			if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Event == "" {
				continue
			}
			select {
			case sub.events <- ev:
			case <-sub.done:
				return
			}
		}
	}()
	return sub
}

func (sub *Subscriber) Close() {
	sub.once.Do(func() { close(sub.done) })
}

func (sub *Subscriber) WaitState(timeout time.Duration, trigger func()) ipc.Event {
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-sub.events:
			if ev.Event == "state" {
				return ev
			}
		case <-deadline:
			if trigger != nil {
				trigger()
				deadline = time.After(timeout)
				trigger = nil
				continue
			}
			return ipc.Event{}
		}
	}
}

func (sub *Subscriber) CountEvents(event string, duration time.Duration) int {
	deadline := time.After(duration)
	n := 0
	for {
		select {
		case ev := <-sub.events:
			if ev.Event == event {
				n++
			}
		case <-deadline:
			return n
		}
	}
}

func writeIPC(conn net.Conn, req ipc.Request) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = conn.Write(b)
	return err
}
