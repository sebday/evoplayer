package library_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/tags"
)

func writeTinyMP3(t *testing.T, path string, extraTags map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono", "-t", "0.05",
		"-codec:a", "libmp3lame", "-q:a", "9", path).Run(); err != nil {
		t.Skip("ffmpeg required")
	}
	if extraTags != nil {
		if err := tags.EmbedMP3(path, extraTags, nil, ""); err != nil {
			t.Fatal(err)
		}
	}
}

func testCacheEnv(t *testing.T, root string) library.Env {
	t.Helper()
	cache := t.TempDir()
	return library.Env{
		MusicRoot:      root,
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
		WaveformDir:    filepath.Join(cache, "waveforms"),
		ArtDir:         filepath.Join(cache, "art"),
	}
}

func TestCacheIncrementalSkipsUnchanged(t *testing.T) {
	root := t.TempDir()
	cache := t.TempDir()
	track := filepath.Join(root, "grime", "track.mp3")
	writeTinyMP3(t, track, map[string]string{"title": "T", "artist": "A", "genre": "Grime"})
	env := library.Env{
		MusicRoot:      root,
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	first, err := library.CacheAll(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.Built != 1 {
		t.Fatalf("first built = %d, want 1", first.Built)
	}
	if len(first.Paths) != 1 || first.Paths[0] != track {
		t.Fatalf("first paths = %v, want [%s]", first.Paths, track)
	}
	second, err := library.CacheAll(env, false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Skipped != 1 || second.Built != 0 || len(second.Paths) != 0 {
		t.Fatalf("second = %+v, want skipped=1 built=0 paths empty", second)
	}
}

func TestNeedsScanEmptyLibrary(t *testing.T) {
	env := testCacheEnv(t, t.TempDir())
	need, err := library.NeedsScan(env)
	if err != nil {
		t.Fatal(err)
	}
	if need {
		t.Fatal("empty library should not need scan")
	}
	boot, err := library.NeedsBootstrap(env)
	if err != nil {
		t.Fatal(err)
	}
	if boot {
		t.Fatal("empty library should not need bootstrap")
	}
}

func TestNeedsScanIgnoresMissingWaveform(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "grime", "track.mp3")
	writeTinyMP3(t, track, map[string]string{"title": "T", "artist": "A", "genre": "Grime"})
	env := testCacheEnv(t, root)
	if _, err := library.CacheAll(env, false); err != nil {
		t.Fatal(err)
	}
	need, err := library.NeedsScan(env)
	if err != nil {
		t.Fatal(err)
	}
	if need {
		t.Fatal("index should not need scan")
	}
	if err := os.MkdirAll(env.WaveformDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(env.WaveformDir, library.CacheKey(env.MusicRoot, track)+".json")
	if err := os.WriteFile(wf, []byte(`{"peaks":[0]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	need, err = library.NeedsScan(env)
	if err != nil {
		t.Fatal(err)
	}
	if need {
		t.Fatal("waveform cache files should not affect scan")
	}
}

func TestNeedsScanWhenFileAdded(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "grime", "track.mp3")
	writeTinyMP3(t, track, map[string]string{"title": "T", "artist": "A", "genre": "Grime"})
	env := testCacheEnv(t, root)
	if _, err := library.CacheAll(env, false); err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(root, "grime", "new.mp3")
	writeTinyMP3(t, added, map[string]string{"title": "N", "artist": "A", "genre": "Grime"})
	need, err := library.NeedsScan(env)
	if err != nil {
		t.Fatal(err)
	}
	if !need {
		t.Fatal("new file should need incremental scan")
	}
	boot, err := library.NeedsBootstrap(env)
	if err != nil {
		t.Fatal(err)
	}
	if boot {
		t.Fatal("populated index should not need bootstrap after a new file")
	}
}

func TestNeedsScanFollowsMusicRootSymlink(t *testing.T) {
	real := t.TempDir()
	track := filepath.Join(real, "grime", "track.mp3")
	if err := os.MkdirAll(filepath.Dir(track), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "Music")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	env := testCacheEnv(t, link)
	need, err := library.NeedsScan(env)
	if err != nil {
		t.Fatal(err)
	}
	if !need {
		t.Fatal("symlink music root with files should need scan")
	}
}

func TestCacheAllCtxCancelDoesNotCommit(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "grime", "track.mp3")
	writeTinyMP3(t, track, map[string]string{"title": "T", "artist": "A", "genre": "Grime"})
	env := testCacheEnv(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := library.CacheAllCtx(ctx, env, true, func(library.CacheProgress) {
		cancel()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	db, err := library.OpenDB(env.LibraryDB)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tracks`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("tracks = %d, want 0 after cancel", n)
	}
}
