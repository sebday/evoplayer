package cli

import (
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

func CmdStats(env paths.Env, args []string) error {
	stats, err := library.LibraryStats(library.EnvFrom(env))
	if err != nil {
		return err
	}
	return printJSON(stats)
}
