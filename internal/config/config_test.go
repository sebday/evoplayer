package config

import (
	"path/filepath"
	"testing"
)

func TestJSONIncludesVizDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	view, err := JSON(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if view.Paths.Root == "" {
		t.Fatal("expected default music root")
	}
	if view.Viz.FrameRate != 45 {
		t.Fatalf("expected default viz frame_rate 45, got %d", view.Viz.FrameRate)
	}
}
