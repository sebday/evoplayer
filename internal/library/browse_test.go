package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowseEntryTrackPathJSON(t *testing.T) {
	raw, err := json.Marshal(BrowseEntry{
		Type: "track",
		Track: Track{
			Path:  "/music/album/track.mp3",
			Title: "Track",
			Art:   "/music/.cache/art/album.jpg",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out["path"] != "/music/album/track.mp3" {
		t.Fatalf("track path missing from browse json: %#v", out)
	}
	if out["art"] != "/music/.cache/art/album.jpg" {
		t.Fatalf("track art missing from browse json: %#v", out)
	}
}

func TestBrowseEntryDirPathJSON(t *testing.T) {
	raw, err := json.Marshal(BrowseEntry{
		Type:  "dir",
		Name:  "drum&bass",
		Count: 12,
		Track: Track{Path: "drum&bass"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out["path"] != "drum&bass" {
		t.Fatalf("dir path missing from browse json: %#v", out)
	}
}

func TestPropagateFolderArt(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(album, "a.mp3")
	b := filepath.Join(album, "b.mp3")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	art := filepath.Join(artDir, "grime_2008.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	tracks := []Track{
		{Path: a, Type: "track"},
		{Path: b, Type: "track"},
	}
	propagateFolderArt(env, tracks)
	for i, want := range []string{art, art} {
		if tracks[i].Art != want {
			t.Fatalf("track %d art = %q, want %q", i, tracks[i].Art, want)
		}
		if tracks[i].Thumb != "" {
			t.Fatalf("track %d thumb = %q, want empty (no sync thumb generation)", i, tracks[i].Thumb)
		}
	}
}

func TestPropagateFolderArtFromCacheSkipsThumb(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	cache := filepath.Join(dir, "cache")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	track := filepath.Join(album, "a.mp3")
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir, CacheDir: cache}
	art := filepath.Join(artDir, artFolderKey(music, track)+".jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	tracks := []Track{{Path: track, Type: "track"}}
	propagateFolderArt(env, tracks)
	if tracks[0].Art != art {
		t.Fatalf("art = %q, want %q", tracks[0].Art, art)
	}
	if tracks[0].Thumb != "" {
		t.Fatalf("thumb = %q, want empty (no sync thumb generation)", tracks[0].Thumb)
	}
	if _, err := os.Stat(filepath.Join(cache, "thumbs")); !os.IsNotExist(err) {
		t.Fatal("propagateFolderArt should not create thumb cache dir")
	}
}

func TestPropagateFolderArtByDirectoryMultiAlbum(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	albumA := filepath.Join(music, "grime", "2008")
	albumB := filepath.Join(music, "grime", "2009")
	for _, album := range []string{albumA, albumB} {
		if err := os.MkdirAll(album, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	a := filepath.Join(albumA, "a.mp3")
	b := filepath.Join(albumB, "b.mp3")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artA := filepath.Join(artDir, "grime_2008.jpg")
	if err := os.WriteFile(artA, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	tracks := []Track{
		{Path: a, Type: "track", Art: artA},
		{Path: b, Type: "track"},
	}
	propagateFolderArtByDirectory(env, tracks)
	if tracks[0].Art != artA {
		t.Fatalf("album A art = %q, want %q", tracks[0].Art, artA)
	}
	if tracks[1].Art != "" {
		t.Fatalf("album B art = %q, want empty (must not inherit album A art)", tracks[1].Art)
	}
}

func TestBrowseQueuePathsOnly(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	albumA := filepath.Join(music, "grime", "2008")
	albumB := filepath.Join(music, "grime", "2009")
	for _, album := range []string{albumA, albumB} {
		if err := os.MkdirAll(album, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{
		filepath.Join(albumA, "a.mp3"),
		filepath.Join(albumB, "b.mp3"),
	} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{MusicRoot: music}
	out, err := Browse(env, BrowseOptions{Rel: "grime", Queue: true, QueuePathsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(out.Paths))
	}
	if len(out.Tracks) != 0 {
		t.Fatalf("expected no tracks payload, got %d", len(out.Tracks))
	}
}

func TestCollectQueueTracksLeafFolder(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.mp3", "b.mp3"} {
		if err := os.WriteFile(filepath.Join(album, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{MusicRoot: music}
	tracks, err := collectQueueTracks(env, "grime/2008", album)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(tracks))
	}
}

func TestCollectQueueTracksNestedFolder(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	genre := filepath.Join(music, "grime")
	album := filepath.Join(genre, "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(album, "track.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music}
	tracks, err := collectQueueTracks(env, "grime", genre)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 nested track, got %d", len(tracks))
	}
}

func TestBrowseQueueMode(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.mp3", "b.mp3"} {
		if err := os.WriteFile(filepath.Join(album, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{MusicRoot: music}
	out, err := Browse(env, BrowseOptions{Rel: "grime/2008", Queue: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Tracks) != 2 {
		t.Fatalf("expected 2 queue tracks, got %d", len(out.Tracks))
	}
}

func TestCollectQueueTracksMixedDirectAndNested(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	root := filepath.Join(music, "dubstep", "vinyl", "afbar")
	album := filepath.Join(root, "album")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "loose.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(album, "nested.mp3"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music}
	tracks, err := collectQueueTracksFS(env, "dubstep/vinyl/afbar", root)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 tracks from mixed folder, got %d", len(tracks))
	}
}

func TestCollectQueueTracksDBNested(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	cache := filepath.Join(dir, "cache")
	dbPath := filepath.Join(cache, "library.sqlite3")
	loose := filepath.Join(music, "dubstep", "vinyl", "afbar", "loose.mp3")
	nested := filepath.Join(music, "dubstep", "vinyl", "afbar", "album", "nested.mp3")
	for _, p := range []string{loose, nested} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, row := range []struct {
		path, parent string
	}{
		{loose, filepath.Dir(loose)},
		{nested, filepath.Dir(nested)},
	} {
		if _, err := db.Exec(`
INSERT INTO tracks(path,genre,parent_dir,title,artist,album,year,label,duration,art,waveform,liked)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			row.path, "dubstep", row.parent, filepath.Base(row.path), "", "", "", "", 0, "", "", 0,
		); err != nil {
			t.Fatal(err)
		}
	}
	env := Env{MusicRoot: music, LibraryDB: dbPath, CacheDir: cache}
	tracks, err := collectQueueTracksDB(env, db, filepath.Join(music, "dubstep", "vinyl", "afbar"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 db tracks, got %d", len(tracks))
	}
	out, err := Browse(env, BrowseOptions{Rel: "dubstep/vinyl/afbar", Queue: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Tracks) != 2 {
		t.Fatalf("expected 2 browse queue tracks, got %d", len(out.Tracks))
	}
}
