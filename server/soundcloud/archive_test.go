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
