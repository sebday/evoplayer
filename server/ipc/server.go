package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
)

type Request struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	ID    int         `json:"id"`
	OK    bool        `json:"ok"`
	Code  string      `json:"code,omitempty"`
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

type Event struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data,omitempty"`
}

type Handler func(req Request) (interface{}, error)

type Server struct {
	path    string
	handler Handler
	ln      net.Listener
	wg      sync.WaitGroup
	mu      sync.Mutex
	conns   map[*clientConn]struct{}
	closed  bool

	closeOnce sync.Once
	closeErr  error

	// OnVizChange reports whether any connection holds a viz subscription.
	OnVizChange func(active bool)
	vizMu       sync.Mutex
	vizSubs     atomic.Int32

	coalesceMu      sync.Mutex
	coalescePending map[string]Event
	coalesceArmed   bool
}

func NewServer(path string, handler Handler) *Server {
	return &Server{path: path, handler: handler, conns: map[*clientConn]struct{}{}}
}

func (s *Server) Listen() error {
	_ = os.Remove(s.path)
	ln, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	s.ln = ln
	return nil
}

func (s *Server) Serve() error {
	if s.ln == nil {
		return errors.New("ipc server not listening")
	}
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		cc := &clientConn{conn: conn}
		if !s.track(cc) {
			_ = conn.Close()
			continue
		}
		go func() {
			defer s.wg.Done()
			s.handleConn(cc)
		}()
	}
}

func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		if s.ln == nil {
			return
		}
		s.closeErr = s.ln.Close()
		s.mu.Lock()
		s.closed = true
		for c := range s.conns {
			_ = c.conn.Close()
		}
		s.mu.Unlock()
		s.wg.Wait()
		_ = os.Remove(s.path)
	})
	return s.closeErr
}

func (s *Server) track(c *clientConn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.wg.Add(1)
	s.conns[c] = struct{}{}
	return true
}

func (s *Server) untrack(c *clientConn) {
	s.mu.Lock()
	delete(s.conns, c)
	s.mu.Unlock()
	s.setViz(c, false)
	_ = c.conn.Close()
}

func (s *Server) subscribe(c *clientConn) {
	s.mu.Lock()
	c.subscribed = true
	s.mu.Unlock()
}

func (s *Server) setViz(c *clientConn, on bool) {
	s.vizMu.Lock()
	defer s.vizMu.Unlock()
	if c.viz == on {
		return
	}
	c.viz = on
	delta := int32(-1)
	if on {
		delta = 1
	}
	active := s.vizSubs.Add(delta) > 0
	if s.OnVizChange != nil {
		s.OnVizChange(active)
	}
}

func (s *Server) HasVizClients() bool {
	return s.vizSubs.Load() > 0
}

func (s *Server) Broadcast(ev Event) {
	if ev.Event == "viz" || ev.Event == "state" {
		s.coalesceBroadcast(ev)
		return
	}
	s.broadcastImmediate(ev)
}

func (s *Server) HasEventClients() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.conns {
		if c.subscribed {
			return true
		}
	}
	return false
}

func (s *Server) broadcastImmediate(ev Event) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	b = append(b, '\n')
	s.mu.Lock()
	clients := make([]*clientConn, 0, len(s.conns))
	for c := range s.conns {
		if c.subscribed {
			clients = append(clients, c)
		}
	}
	s.mu.Unlock()
	for _, c := range clients {
		_ = c.writeRaw(b)
	}
}

func responseFromError(id int, err error) Response {
	resp := Response{ID: id, OK: false, Error: err.Error()}
	if ae, ok := AsError(err); ok {
		resp.Code = ae.Code
		if ae.Data != nil {
			resp.Data = ae.Data
		}
	}
	return resp
}

func (s *Server) handleConn(cc *clientConn) {
	defer s.untrack(cc)
	sc := bufio.NewScanner(cc.conn)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = cc.writeJSON(responseFromError(0, ErrInvalidParams("invalid request")))
			continue
		}
		switch req.Method {
		case "subscribe":
			s.subscribe(cc)
		case "viz.subscribe":
			s.setViz(cc, true)
		case "viz.unsubscribe":
			s.setViz(cc, false)
		}
		if req.ID == 0 {
			go func(r Request) {
				_, _ = s.handler(r)
			}(req)
			continue
		}
		if isSlowMethod(req.Method) {
			go s.respond(cc, req)
			continue
		}
		s.respond(cc, req)
	}
}

func (s *Server) respond(cc *clientConn, req Request) {
	data, err := s.handler(req)
	if err != nil {
		_ = cc.writeJSON(responseFromError(req.ID, err))
		return
	}
	_ = cc.writeJSON(Response{ID: req.ID, OK: true, Data: data})
}

func Call(path string, req Request) (Response, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()
	if err := writeRequest(conn, req); err != nil {
		return Response{}, err
	}
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	if !sc.Scan() {
		return Response{}, fmt.Errorf("no response")
	}
	var resp Response
	if err := json.Unmarshal(sc.Bytes(), &resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

func writeRequest(conn net.Conn, req Request) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = conn.Write(b)
	return err
}

func DecodeParams[T any](raw json.RawMessage, out *T) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
