package worker

import (
	"context"
	"os"

	"github.com/sebday/evoplayer/server/download"
	"github.com/sebday/evoplayer/server/paths"
)

// RunImportIncoming runs .incoming import in the current process.
func RunImportIncoming(ctx context.Context, env paths.Env) int {
	DeprioritizeProcess()
	rep := NewNDJSONReporter(os.Stdout)
	if err := download.ImportLibraryIncoming(ctx, env, rep); err != nil {
		_ = rep.Error(err.Error())
		return 1
	}
	return 0
}
