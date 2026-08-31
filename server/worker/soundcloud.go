package worker

import (
	"context"
	"os"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/soundcloud"
)

// RunSoundCloudDownload runs SoundCloud likes sync in the current process.
// Progress is written as NDJSON to stdout. Returns an exit code for os.Exit.
func RunSoundCloudDownload(ctx context.Context, env paths.Env, importAfter bool) int {
	DeprioritizeProcess()
	rep := NewNDJSONReporter(os.Stdout)
	if err := soundcloud.DownloadEnvReportCtx(ctx, env, rep); err != nil {
		_ = rep.Error(err.Error())
		return 1
	}
	if importAfter {
		if err := library.RunImportCtx(ctx, library.EnvFrom(env), rep); err != nil {
			_ = rep.Error(err.Error())
			return 1
		}
	}
	return 0
}
