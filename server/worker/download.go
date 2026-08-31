package worker

import (
	"context"
	"os"
	"strings"

	"github.com/sebday/evoplayer/server/download"
	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/paths"
)

// RunDownloadURL downloads a YouTube or SoundCloud URL in the current process.
func RunDownloadURL(ctx context.Context, env paths.Env, rawURL string, importAfter bool) int {
	DeprioritizeProcess()
	rep := NewNDJSONReporter(os.Stdout)
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		_ = rep.Error("url required")
		return 1
	}
	_, err := download.DownloadFromURLCtx(ctx, env, rawURL, rep, func(phase string, pct int) {
		rep.Progress(jobs.Progress{Phase: phase, Done: pct, Total: 100})
	})
	if err != nil {
		_ = rep.Error(err.Error())
		return 1
	}
	if importAfter {
		if err := download.ImportLibraryIncoming(ctx, env, rep); err != nil {
			_ = rep.Error(err.Error())
			return 1
		}
	}
	return 0
}
