package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtCacheFindRelativePath(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	track := filepath.Join(music, "dnb", "album", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	art := filepath.Join(artDir, "dnb_album.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	got := artCacheFind(env, "dnb/album/track.mp3")
	if got != art {
		t.Fatalf("artCacheFind relative = %q, want %q", got, art)
	}
}

func TestArtCacheFindUsesAlbumFolderKey(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	track := filepath.Join(music, "dnb", "album", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	art := filepath.Join(artDir, "dnb_album.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, ArtDir: artDir}
	got := artCacheFind(env, track)
	if got != art {
		t.Fatalf("artCacheFind = %q, want %q", got, art)
	}
}

func TestArtCacheFindMissingReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	artDir := filepath.Join(dir, "art")
	track := filepath.Join(music, "dnb", "album", "track.mp3")
	env := Env{MusicRoot: music, ArtDir: artDir}
	if artCacheFind(env, track) != "" {
		t.Fatal("expected empty art for missing cache")
	}
}

func TestWaveformCacheFindUsesCacheKey(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	wfDir := filepath.Join(dir, "waveforms")
	track := filepath.Join(music, "dnb", "album", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(wfDir, "dnb_album_track_mp3.json")
	if err := os.WriteFile(wf, []byte(`{"data":[1]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{MusicRoot: music, WaveformDir: wfDir}
	got := waveformCacheFind(env, track)
	if got != wf {
		t.Fatalf("waveformCacheFind = %q, want %q", got, wf)
	}
}

func TestTrackCacheSlug(t *testing.T) {
	if got := trackCacheSlug("drum&bass/album name/track.mp3"); got != "drum&bass_album_name_track_mp3" {
		t.Fatalf("unexpected slug: %q", got)
	}
}

func TestMaintainClearsDirty(t *testing.T) {
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	track := filepath.Join(music, "dnb", "album", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := Env{
		MusicRoot: music,
		CacheDir:  filepath.Join(dir, "cache"),
		ArtDir:    filepath.Join(dir, "art"),
	}
	markArtDirty(env, track, "track")
	snap, err := artDirtySnapshot(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Tracks) == 0 {
		t.Fatal("expected dirty track before maintain")
	}
	st1, err := os.Stat(track)
	if err != nil {
		t.Fatal(err)
	}
	if err := Maintain(env); err != nil {
		t.Fatal(err)
	}
	snap, err = artDirtySnapshot(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Dirs) != 0 || len(snap.Tracks) != 0 {
		t.Fatalf("dirty after maintain = %+v", snap)
	}
	if err := Maintain(env); err != nil {
		t.Fatal(err)
	}
	st2, err := os.Stat(track)
	if err != nil {
		t.Fatal(err)
	}
	if !st2.ModTime().Equal(st1.ModTime()) || st2.Size() != st1.Size() {
		t.Fatal("second maintain rewrote audio after dirty was cleared")
	}
}
