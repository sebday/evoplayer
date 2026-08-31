package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsMixDurationOnly(t *testing.T) {
	short := filepath.Join("house", "soundcloud", "artist-track.mp3")
	if IsMix(short, float64(MixMinDurationSec-1)) {
		t.Fatal("24:59 track should not be a mix")
	}
	if !IsMix(short, float64(MixMinDurationSec)) {
		t.Fatal("25:00 track should be a mix")
	}
	if IsMix(filepath.Join("house", "dj_hype-fabriclive_mix.mp3"), 60) {
		t.Fatal("short track with -mix in name should not be a mix")
	}
	if IsMix(filepath.Join("house", "soundcloud", "short-tune.mp3"), 180) {
		t.Fatal("short track should not be a mix")
	}
}

func TestLikedMixPaths(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "library.sqlite3")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mix := filepath.Join(root, "dnb", "mixes", "2026", "long.mp3")
	named := filepath.Join(root, "dnb", "mixes", "2026", "short_mix.mp3")
	short := filepath.Join(root, "dnb", "soundcloud", "tune.mp3")
	for _, p := range []string{mix, named, short} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	insert := func(path string, dur float64) {
		t.Helper()
		if _, err := db.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked)
VALUES(?,?,?,?,?,?,?,?,?,?,?,1)`,
			path, "dnb", filepath.Dir(path), "T", "A", "", "2026", "", dur, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	insert(mix, float64(MixMinDurationSec))
	insert(named, 180)
	insert(short, 180)

	paths, err := LikedMixPaths(db, Env{MusicRoot: root, LibraryDB: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != mix {
		t.Fatalf("paths = %#v, want [%s]", paths, mix)
	}
}
