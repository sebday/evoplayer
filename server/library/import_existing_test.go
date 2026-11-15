package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveIncomingDuplicateKeepsLibraryFile(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "Drum & Bass", "Soundcloud", "artist-title.mp3")
	if err := os.MkdirAll(filepath.Dir(lib), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lib, []byte("library"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "incoming", "Artist - Title.mp3")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("incoming bigger copy"), 0o644); err != nil {
		t.Fatal(err)
	}
	known := map[string]string{"artist-title.mp3": lib}

	removed, err := removeIncomingDuplicate(src, known, "artist-title.mp3")
	if err != nil || !removed {
		t.Fatalf("removeIncomingDuplicate() = %v, %v", removed, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("incoming copy still exists")
	}
	body, err := os.ReadFile(lib)
	if err != nil || string(body) != "library" {
		t.Fatalf("library file = %q, %v", body, err)
	}

	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, err = removeIncomingDuplicate(src, known, "other-track.mp3")
	if err != nil || removed {
		t.Fatalf("unknown name removed = %v, %v", removed, err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatal(err)
	}
}
