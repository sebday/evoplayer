package worker

import (
	"context"
	"os"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

// RunCache indexes the library (or one genre folder) in the current process.
func RunCache(ctx context.Context, env paths.Env, genre string, force bool) int {
	DeprioritizeProcess()
	rep := NewNDJSONReporter(os.Stdout)
	libEnv := library.EnvFrom(env)
	onProgress := func(p library.CacheProgress) {
		rep.Progress(jobs.Progress{
			Phase:  "index",
			Folder: p.Folder,
			Done:   p.Done,
			Total:  p.Total,
		})
	}
	var err error
	if genre != "" {
		_, err = library.CacheGenreCtx(ctx, libEnv, genre, force, onProgress)
	} else {
		_, err = library.CacheAllCtx(ctx, libEnv, force, onProgress)
	}
	if err != nil {
		_ = rep.Error(err.Error())
		return 1
	}
	return 0
}
