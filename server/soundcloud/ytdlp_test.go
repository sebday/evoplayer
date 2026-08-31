package soundcloud

import (
	"strings"
	"testing"
)

func TestYtDlpSoundcloudArgsMatchesScdlDefaults(t *testing.T) {
	args := ytDlpSoundcloudArgs("tok-123", "cid-456", "/tmp/out.%(ext)s", "https://soundcloud.com/a/b", true)
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--use-extractors soundcloud.*",
		"--extractor-args soundcloud:client_id=cid-456;formats=*_aac,*_mp3",
		"-f ba",
		"-u oauth",
		"-p tok-123",
		"https://soundcloud.com/a/b",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q in %q", want, joined)
		}
	}
}

func TestYtDlpSoundcloudArgsWithoutAuth(t *testing.T) {
	args := ytDlpSoundcloudArgs("", "", "/tmp/out.%(ext)s", "https://soundcloud.com/a/b", true)
	for _, arg := range args {
		if arg == "-u" || arg == "-p" {
			t.Fatalf("unexpected auth args: %v", args)
		}
	}
	if !strings.Contains(strings.Join(args, " "), "soundcloud:formats=*_aac,*_mp3") {
		t.Fatalf("expected default formats arg, got %v", args)
	}
}

func TestYtdlpProgressPhase(t *testing.T) {
	got := ytdlpProgressPhase("[download]  12.3% of ~   5.00MiB at  500.00KiB/s ETA 00:08")
	if got == "" || !strings.Contains(got, "12.3%") {
		t.Fatalf("got %q", got)
	}
}

func TestLastNonEmptyLine(t *testing.T) {
	got := lastNonEmptyLine("line1\n\nERROR: drm protected\n")
	if got != "ERROR: drm protected" {
		t.Fatalf("got %q", got)
	}
}
