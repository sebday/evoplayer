package config

import (
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/viz"
)

func TestVizConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	cfg, err := VizConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	def := viz.DefaultConfig()
	if cfg.FrameRate != def.FrameRate || cfg.Sensitivity != def.Sensitivity {
		t.Fatalf("defaults mismatch: %+v vs %+v", cfg, def)
	}
}

func TestVizConfigFromToml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	if err := Set(path, "viz", "frame_rate", "30"); err != nil {
		t.Fatal(err)
	}
	cfg, err := VizConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FrameRate != 30 {
		t.Fatalf("frame_rate: got %d want 30", cfg.FrameRate)
	}
}
