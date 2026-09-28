package worker

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/soundcloud"
)

// RunDiscoverPreview downloads one SoundCloud track into the discover cache.
func RunDiscoverPreview(ctx context.Context, env paths.Env, id int64) int {
	DeprioritizeProcess()
	rep := NewNDJSONReporter(os.Stdout)
	if id == 0 {
		_ = rep.Error("id required")
		return 1
	}
	if _, err := soundcloud.DownloadPreview(ctx, env, id, rep); err != nil {
		_ = rep.Error(err.Error())
		return 1
	}
	return 0
}

// RunDiscoverKeep downloads or moves a discover track into .incoming.
func RunDiscoverKeep(ctx context.Context, env paths.Env, id int64) int {
	DeprioritizeProcess()
	rep := NewNDJSONReporter(os.Stdout)
	if id == 0 {
		_ = rep.Error("id required")
		return 1
	}
	if _, err := soundcloud.Keep(ctx, env, id, rep); err != nil {
		_ = rep.Error(err.Error())
		return 1
	}
	return 0
}

// ParseTrackID reads the first numeric argument.
func ParseTrackID(args []string) int64 {
	raw := ""
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		raw = strings.TrimSpace(a)
		break
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
