package cli

import (
	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/paths"
)

func CmdStats(env paths.Env, args []string) error {
	jsonOut := hasFlag(args, "--json")
	stats, err := library.LibraryStats(library.EnvFrom(env))
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(stats)
	}
	return printJSON(stats)
}
