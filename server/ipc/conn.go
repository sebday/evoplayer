package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const writeTimeout = 2 * time.Second

type clientConn struct {
	conn       net.Conn
	mu         sync.Mutex
	subscribed bool
	viz        bool
}

func (c *clientConn) writeJSON(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.writeRaw(append(b, '\n'))
}

func (c *clientConn) writeRaw(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_, err := c.conn.Write(b)
	if err != nil {
		_ = c.conn.Close()
	}
	return err
}

func isSlowMethod(method string) bool {
	switch {
	case strings.HasPrefix(method, "library."):
		return method != "library.meta"
	case strings.HasPrefix(method, "job."):
		return true
	case strings.HasPrefix(method, "scrobble."):
		return true
	case strings.HasPrefix(method, "discover."):
		return true
	default:
		return false
	}
}

// Conn is a persistent client connection for sequential request/response calls.
type Conn struct {
	conn   net.Conn
	sc     *bufio.Scanner
	nextID int
}

func Dial(path string) (*Conn, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	return &Conn{conn: conn, sc: sc}, nil
}

func (c *Conn) Call(method string, params json.RawMessage) (Response, error) {
	c.nextID++
	id := c.nextID
	if err := writeRequest(c.conn, Request{ID: id, Method: method, Params: params}); err != nil {
		return Response{}, err
	}
	for c.sc.Scan() {
		var resp Response
		if err := json.Unmarshal(c.sc.Bytes(), &resp); err != nil {
			return Response{}, err
		}
		if resp.ID == id {
			return resp, nil
		}
	}
	if err := c.sc.Err(); err != nil {
		return Response{}, err
	}
	return Response{}, fmt.Errorf("no response")
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
