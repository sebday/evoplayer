package syncarchive

import (
	"os"
	"strings"
	"testing"
)

func TestLoadLegacyAndPrefixed(t *testing.T) {
	dir := t.TempDir()
	path := Path(dir)
	if err := os.WriteFile(path, []byte("12345\nsc:99\nyt:abc123\n67890\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !a.HasSC(12345) || !a.HasSC(99) || !a.HasSC(67890) {
		t.Fatal("expected soundcloud ids")
	}
	if !a.HasYT("abc123") {
		t.Fatal("expected youtube id")
	}
}

func TestAddPersists(t *testing.T) {
	dir := t.TempDir()
	path := Path(dir)
	a, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AddSC(42); err != nil {
		t.Fatal(err)
	}
	if err := a.AddYT("vid1"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.HasSC(42) || !reloaded.HasYT("vid1") {
		t.Fatal("archive file should persist ids")
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	if !strings.Contains(text, "sc:42\n") || !strings.Contains(text, "yt:vid1\n") {
		t.Fatalf("file = %q", text)
	}
}
