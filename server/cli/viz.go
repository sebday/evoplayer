package cli

import (
	"fmt"
	"strconv"

	"github.com/sebday/evoplayer/server/paths"
)

func CmdViz(env paths.Env, exe string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer viz <apply|stream|get>")
	}
	switch args[0] {
	case "apply":
		return cmdVizApply(env, exe)
	case "stream":
		fps := 30
		if len(args) >= 3 && args[1] == "--fps" {
			if n, err := strconv.Atoi(args[2]); err == nil {
				fps = n
			}
		}
		return cmdVizStream(env, exe, fps)
	case "get":
		return cmdSpectrumGet(env, exe)
	default:
		return fmt.Errorf("evoplayer: unknown viz subcommand: %s", args[0])
	}
}

func cmdVizApply(env paths.Env, exe string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	resp, err := IPC(env, "viz.config.apply", nil)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}
