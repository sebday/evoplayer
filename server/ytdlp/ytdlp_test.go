package ytdlp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestBrowsersOrder(t *testing.T) {
	if got := Browsers("brave", "chromium"); !slices.Equal(got, []string{"brave", "chromium", ""}) {
		t.Fatalf("got %q", got)
	}
	if got := Browsers(); !slices.Equal(got, []string{"", "brave", "chromium"}) {
		t.Fatalf("got %q", got)
	}
}

func TestNetrcPrivate(t *testing.T) {
	args, cleanup, err := Netrc("soundcloud", "oauth", "secret")
	if err != nil {
		t.Fatal(err)
	}
	path := args[len(args)-1]
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", st.Mode().Perm())
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("netrc not removed: %v", err)
	}
}

func TestToMP3(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "in.wav")
	if err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "anullsrc=r=44100", "-t", "1", src).Run(); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out.mp3")
	var progressed bool
	if err := ToMP3(context.Background(), src, dest, func(float64) { progressed = true }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("temp file left behind")
	}
	if !progressed {
		t.Fatal("no progress reported")
	}
	bad := filepath.Join(dir, "bad.mp3")
	if err := ToMP3(context.Background(), filepath.Join(dir, "missing.wav"), bad, nil); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Fatal("failed conversion left dest")
	}
}
