package youtube

import (
	"os"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/internal/paths"
)

func TestYtdlpInfoArtist(t *testing.T) {
	info := ytdlpInfo{Artist: "A", Uploader: "U", Channel: "C"}
	if got := info.artist(); got != "A" {
		t.Fatalf("artist = %q", got)
	}
	info.Artist = ""
	if got := info.artist(); got != "U" {
		t.Fatalf("uploader fallback = %q", got)
	}
}

func TestYtdlpBaseArgsCookies(t *testing.T) {
	got := strings.Join(ytdlpBaseArgs("brave"), " ")
	if !strings.Contains(got, "--cookies-from-browser brave") {
		t.Fatalf("args = %q", got)
	}
	if strings.Contains(strings.Join(ytdlpBaseArgs(""), " "), "--cookies-from-browser") {
		t.Fatal("empty browser should omit cookies")
	}
}

func TestLastNonEmptyLine(t *testing.T) {
	got := lastNonEmptyLine("a\nERROR: unexpected status code: 403\n\n")
	if got != "ERROR: unexpected status code: 403" {
		t.Fatalf("got %q", got)
	}
}

func TestCookieBrowserOrderPrefersWorking(t *testing.T) {
	got := cookieBrowserOrder("brave")
	if got[0] != "brave" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseYtDlpPercent(t *testing.T) {
	got, ok := parseYtDlpPercent("[download]  45.2% of 80.00MiB")
	if !ok || got != 45 {
		t.Fatalf("got %d %v", got, ok)
	}
}

func TestDefaultGenreEmptyWithoutConfig(t *testing.T) {
	if got := defaultGenre(""); got != "" {
		t.Fatalf("default genre = %q, want empty", got)
	}
}

func TestDownloadURLRequiresYtDlp(t *testing.T) {
	orig := lookYtDlp
	t.Cleanup(func() { lookYtDlp = orig })
	lookYtDlp = func() (string, error) { return "", os.ErrNotExist }
	_, err := DownloadURL(paths.Env{MusicRoot: t.TempDir()}, "https://www.youtube.com/watch?v=pjVX_-rdB10")
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required") {
		t.Fatalf("err = %v", err)
	}
}
