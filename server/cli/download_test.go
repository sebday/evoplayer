package cli

import (
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/paths"
)

func TestPopHTTPArg(t *testing.T) {
	url, rest := popHTTPArg([]string{"--no-import", "https://soundcloud.com/a/b"})
	if url != "https://soundcloud.com/a/b" {
		t.Fatalf("url = %q", url)
	}
	if len(rest) != 1 || rest[0] != "--no-import" {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestPopHTTPArgNoURL(t *testing.T) {
	url, rest := popHTTPArg([]string{"--import"})
	if url != "" || len(rest) != 1 {
		t.Fatalf("url = %q rest = %#v", url, rest)
	}
}

func TestDownloadURLSubcommandRemoved(t *testing.T) {
	err := CmdDownload(paths.Env{}, []string{"url", "https://soundcloud.com/a/b"})
	if err == nil {
		t.Fatal("download url subcommand should be removed")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("err = %v", err)
	}
}
