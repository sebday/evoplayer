package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyStateCopiesMissingFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("HOME", filepath.Join(dir, "home"))

	legacy := legacyStateDir()
	if err := os.MkdirAll(filepath.Join(legacy, "playlists"), 0o755); err != nil {
		t.Fatal(err)
	}
	likes := []byte(`{"/music/a.mp3":{"title":"A","artist":"B","liked_at":"t"}}`)
	if err := os.WriteFile(filepath.Join(legacy, "likes.json"), likes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "playlists", "night.m3u"), []byte("/music/a.mp3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "daemon.lock"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "music", ".evoplayer")
	if err := os.MkdirAll(filepath.Join(dest, "playlists"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyState(dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "likes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(likes) {
		t.Fatalf("likes = %s", got)
	}
	if _, err := os.ReadFile(filepath.Join(dest, "playlists", "night.m3u")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "daemon.lock")); !os.IsNotExist(err) {
		t.Fatal("daemon lock should stay in the old state dir")
	}

	kept := []byte("keep")
	if err := os.WriteFile(filepath.Join(dest, "likes.json"), kept, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyState(dest); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(filepath.Join(dest, "likes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(kept) {
		t.Fatalf("existing likes overwritten: %s", got)
	}
}
