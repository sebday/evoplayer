package youtube

import (
	"os"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
)

func TestParseYtDlpPercent(t *testing.T) {
	got, ok := parseYtDlpPercent("[download]  45.2% of 80.00MiB")
	if !ok || got != 45 {
		t.Fatalf("got %d %v", got, ok)
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
