package library_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
)

type importTestReporter struct {
	lines []string
}

func (r *importTestReporter) Progress(jobs.Progress) {}
func (r *importTestReporter) Line(s string)            { r.lines = append(r.lines, s) }

func TestRunImportCtxLogsMoves(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "grime", "soundcloud"), 0o755); err != nil {
		t.Fatal(err)
	}
	track := filepath.Join(root, ".incoming", "Artist - Title.mp3")
	writeTinyMP3(t, track, map[string]string{
		"title":  "Title",
		"artist": "Artist",
		"genre":  "grime",
	})
	env := testCacheEnv(t, root)
	rep := &importTestReporter{}
	if err := library.RunImportCtx(context.Background(), env, rep); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(rep.lines, "\n")
	if !strings.Contains(joined, "· importing .incoming") {
		t.Fatalf("missing start log: %q", joined)
	}
	if !strings.Contains(joined, "✓ grime/soundcloud/") {
		t.Fatalf("missing ok log: %q", joined)
	}
	if !strings.Contains(joined, "· imported 1") {
		t.Fatalf("missing summary: %q", joined)
	}
}

func TestRunImportCtxLogsEmptyIncoming(t *testing.T) {
	root := t.TempDir()
	env := testCacheEnv(t, root)
	rep := &importTestReporter{}
	if err := library.RunImportCtx(context.Background(), env, rep); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(rep.lines, "\n")
	if !strings.Contains(joined, "· nothing to import") {
		t.Fatalf("missing empty log: %q", joined)
	}
}

func TestRunImportCtxSkipsUntagged(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, ".incoming", "Artist - Title.mp3")
	writeTinyMP3(t, track, map[string]string{
		"title":  "Title",
		"artist": "Artist",
	})
	env := testCacheEnv(t, root)
	rep := &importTestReporter{}
	if err := library.RunImportCtx(context.Background(), env, rep); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(rep.lines, "\n")
	if !strings.Contains(joined, "↷") || !strings.Contains(joined, "(no genre)") {
		t.Fatalf("missing skip log: %q", joined)
	}
	if !strings.Contains(joined, "· skipped 1") {
		t.Fatalf("missing skipped summary: %q", joined)
	}
	if _, err := os.Stat(track); err != nil {
		t.Fatal("untagged file should remain in .incoming")
	}
}
