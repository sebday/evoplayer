package ipc

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
)

type clientConn struct {
	conn net.Conn
	mu   sync.Mutex
}

func (c *clientConn) writeJSON(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	c.mu.Lock()
	_, err = c.conn.Write(b)
	c.mu.Unlock()
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
	default:
		return false
	}
}
