package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtLinkAliasKeepsSharedBytes(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "content.jpg")
	dest := filepath.Join(dir, "track.jpg")
	body := []byte("jpeg-bytes")
	if err := os.WriteFile(content, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(content, dest); err != nil {
		t.Fatal(err)
	}
	if err := artLinkFolderAlias(dest, content); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(content)
	if err != nil || string(got) != string(body) {
		t.Fatalf("content = %q, err %v", got, err)
	}
}

func TestArtLinkAliasReplacesEmptyDest(t *testing.T) {
	dir := t.TempDir()
	content := filepath.Join(dir, "content.jpg")
	dest := filepath.Join(dir, "track.jpg")
	body := []byte("jpeg-bytes")
	if err := os.WriteFile(content, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := artLinkFolderAlias(dest, content); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != string(body) {
		t.Fatalf("dest = %q, err %v", got, err)
	}
	contentGot, err := os.ReadFile(content)
	if err != nil || string(contentGot) != string(body) {
		t.Fatalf("content = %q, err %v", contentGot, err)
	}
}
