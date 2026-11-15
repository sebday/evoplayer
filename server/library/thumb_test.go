package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoverThumbPathSeparate(t *testing.T) {
	dir := t.TempDir()
	art := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{CacheDir: dir}
	row := thumbPathForArt(env, art)
	cover := coverThumbPath(env, art)
	if row == "" || cover == "" || row == cover {
		t.Fatalf("row %q cover %q", row, cover)
	}
	if !strings.HasSuffix(row, ".jpg") || strings.Contains(filepath.Base(row), "@") {
		t.Fatalf("row thumb %q", row)
	}
	if !strings.HasSuffix(cover, "@256.jpg") {
		t.Fatalf("cover thumb %q", cover)
	}
}
