package cli

import (
	"fmt"
	"strings"

	"github.com/sebday/evoplayer/server/paths"
)

func CmdDownload(env paths.Env, args []string) error {
	if len(args) > 0 && strings.EqualFold(args[0], "url") {
		return cmdDownloadURL(env, args[1:])
	}
	doImport := hasFlag(args, "--import")
	return runLibraryJob(env, "library.soundcloud.download", map[string]any{"import": doImport})
}

func cmdDownloadURL(env paths.Env, args []string) error {
	noImport := hasFlag(args, "--no-import")
	url := ""
	for _, arg := range args {
		if arg == "--no-import" {
			continue
		}
		if url == "" {
			url = strings.TrimSpace(arg)
		}
	}
	if url == "" {
		return fmt.Errorf("usage: evoplayer download url <youtube-or-soundcloud-url> [--no-import]")
	}
	params := map[string]any{"url": url}
	if noImport {
		params["import"] = false
	}
	return runLibraryJob(env, "library.download", params)
}
