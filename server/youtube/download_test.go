package youtube

import (
	"context"
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

func TestVideoID(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=m7EwpN1jWTg": "m7EwpN1jWTg",
		"https://youtu.be/m7EwpN1jWTg":                "m7EwpN1jWTg",
		"https://example.com/x":                       "",
	}
	for url, want := range cases {
		if got := VideoID(url); got != want {
			t.Fatalf("VideoID(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestDownloadURLRequiresYtDlp(t *testing.T) {
	orig := lookYtDlp
	t.Cleanup(func() { lookYtDlp = orig })
	lookYtDlp = func() (string, error) { return "", os.ErrNotExist }
	_, err := DownloadURLCtx(context.Background(), paths.Env{MusicRoot: t.TempDir()}, "https://www.youtube.com/watch?v=pjVX_-rdB10", nil)
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required") {
		t.Fatalf("err = %v", err)
	}
}
