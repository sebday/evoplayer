package download

import (
	"context"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/soundcloud"
)

// ImportLibraryIncoming normalizes audio in .incoming and moves files into the library.
func ImportLibraryIncoming(ctx context.Context, env paths.Env, rep jobs.Reporter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep == nil {
		rep = jobs.NopReporter
	}
	if err := soundcloud.NormalizeIncoming(ctx, env.MusicRoot); err != nil {
		return err
	}
	return library.RunImportCtx(ctx, library.EnvFrom(env), rep)
}
