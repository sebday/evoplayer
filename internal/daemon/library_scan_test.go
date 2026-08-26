package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/paths"
)

func testDaemonEnv(t *testing.T, musicRoot string) paths.Env {
	t.Helper()
	dir := t.TempDir()
	cache := filepath.Join(dir, "cache")
	state := filepath.Join(dir, "state")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	return paths.Env{
		MusicRoot:      musicRoot,
		StateDir:       state,
		CacheDir:       cache,
		LibraryDB:      filepath.Join(cache, "library.sqlite3"),
		LikesFile:      filepath.Join(cache, "likes.json"),
		TracksCacheDir: cache,
		WaveformDir:    filepath.Join(cache, "waveforms"),
		ArtDir:         filepath.Join(cache, "art"),
		SocketPath:     filepath.Join(dir, "evoplayer.sock"),
		DaemonLock:     filepath.Join(dir, "daemon.lock"),
		PlayerState:    filepath.Join(state, "player.json"),
		MusicConfig:    filepath.Join(state, "music.toml"),
	}
}

func newTestDaemon(t *testing.T, env paths.Env) *Daemon {
	t.Helper()
	d := New(env)
	t.Cleanup(func() {
		d.jobs.Cancel()
		d.warm.Close()
	})
	return d
}

func writeDummyTrack(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeWaveform(t *testing.T, env library.Env, track string) {
	t.Helper()
	if err := os.MkdirAll(env.WaveformDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(env.WaveformDir, library.CacheKey(env.MusicRoot, track)+".json")
	if err := os.WriteFile(wf, []byte(`{"peaks":[0]}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func waitJobIdle(t *testing.T, d *Daemon) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for d.jobRunning() {
		if time.Now().After(deadline) {
			t.Fatal("job still running")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStartBootstrapScanOnEmptyIndex(t *testing.T) {
	root := t.TempDir()
	writeDummyTrack(t, filepath.Join(root, "grime", "track.mp3"))
	d := newTestDaemon(t, testDaemonEnv(t, root))
	if !d.startBootstrapScan() {
		t.Fatal("expected bootstrap scan on empty index")
	}
	st := d.jobs.Status()
	if st == nil || st.Name != "scan" {
		t.Fatalf("job = %+v, want scan", st)
	}
	d.jobs.Cancel()
	waitJobIdle(t, d)
}

func TestStartBootstrapScanSkippedWhenIndexed(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "grime", "track.mp3")
	writeDummyTrack(t, track)
	env := testDaemonEnv(t, root)
	if _, err := library.CacheAll(library.EnvFrom(env), false); err != nil {
		t.Fatal(err)
	}
	d := newTestDaemon(t, env)
	if d.startBootstrapScan() {
		t.Fatal("populated index should not start bootstrap scan")
	}
	st := d.jobs.Status()
	if st != nil && st.Status == "running" {
		t.Fatalf("unexpected running job: %+v", st)
	}
}

func TestRunLibrarySyncWarmsOnlyMissingAndNew(t *testing.T) {
	root := t.TempDir()
	keep := filepath.Join(root, "grime", "keep.mp3")
	stale := filepath.Join(root, "grime", "stale.mp3")
	writeDummyTrack(t, keep)
	writeDummyTrack(t, stale)
	env := testDaemonEnv(t, root)
	lib := library.EnvFrom(env)
	if _, err := library.CacheAll(lib, false); err != nil {
		t.Fatal(err)
	}
	writeWaveform(t, lib, keep)

	added := filepath.Join(root, "grime", "added.mp3")
	writeDummyTrack(t, added)

	d := newTestDaemon(t, env)
	warmed, err := d.runLibrarySync(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{stale: true, added: true}
	if len(warmed) != 2 {
		t.Fatalf("warm paths = %v, want 2 (stale + added)", warmed)
	}
	for _, p := range warmed {
		if !want[p] {
			t.Fatalf("unexpected warm path %q in %v", p, warmed)
		}
	}
	indexed, err := library.ListTrackPaths(lib)
	if err != nil {
		t.Fatal(err)
	}
	if len(indexed) != 3 {
		t.Fatalf("indexed = %d, want 3 after incremental sync", len(indexed))
	}
}

func TestRunLibrarySyncMaintainsDirtyArt(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "grime", "track.mp3")
	writeDummyTrack(t, track)
	env := testDaemonEnv(t, root)
	lib := library.EnvFrom(env)
	if _, err := library.CacheAll(lib, false); err != nil {
		t.Fatal(err)
	}
	writeWaveform(t, lib, track)
	payload, err := json.Marshal(map[string]any{
		"dirs":   map[string]any{},
		"tracks": map[string]any{track: map[string]string{"at": "2026-01-01T00:00:00Z"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dirty := filepath.Join(env.CacheDir, "art-dirty.json")
	if err := os.WriteFile(dirty, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	d := newTestDaemon(t, env)
	if _, err := d.runLibrarySync(context.Background()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(dirty)
	if err != nil {
		t.Fatal(err)
	}
	var snap struct {
		Dirs   map[string]any `json:"dirs"`
		Tracks map[string]any `json:"tracks"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	if len(snap.Dirs) != 0 || len(snap.Tracks) != 0 {
		t.Fatalf("dirty after sync = %s", raw)
	}
}

func TestSyncLibraryOnceSkipsRunningJob(t *testing.T) {
	root := t.TempDir()
	writeDummyTrack(t, filepath.Join(root, "grime", "track.mp3"))
	env := testDaemonEnv(t, root)
	d := newTestDaemon(t, env)
	gate := make(chan struct{})
	if _, err := d.jobs.Start("import", func(context.Context) error {
		<-gate
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	d.syncLibraryOnce(context.Background())
	close(gate)
	waitJobIdle(t, d)
	need, err := library.NeedsScan(library.EnvFrom(env))
	if err != nil {
		t.Fatal(err)
	}
	if !need {
		t.Fatal("sync should have skipped while a job was running")
	}
}

func TestLoopLibrarySyncStopsOnCancel(t *testing.T) {
	d := newTestDaemon(t, testDaemonEnv(t, t.TempDir()))
	d.syncDelay = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		d.loopLibrarySync(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loopLibrarySync did not stop")
	}
}
