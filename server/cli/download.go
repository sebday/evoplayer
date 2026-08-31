package cli

import (
	"fmt"
	"strings"

	"github.com/sebday/evoplayer/server/download"
	"github.com/sebday/evoplayer/server/paths"
)

func CmdDownload(env paths.Env, args []string) error {
	if len(args) > 0 && strings.EqualFold(args[0], "url") {
		return fmt.Errorf("usage: evoplayer download <youtube-or-soundcloud-url> [--no-import]")
	}
	if url, rest := popHTTPArg(args); url != "" {
		return runDownloadURL(env, append([]string{url}, rest...))
	}
	doImport := hasFlag(args, "--import")
	return runLibraryJob(env, "library.soundcloud.download", map[string]any{"import": doImport})
}

func runDownloadURL(env paths.Env, args []string) error {
	noImport := hasFlag(args, "--no-import")
	url := ""
	for _, arg := range args {
		if arg == "--no-import" || arg == "--import" {
			continue
		}
		if url == "" {
			url = strings.TrimSpace(arg)
		}
	}
	if url == "" {
		return fmt.Errorf("usage: evoplayer download <youtube-or-soundcloud-url> [--no-import]")
	}
	if download.ClassifyURL(url) == "" {
		return fmt.Errorf("unsupported download url (youtube or soundcloud only)")
	}
	params := map[string]any{"url": url}
	if noImport {
		params["import"] = false
	}
	return runLibraryJob(env, "library.download", params)
}

func popHTTPArg(args []string) (string, []string) {
	var url string
	var rest []string
	for _, arg := range args {
		if arg == "--no-import" || arg == "--import" {
			rest = append(rest, arg)
			continue
		}
		trimmed := strings.TrimSpace(arg)
		if url == "" && isHTTPURL(trimmed) {
			url = trimmed
			continue
		}
		rest = append(rest, arg)
	}
	return url, rest
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
