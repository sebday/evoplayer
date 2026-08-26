//go:build integration

package integration_test

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sebday/evoplayer/internal/testharness"
)

type browseResult struct {
	Path        string           `json:"path"`
	Entries     []map[string]any `json:"entries"`
	TrackTotal  int              `json:"trackTotal"`
	TrackOffset int              `json:"trackOffset"`
}

func copyFixtureMusic(t *testing.T, dest string) {
	t.Helper()
	src := filepath.Join(testharness.RepoRoot(), "tests", "fixtures", "music")
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		out := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, in)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBrowseRootAndFolder(t *testing.T) {
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	h.StartDaemon()
	defer h.Stop()

	var root browseResult
	h.IPCData("library.browse", map[string]any{"path": ""}, &root)
	found := false
	for _, e := range root.Entries {
		if e["type"] == "dir" && e["name"] == "grime" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("grime genre missing: %#v", root.Entries)
	}

	var folder browseResult
	h.IPCData("library.browse", map[string]any{
		"path":   "grime/2008",
		"offset": 0,
		"limit":  50,
	}, &folder)
	tracks := 0
	for _, e := range folder.Entries {
		if e["type"] == "track" {
			tracks++
		}
	}
	if tracks != 2 {
		t.Fatalf("expected 2 tracks, got %d: %#v", tracks, folder.Entries)
	}
	if folder.TrackTotal < 2 {
		t.Fatalf("trackTotal=%d", folder.TrackTotal)
	}
}

func TestBrowseFolderArtPropagation(t *testing.T) {
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	album := filepath.Join(h.MusicRoot, "grime", "2008")
	artDir := h.Env().ArtDir
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	folderArt := filepath.Join(artDir, "grime_2008.jpg")
	if err := os.WriteFile(folderArt, []byte("fake-jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = album
	h.StartDaemon()
	defer h.Stop()

	var folder browseResult
	h.IPCData("library.browse", map[string]any{
		"path":   "grime/2008",
		"offset": 0,
		"limit":  50,
	}, &folder)
	withArt := 0
	for _, e := range folder.Entries {
		if e["type"] != "track" {
			continue
		}
		art, _ := e["art"].(string)
		if art != "" {
			withArt++
		}
	}
	if withArt < 2 {
		t.Fatalf("expected art on all tracks, got %d with art: %#v", withArt, folder.Entries)
	}
}

func TestBrowseQueueIPC(t *testing.T) {
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	h.StartDaemon()
	defer h.Stop()

	var result map[string]any
	h.IPCData("library.browse", map[string]any{
		"path":  "grime/2008",
		"queue": true,
	}, &result)
	raw, ok := result["tracks"].([]any)
	if !ok || len(raw) != 2 {
		t.Fatalf("expected 2 queue tracks, got %#v", result["tracks"])
	}
	first, _ := raw[0].(map[string]any)
	start, _ := first["path"].(string)
	if start == "" {
		t.Fatal("missing track path")
	}
	paths := make([]string, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]any)
		p, _ := m["path"].(string)
		if p != "" {
			paths = append(paths, p)
		}
	}
	resp := h.IPC("queue.replace", map[string]any{
		"paths":      paths,
		"start_path": start,
	})
	if !resp.OK {
		t.Fatalf("queue.replace: %s", resp.Error)
	}
}

func TestBrowseQueueNestedFolder(t *testing.T) {
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	root := filepath.Join(h.MusicRoot, "dubstep", "vinyl", "afbar")
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
	h.StartDaemon()
	defer h.Stop()

	var result map[string]any
	h.IPCData("library.browse", map[string]any{
		"path":  "dubstep/vinyl/afbar",
		"queue": true,
	}, &result)
	raw, ok := result["tracks"].([]any)
	if !ok || len(raw) != 2 {
		t.Fatalf("expected 2 nested queue tracks, got %#v", result["tracks"])
	}
}

func TestRapidBrowseSwitch(t *testing.T) {
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	h.StartDaemon()
	defer h.Stop()

	for i := 0; i < 5; i++ {
		var a browseResult
		h.IPCData("library.browse", map[string]any{"path": "grime/2008", "offset": 0, "limit": 50}, &a)
		var b browseResult
		h.IPCData("library.browse", map[string]any{"path": "", "offset": 0, "limit": 50}, &b)
	}
}

func TestJobStatusAndCancel(t *testing.T) {
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	h.StartDaemon()
	defer h.Stop()

	resp := h.IPC("library.import", nil)
	if !resp.OK {
		t.Fatalf("library.import: %s", resp.Error)
	}
	var status map[string]any
	h.IPCData("job.status", nil, &status)
	if status["status"] == "idle" {
		t.Fatal("expected running job")
	}
	var result map[string]any
	h.IPCData("job.cancel", nil, &result)
	if result["cancelled"] != true {
		t.Fatalf("expected cancelled true, got %#v", result)
	}
}

func TestWarmTrack(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	track := filepath.Join(h.MusicRoot, "grime", "2008", "track-a.mp3")
	h.StartDaemon()
	defer h.Stop()

	var warm map[string]any
	h.IPCData("library.warm", map[string]any{"path": track}, &warm)
	if warm["path"] != track {
		t.Fatalf("unexpected warm response: %#v", warm)
	}
}

func TestBrowseJSONRoundTrip(t *testing.T) {
	h := testharness.New(t)
	copyFixtureMusic(t, h.MusicRoot)
	h.StartDaemon()
	defer h.Stop()

	resp := h.IPC("library.browse", map[string]any{"path": "grime/2008"})
	if !resp.OK {
		t.Fatal(resp.Error)
	}
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatal(err)
	}
	var decoded browseResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
}
