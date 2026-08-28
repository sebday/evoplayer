package ipc_test

import (
	"net"
	"testing"
	"time"

	"github.com/sebday/evoplayer/server/ipc"
)

func TestSlowMethodDoesNotBlockNextRequest(t *testing.T) {
	path := t.TempDir() + "/test.sock"
	block := make(chan struct{})
	srv := ipc.NewServer(path, func(req ipc.Request) (interface{}, error) {
		if req.Method == "library.browse" {
			<-block
			return map[string]string{"ok": "1"}, nil
		}
		return map[string]string{"fast": "1"}, nil
	})
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	go func() { _ = srv.Serve() }()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(`{"id":1,"method":"library.browse"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := conn.Write([]byte(`{"id":2,"method":"library.meta"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	close(block)
	time.Sleep(50 * time.Millisecond)
}
