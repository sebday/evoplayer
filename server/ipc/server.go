package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/sebday/evoplayer/server/perf"
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
	path         string
	handler      Handler
	ln           net.Listener
	wg           sync.WaitGroup
	mu           sync.Mutex
	clients      []*clientConn
	OnDisconnect func()

	coalesceMu      sync.Mutex
	coalescePending map[string]Event
	coalesceTimer   *time.Timer
}

func NewServer(path string, handler Handler) *Server {
	return &Server{path: path, handler: handler}
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
			return err
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleConn(c)
		}(conn)
	}
}

func (s *Server) Close() error {
	if s.ln == nil {
		return nil
	}
	err := s.ln.Close()
	s.wg.Wait()
	s.mu.Lock()
	for _, c := range s.clients {
		_ = c.conn.Close()
	}
	s.clients = nil
	s.mu.Unlock()
	_ = os.Remove(s.path)
	return err
}

// onClientDisconnect releases topic subscriptions for a dropped client connection.
func (s *Server) onClientDisconnect() {
	if s.OnDisconnect != nil {
		s.OnDisconnect()
	}
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
	n := len(s.clients)
	s.mu.Unlock()
	return n > 0
}

func (s *Server) broadcastImmediate(ev Event) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	b = append(b, '\n')
	s.mu.Lock()
	clients := append([]*clientConn(nil), s.clients...)
	s.mu.Unlock()
	alive := make([]*clientConn, 0, len(clients))
	for _, c := range clients {
		if err := c.writeRaw(b); err != nil {
			_ = c.conn.Close()
			continue
		}
		alive = append(alive, c)
	}
	s.mu.Lock()
	s.clients = alive
	s.mu.Unlock()
}

func (c *clientConn) writeRaw(b []byte) error {
	c.mu.Lock()
	_, err := c.conn.Write(b)
	c.mu.Unlock()
	return err
}

func (s *Server) addClient(c *clientConn) {
	s.mu.Lock()
	s.clients = append(s.clients, c)
	s.mu.Unlock()
}

func (s *Server) removeClient(c *clientConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.clients[:0]
	for _, client := range s.clients {
		if client != c {
			out = append(out, client)
		}
	}
	s.clients = out
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

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	cc := &clientConn{conn: conn}
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	subscribed := false
	onDisconnect := func() {
		if subscribed {
			s.removeClient(cc)
			s.onClientDisconnect()
		}
	}
	defer onDisconnect()
	for sc.Scan() {
		line := sc.Bytes()
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = cc.writeJSON(responseFromError(0, ErrInvalidParams("invalid request")))
			continue
		}
		if req.Method == "subscribe" && !subscribed {
			subscribed = true
			s.addClient(cc)
		}
		if req.ID == 0 {
			go func(r Request) {
				_, _ = s.handler(r)
			}(req)
			continue
		}
		if isSlowMethod(req.Method) {
			go func(r Request) {
				perf.IncIPCQueue()
				defer perf.DecIPCQueue()
				start := time.Now()
				data, err := s.handler(r)
				perf.RecordRequest(time.Since(start))
				if err != nil {
					_ = cc.writeJSON(responseFromError(r.ID, err))
				} else {
					_ = cc.writeJSON(Response{ID: r.ID, OK: true, Data: data})
				}
			}(req)
			continue
		}
		start := time.Now()
		data, err := s.handler(req)
		perf.RecordRequest(time.Since(start))
		if err != nil {
			_ = cc.writeJSON(responseFromError(req.ID, err))
		} else {
			_ = cc.writeJSON(Response{ID: req.ID, OK: true, Data: data})
		}
	}
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
