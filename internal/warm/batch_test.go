package warm

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/paths"
)

func warmTestEnv(t *testing.T) (paths.Env, string) {
	t.Helper()
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	cache := filepath.Join(dir, "cache")
	artDir := filepath.Join(cache, "art")
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
	art := filepath.Join(artDir, "grime_2008.jpg")
	if err := os.WriteFile(art, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := paths.Env{
		MusicRoot: music,
		CacheDir:  cache,
		ArtDir:    artDir,
	}
	return env, track
}

func TestTrackAssetsSkipsWaveform(t *testing.T) {
	env, track := warmTestEnv(t)
	res, err := TrackAssets(env, track)
	if err != nil {
		t.Fatal(err)
	}
	if res.Waveform != "" {
		t.Fatalf("waveform = %q, want empty", res.Waveform)
	}
	if res.WaveformBuilt {
		t.Fatal("expected WaveformBuilt false")
	}
	if res.Art == "" {
		t.Fatal("expected cached art path")
	}
}

func TestBatchTracksReturnsCachedArt(t *testing.T) {
	env, track := warmTestEnv(t)
	results, err := BatchTracks(env, []string{track}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != track {
		t.Fatalf("path = %q, want %q", results[0].Path, track)
	}
	if results[0].Art == "" {
		t.Fatal("expected art path in batch result")
	}
	if results[0].Waveform != "" {
		t.Fatalf("waveform = %q, want empty", results[0].Waveform)
	}
}

func TestTrackAssetsUncachedSkipsWaveform(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	music := filepath.Join(dir, "music")
	cache := filepath.Join(dir, "cache")
	artDir := filepath.Join(cache, "art")
	album := filepath.Join(music, "grime", "2008")
	if err := os.MkdirAll(album, 0o755); err != nil {
		t.Fatal(err)
	}
	track := filepath.Join(album, "track.mp3")
	cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error",
		"-t", "0.2", "-f", "lavfi", "-i", "sine=frequency=440:duration=0.2",
		"-c:a", "libmp3lame", track)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v: %s", err, out)
	}
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	env := paths.Env{MusicRoot: music, CacheDir: cache, ArtDir: artDir}
	res, err := TrackAssets(env, track)
	if err != nil {
		t.Fatal(err)
	}
	if res.Waveform != "" || res.WaveformBuilt {
		t.Fatalf("expected no waveform work, got %#v", res)
	}
}
