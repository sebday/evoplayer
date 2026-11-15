package soundcloud

import (
	"testing"

	"github.com/sebday/evoplayer/server/library"
)

func TestLibraryHasBasenameSkipsExistingImportName(t *testing.T) {
	base := library.ImportBasename("Drum&BassArena", "Koherent - Night Cycles")
	known := map[string]bool{base: true}
	if !libraryHasBasename(known, "Drum&BassArena", "Koherent - Night Cycles") {
		t.Fatal("expected the library filename to count as a duplicate")
	}
	if libraryHasBasename(known, "Other", "Night Cycles") {
		t.Fatal("a different artist should not match")
	}
}
