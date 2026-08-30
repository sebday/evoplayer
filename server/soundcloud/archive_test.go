package soundcloud

import (
	"os"
	"testing"
)

func TestArchiveLoadSkipsLegacyLines(t *testing.T) {
	dir := t.TempDir()
	path := ArchivePath(dir)
	if err := os.WriteFile(path, []byte("12345\nyoutube abc\n67890\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := LoadArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Has(12345) || !a.Has(67890) {
		t.Fatal("expected numeric ids in archive")
	}
}

func TestArchiveAddPersists(t *testing.T) {
	dir := t.TempDir()
	path := ArchivePath(dir)
	a, err := LoadArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	if a.Has(42) {
		t.Fatal("new archive should not contain id")
	}
	if err := a.Add(42); err != nil {
		t.Fatal(err)
	}
	if !a.Has(42) {
		t.Fatal("archive should remember added id")
	}
	reloaded, err := LoadArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Has(42) {
		t.Fatal("archive file should persist id")
	}
	if err := a.Add(42); err != nil {
		t.Fatal(err)
	}
}

func TestArchivePendingSkipsKnownIDs(t *testing.T) {
	a := &Archive{seen: map[string]bool{"1": true, "3": true}}
	tracks := []Track{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
	pending := 0
	for _, track := range tracks {
		if !a.Has(track.ID) {
			pending++
		}
	}
	if pending != 2 {
		t.Fatalf("pending = %d, want 2", pending)
	}
}
