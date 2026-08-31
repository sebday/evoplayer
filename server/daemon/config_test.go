package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/ipc"
)

func TestConfigSetPathsRoot(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	d := newTestDaemon(t, testDaemonEnv(t, root))

	data, err := json.Marshal(map[string]string{
		"section": "paths",
		"key":     "root",
		"value":   other,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := d.handle(ipc.Request{Method: "config.set", Params: data})
	if err != nil {
		t.Fatal(err)
	}
	view, ok := out.(config.JSONView)
	if !ok {
		t.Fatalf("config.set returned %T", out)
	}
	if view.Paths.Root != other {
		t.Fatalf("view root = %q, want %q", view.Paths.Root, other)
	}
	if d.Env.MusicRoot != other {
		t.Fatalf("daemon root = %q, want %q", d.Env.MusicRoot, other)
	}
	got, err := config.Get(d.Env.MusicConfig, "paths", "root", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != other {
		t.Fatalf("music.toml root = %q, want %q", got, other)
	}

	bad, err := json.Marshal(map[string]string{
		"section": "paths",
		"key":     "root",
		"value":   filepath.Join(root, "missing"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.handle(ipc.Request{Method: "config.set", Params: bad}); err == nil {
		t.Fatal("expected validation error for missing directory")
	}
	if d.Env.MusicRoot != other {
		t.Fatalf("daemon root should stay %q after failed set, got %q", other, d.Env.MusicRoot)
	}
}

func TestConfigSetPathsRootRejectsFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := newTestDaemon(t, testDaemonEnv(t, root))
	data, err := json.Marshal(map[string]string{
		"section": "paths",
		"key":     "root",
		"value":   file,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.handle(ipc.Request{Method: "config.set", Params: data}); err == nil {
		t.Fatal("expected validation error for file path")
	}
}
