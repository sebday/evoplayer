package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/worker"
)

func CmdJobWorker(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer _job <soundcloud-download|import-incoming|download-url|cache|discover-preview|discover-keep> [args]")
	}
	rest := args[1:]
	var run func(ctx context.Context) int
	switch args[0] {
	case "soundcloud-download":
		run = func(ctx context.Context) int {
			return worker.RunSoundCloudDownload(ctx, env, hasFlag(rest, "--import"))
		}
	case "import-incoming":
		run = func(ctx context.Context) int { return worker.RunImportIncoming(ctx, env) }
	case "download-url":
		run = func(ctx context.Context) int {
			return worker.RunDownloadURL(ctx, env, firstNonFlagArg(rest), hasFlag(rest, "--import"))
		}
	case "cache":
		run = func(ctx context.Context) int {
			return worker.RunCache(ctx, env, flagValue(rest, "--genre"), hasFlag(rest, "--force"))
		}
	case "discover-preview":
		run = func(ctx context.Context) int { return worker.RunDiscoverPreview(ctx, env, worker.ParseTrackID(rest)) }
	case "discover-keep":
		run = func(ctx context.Context) int { return worker.RunDiscoverKeep(ctx, env, worker.ParseTrackID(rest)) }
	default:
		return fmt.Errorf("evoplayer: unknown job %q", args[0])
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx)
	stop()
	os.Exit(code)
	return nil
}

func firstNonFlagArg(args []string) string {
	skipNext := false
	for _, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if a == "--genre" {
			skipNext = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}

func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
