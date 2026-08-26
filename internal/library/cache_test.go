package library_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/tags"
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

func TestImportUpsertsWithoutWiping(t *testing.T) {
	root := t.TempDir()
	cache := t.TempDir()
	existing := filepath.Join(root, "house", "keep.mp3")
	writeTinyMP3(t, existing, map[string]string{"title": "Keep", "genre": "House"})
	env := library.Env{
		MusicRoot:      root,
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
	}
	if _, err := library.CacheAll(env, false); err != nil {
		t.Fatal(err)
	}
	incoming := filepath.Join(root, ".incoming")
	src := filepath.Join(incoming, "new.mp3")
	writeTinyMP3(t, src, map[string]string{"title": "New", "artist": "N", "genre": "misc"})
	if err := library.RunImport(env); err != nil {
		t.Fatal(err)
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
	if n != 2 {
		t.Fatalf("tracks = %d, want 2 (import must not wipe)", n)
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

func TestNeedsBootstrapWhenIndexEmpty(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "grime", "track.mp3")
	writeTinyMP3(t, track, map[string]string{"title": "T", "artist": "A", "genre": "Grime"})
	env := testCacheEnv(t, root)
	boot, err := library.NeedsBootstrap(env)
	if err != nil {
		t.Fatal(err)
	}
	if !boot {
		t.Fatal("uncached library should need bootstrap")
	}
	need, err := library.NeedsScan(env)
	if err != nil {
		t.Fatal(err)
	}
	if !need {
		t.Fatal("uncached library should need scan")
	}
	if _, err := library.CacheAll(env, false); err != nil {
		t.Fatal(err)
	}
	boot, err = library.NeedsBootstrap(env)
	if err != nil {
		t.Fatal(err)
	}
	if boot {
		t.Fatal("indexed library should not need bootstrap")
	}
	need, err = library.NeedsScan(env)
	if err != nil {
		t.Fatal(err)
	}
	if need {
		t.Fatal("fresh index should not need scan without file changes")
	}
	missing, err := library.MissingWaveforms(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 1 || missing[0] != track {
		t.Fatalf("missing waveforms = %v, want [%s]", missing, track)
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
		t.Fatal("index without waveform should not need scan")
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
		t.Fatal("fresh index with waveform should not need scan")
	}
	missing, err := library.MissingWaveforms(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing waveforms = %v, want empty", missing)
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

func TestCacheFolderLabelRootFile(t *testing.T) {
	env := library.Env{MusicRoot: "/music"}
	if got := library.CacheFolderLabel(env, "/music/track.mp3"); got != "" {
		t.Fatalf("root file folder = %q, want empty", got)
	}
	if got := library.CacheFolderLabel(env, "/music/a/b/c.mp3"); got != "a/b" {
		t.Fatalf("nested folder = %q, want a/b", got)
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
	var empty int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tracks WHERE path = '' OR path IS NULL`).Scan(&empty); err != nil {
		t.Fatal(err)
	}
	if empty != 0 {
		t.Fatalf("empty-path tracks = %d, want 0", empty)
	}
}
