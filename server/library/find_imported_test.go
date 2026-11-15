package library

import (
	"path/filepath"
	"testing"
)

func TestFindImported(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	base := ImportBasename("Alias", "Heat")
	if base == "" {
		t.Fatal("empty basename")
	}
	filed := filepath.Join("/music", "Dubstep", base)
	_, err = db.Exec(`INSERT INTO tracks (path, genre, parent_dir, title, artist) VALUES (?, 'Dubstep', 'Dubstep', 'Heat', 'Alias')`, filed)
	if err != nil {
		t.Fatal(err)
	}
	got, err := FindImported(db, "Alias", "Heat")
	if err != nil || got != filed {
		t.Fatalf("FindImported() = %q, %v", got, err)
	}

	renamed := "/music/Dubstep/renamed.mp3"
	_, err = db.Exec(`INSERT INTO tracks (path, genre, parent_dir, title, artist) VALUES (?, 'Dubstep', 'Dubstep', 'Gladiator', 'Noisia')`, renamed)
	if err != nil {
		t.Fatal(err)
	}
	got, err = FindImported(db, "Noisia", "Gladiator")
	if err != nil || got != renamed {
		t.Fatalf("renamed FindImported() = %q, %v", got, err)
	}

	got, err = FindImported(db, "Missing", "Track")
	if err != nil || got != "" {
		t.Fatalf("missing FindImported() = %q, %v", got, err)
	}
}
