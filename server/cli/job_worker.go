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
		return fmt.Errorf("usage: evoplayer _job <soundcloud-download|import-incoming|download-url|cache> [args]")
	}
	switch args[0] {
	case "soundcloud-download":
		return runSoundCloudWorker(env, args[1:])
	case "import-incoming":
		return runImportWorker(env)
	case "download-url":
		return runDownloadURLWorker(env, args[1:])
	case "cache":
		return runCacheWorker(env, args[1:])
	default:
		return fmt.Errorf("evoplayer: unknown job %q", args[0])
	}
}

func runImportWorker(env paths.Env) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(worker.RunImportIncoming(ctx, env))
	return nil
}

func runSoundCloudWorker(env paths.Env, args []string) error {
	importAfter := hasFlag(args, "--import")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(worker.RunSoundCloudDownload(ctx, env, importAfter))
	return nil
}

func runDownloadURLWorker(env paths.Env, args []string) error {
	importAfter := hasFlag(args, "--import")
	rawURL := firstNonFlagArg(args)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(worker.RunDownloadURL(ctx, env, rawURL, importAfter))
	return nil
}

func runCacheWorker(env paths.Env, args []string) error {
	force := hasFlag(args, "--force")
	genre := flagValue(args, "--genre")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(worker.RunCache(ctx, env, genre, force))
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
