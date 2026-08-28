package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if view.Soundcloud.OAuthSource != "" {
		t.Fatalf("toml json must not include oauth token source, got %q", view.Soundcloud.OAuthSource)
	}
}

func TestSetRejectsOAuthToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	if err := os.WriteFile(path, []byte("[soundcloud]\nuser = \"seb-day\"\noauth_token = \"stale\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Set(path, "soundcloud", "oauth_token", "nope")
	if err == nil {
		t.Fatal("expected error")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "oauth_token") {
		t.Fatalf("oauth_token still in toml: %s", raw)
	}
}

func TestPruneDerivedRemovesOAuthToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	if err := os.WriteFile(path, []byte("[soundcloud]\nuser = \"seb-day\"\noauth_token = \"stale\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := JSON(path, dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "oauth_token") {
		t.Fatalf("oauth_token still in toml: %s", raw)
	}
}
