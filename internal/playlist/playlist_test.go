package playlist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/library"
)

func TestValidateUserName(t *testing.T) {
	root := t.TempDir()
	env := Env{
		MusicRoot:   root,
		PlaylistDir: filepath.Join(root, "playlists"),
	}
	if err := validateUserName(env, ""); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := validateUserName(env, "all"); err == nil {
		t.Fatal("expected error for reserved name")
	}
	if err := validateUserName(env, "mixes"); err == nil {
		t.Fatal("expected error for reserved mixes name")
	}
	if err := validateUserName(env, "My Mix"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
}

func TestWriteAndReadM3U(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.m3u")
	trackA := filepath.Join(dir, "a.flac")
	trackB := filepath.Join(dir, "b.mp3")
	for _, p := range []string{trackA, trackB} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths := []string{trackA, trackB}
	if err := writeM3U(path, paths); err != nil {
		t.Fatal(err)
	}
	got, err := readM3UPaths(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != paths[0] {
		t.Fatalf("unexpected paths: %#v", got)
	}
}

func TestToggleStar(t *testing.T) {
	stars, starred := toggleStar([]string{"a"}, "b")
	if !starred || len(stars) != 2 {
		t.Fatalf("unexpected toggle: %#v %v", stars, starred)
	}
	stars, starred = toggleStar(stars, "b")
	if starred || len(stars) != 1 {
		t.Fatalf("unexpected untoggle: %#v %v", stars, starred)
	}
}

func TestCreateUserPlaylist(t *testing.T) {
	root := t.TempDir()
	env := Env{
		PlaylistDir: filepath.Join(root, "playlists"),
		MusicRoot:   filepath.Join(root, "music"),
	}
	if err := os.MkdirAll(env.MusicRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateUser(env, "road trip"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(env.PlaylistDir, "road trip.m3u")); err != nil {
		t.Fatal(err)
	}
}

func TestListIndexIncludesMixes(t *testing.T) {
	root := t.TempDir()
	cache := t.TempDir()
	music := filepath.Join(root, "music")
	mix := filepath.Join(music, "dnb", "mixes", "2026", "set.mp3")
	short := filepath.Join(music, "dnb", "soundcloud", "tune.mp3")
	for _, p := range []string{mix, short} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dbPath := filepath.Join(cache, "library.sqlite3")
	db, err := library.OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked)
VALUES(?,?,?,?,?,?,?,?,?,?,?,1)`,
		mix, "dnb", filepath.Dir(mix), "Set", "DJ", "", "2026", "", float64(library.MixMinDurationSec), "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked)
VALUES(?,?,?,?,?,?,?,?,?,?,?,1)`,
		short, "dnb", filepath.Dir(short), "Tune", "A", "", "2026", "", 180.0, "", ""); err != nil {
		t.Fatal(err)
	}

	libEnv := library.Env{
		MusicRoot: music,
		LibraryDB: dbPath,
		LikesFile: filepath.Join(cache, "likes.json"),
	}
	env := Env{
		Env:         libEnv,
		MusicRoot:   music,
		PlaylistDir: filepath.Join(cache, "playlists"),
	}
	items, err := ListIndex(env)
	if err != nil {
		t.Fatal(err)
	}
	var mixes *IndexItem
	for i := range items {
		if items[i].Name == "mixes" {
			mixes = &items[i]
			break
		}
	}
	if mixes == nil {
		t.Fatalf("mixes playlist missing: %#v", items)
	}
	if mixes.Count != 1 || mixes.Kind != "system" {
		t.Fatalf("mixes = %#v, want count=1 kind=system", *mixes)
	}
	page, err := TracksPageFor(env, "mixes", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Path != mix {
		t.Fatalf("tracks = %#v", page)
	}
}
