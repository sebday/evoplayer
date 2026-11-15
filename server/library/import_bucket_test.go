package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLibraryBucketUsesExistingFolder(t *testing.T) {
	root := t.TempDir()
	genre := filepath.Join(root, "Dubstep")
	if err := os.MkdirAll(filepath.Join(genre, "Soundcloud"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := libraryBucket(root, "Dubstep", "soundcloud")
	want := filepath.Join(genre, "Soundcloud")
	if got != want {
		t.Fatalf("libraryBucket() = %q, want %q", got, want)
	}
	missing := libraryBucket(root, "Dubstep", "youtube")
	wantMissing := filepath.Join(genre, "Youtube")
	if missing != wantMissing {
		t.Fatalf("libraryBucket() = %q, want %q", missing, wantMissing)
	}
}
