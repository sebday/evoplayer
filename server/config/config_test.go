package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestDefaultMusicRootPrefersLowercase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	music := filepath.Join(home, "music")
	upper := filepath.Join(home, "Music")
	if err := os.MkdirAll(music, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(upper, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := DefaultMusicRoot(); got != music {
		t.Fatalf("DefaultMusicRoot: got %q want %q", got, music)
	}
}

func TestDefaultMusicRootFallsBackToMusic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	upper := filepath.Join(home, "Music")
	if err := os.MkdirAll(upper, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := DefaultMusicRoot(); got != upper {
		t.Fatalf("DefaultMusicRoot: got %q want %q", got, upper)
	}
}

func TestDefaultMusicRootWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, "music")
	if got := DefaultMusicRoot(); got != want {
		t.Fatalf("DefaultMusicRoot: got %q want %q", got, want)
	}
}

func TestResolveRootUsesToml(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	root := filepath.Join(dir, "library")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Set(path, "paths", "root", root); err != nil {
		t.Fatal(err)
	}
	if got := ResolveRoot(path); got != root {
		t.Fatalf("ResolveRoot: got %q want %q", got, root)
	}
}

func TestResolveRootFallsBackWhenTomlMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	music := filepath.Join(home, "music")
	if err := os.MkdirAll(music, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "music.toml")
	if err := Set(path, "paths", "root", filepath.Join(home, "Music")); err != nil {
		t.Fatal(err)
	}
	if got := ResolveRoot(path); got != music {
		t.Fatalf("ResolveRoot: got %q want %q", got, music)
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
